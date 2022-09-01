package main

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/golang-migrate/migrate/v4"
	"github.com/lncapital/torq/build"
	"github.com/lncapital/torq/cmd/torq/internal/subscribe"
	"github.com/lncapital/torq/cmd/torq/internal/torqsrv"
	"github.com/lncapital/torq/internal/database"
	"github.com/lncapital/torq/internal/settings"
	"github.com/lncapital/torq/pkg/lnd_connect"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
	"github.com/urfave/cli/v2/altsrc"
)

var startchan = make(chan struct{})
var stopchan = make(chan struct{})

type subscriptions struct {
	mu          sync.RWMutex
	runningList map[int]func()
}

func (rs *subscriptions) AddSubscription(localNodeId int, cancelFunc func()) {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	if rs.runningList == nil {
		rs.runningList = make(map[int]func())
	}

	rs.runningList[localNodeId] = cancelFunc
}

func (rs *subscriptions) RemoveSubscription(localNodeId int) {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	if rs.runningList == nil {
		rs.runningList = make(map[int]func())
	}
	delete(rs.runningList, localNodeId)
}

func (rs *subscriptions) Contains(localNodeId int) bool {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	_, exists := rs.runningList[localNodeId]
	return exists
}

func (rs *subscriptions) GetCancelFuncs() (funcs []func()) {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	for _, v := range rs.runningList {
		funcs = append(funcs, v)
	}

	return funcs
}

var runningSubscriptions subscriptions

func main() {

	app := cli.NewApp()
	app.Name = "torq"
	app.EnableBashCompletion = true
	app.Version = build.Version()

	homedir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal().Msgf("error finding home directory of user: %v", err)
	}

	cmdFlags := []cli.Flag{

		// All these flags can be set though a common config file.
		&cli.StringFlag{
			Name:    "config",
			Value:   homedir + "/.torq/torq.conf",
			Aliases: []string{"c"},
			Usage:   "Path to config file",
		},

		// Torq connection details
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:  "torq.password",
			Usage: "Password used to access the API and frontend.",
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:  "torq.port",
			Value: "8080",
			Usage: "Port to serve the HTTP API",
		}),
		altsrc.NewBoolFlag(&cli.BoolFlag{
			Name:  "torq.no-sub",
			Value: false,
			Usage: "Start the server without subscribing to node data.",
		}),

		// Torq database
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:  "db.name",
			Value: "torq",
			Usage: "Name of the database",
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:  "db.port",
			Value: "5432",
			Usage: "port of the database",
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:  "db.host",
			Value: "localhost",
			Usage: "host of the database",
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:  "db.user",
			Value: "torq",
			Usage: "Name of the postgres user with access to the database",
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:  "db.password",
			Value: "password",
			Usage: "Name of the postgres user with access to the database",
		}),
	}

	start := &cli.Command{
		Name:  "start",
		Usage: "Start the main daemon",
		Action: func(c *cli.Context) error {
			// Print startup message
			fmt.Printf("Starting Torq v%s\n", build.Version())

			fmt.Println("Connecting to the Torq database")
			db, err := database.PgConnect(c.String("db.name"), c.String("db.user"),
				c.String("db.password"), c.String("db.host"), c.String("db.port"))
			if err != nil {
				return errors.Wrap(err, "start cmd")
			}

			defer func() {
				cerr := db.Close()
				if err == nil {
					err = cerr
				}
			}()

			fmt.Println("Checking for migrations..")
			// Check if the database needs to be migrated.
			err = database.MigrateUp(db)
			if err != nil && !errors.Is(err, migrate.ErrNoChange) {
				return err
			}

			if !c.Bool("torq.no-sub") {
				// initialise package level var for keeping state of subsciptions
				runningSubscriptions = subscriptions{}

				// go routine that responds to start command and starts all subscriptions
				go (func() {
					for {
						select {
						case <-startchan:

							nodes, err := settings.GetConnectionDetails(db)
							if err != nil {
								log.Error().Err(errors.Wrap(err, "Getting connection details")).Send()
								return
							}
							for _, node := range nodes {
								go (func(node settings.ConnectionDetails) {

									ctx := context.Background()
									ctx, cancel := context.WithCancel(ctx)

									log.Info().Msgf("Subscribing to LND for node id: %v", node.LocalNodeId)
									runningSubscriptions.AddSubscription(node.LocalNodeId, cancel)
									conn, err := lnd_connect.Connect(
										node.GRPCAddress,
										node.TLSFileBytes,
										node.MacaroonFileBytes)
									if err != nil {
										log.Error().Err(err).Msgf("Failed to connect to lnd for node id: %v", node.LocalNodeId)
										runningSubscriptions.RemoveSubscription(node.LocalNodeId)
										return
									}

									err = subscribe.Start(ctx, conn, db, node.LocalNodeId)
									if err != nil {
										log.Error().Err(err).Send()
										// only log the error, don't return
									}
									log.Info().Msgf("LND Subscription stopped for node id: %v", node.LocalNodeId)
									runningSubscriptions.RemoveSubscription(node.LocalNodeId)
								})(node)
							}
						}
					}
				})()

				// starts LND subscription when Torq starts
				startchan <- struct{}{}

				// go routine that looks for stop signals and cancels the context(s)
				go (func() {
					for {
						select {
						case <-stopchan:
							for _, cancelFunc := range runningSubscriptions.GetCancelFuncs() {
								cancelFunc()
							}
						}
					}
				})()

			}

			torqsrv.Start(c.Int("torq.port"), c.String("torq.password"), db, RestartLNDSubscription)

			return nil
		},
	}

	migrateUp := &cli.Command{
		Name:  "migrate_up",
		Usage: "Migrates the database to the latest version",
		Action: func(c *cli.Context) error {
			db, err := database.PgConnect(c.String("db.name"), c.String("db.user"),
				c.String("db.password"), c.String("db.host"), c.String("db.port"))
			if err != nil {
				return err
			}

			defer func() {
				cerr := db.Close()
				if err == nil {
					err = cerr
				}
			}()

			err = database.MigrateUp(db)
			if err != nil {
				return err
			}

			return nil
		},
	}

	app.Flags = cmdFlags

	app.Before = altsrc.InitInputSourceWithContext(cmdFlags, loadFlags())

	app.Commands = cli.Commands{
		start,
		migrateUp,
	}

	err = app.Run(os.Args)
	if err != nil {
		log.Fatal().Err(err).Send()
	}

}

var restartLock sync.RWMutex

func RestartLNDSubscription() error {
	locked := restartLock.TryLock()
	if !locked {
		return errors.New("Already restarting")
	}
	defer restartLock.Unlock()

	log.Info().Msg("Stopping subscriptions")
	stopchan <- struct{}{}

	for {
		if len(runningSubscriptions.GetCancelFuncs()) == 0 {
			break
		}
		time.Sleep(1 * time.Second)
	}

	log.Info().Msg("All subscriptions stopped")
	log.Info().Msg("Restarting subscriptions")
	startchan <- struct{}{}
	return nil
}

func loadFlags() func(context *cli.Context) (altsrc.InputSourceContext, error) {
	return func(context *cli.Context) (altsrc.InputSourceContext, error) {
		if _, err := os.Stat(context.String("config")); err == nil {
			return altsrc.NewTomlSourceFromFile(context.String("config"))
		}
		return altsrc.NewMapInputSource("", map[interface{}]interface{}{}), nil
	}
}
