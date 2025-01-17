package main

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/golang-migrate/migrate/v4"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
	"github.com/urfave/cli/v2/altsrc"

	"github.com/lncapital/torq/build"
	"github.com/lncapital/torq/cmd/torq/internal/subscribe"
	"github.com/lncapital/torq/cmd/torq/internal/torqsrv"
	"github.com/lncapital/torq/internal/channels"
	"github.com/lncapital/torq/internal/database"
	"github.com/lncapital/torq/internal/settings"
	"github.com/lncapital/torq/pkg/broadcast"
	"github.com/lncapital/torq/pkg/commons"
	"github.com/lncapital/torq/pkg/lnd_connect"
)

var eventChannel = make(chan interface{}) //nolint:gochecknoglobals

var startchan = make(chan struct{}) //nolint:gochecknoglobals
var stopchan = make(chan struct{})  //nolint:gochecknoglobals

type subscriptions struct {
	mu          sync.RWMutex
	runningList map[int]func()
}

func (rs *subscriptions) AddSubscription(nodeId int, cancelFunc func()) {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	if rs.runningList == nil {
		rs.runningList = make(map[int]func())
	}

	rs.runningList[nodeId] = cancelFunc
}

func (rs *subscriptions) RemoveSubscription(nodeId int) {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	if rs.runningList == nil {
		rs.runningList = make(map[int]func())
	}
	delete(rs.runningList, nodeId)
}

func (rs *subscriptions) Contains(nodeId int) bool {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	_, exists := rs.runningList[nodeId]
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

var runningSubscriptions subscriptions //nolint:gochecknoglobals

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

		&cli.BoolFlag{
			Name:  "debug",
			Value: false,
			Usage: "Enable debug logging",
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

		// LND connection details
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:  "lnd.url",
			Usage: "Host:Port of the LND node",
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:  "lnd.macaroon-path",
			Usage: "Path on disk to LND Macaroon",
		}),
		altsrc.NewStringFlag(&cli.StringFlag{
			Name:  "lnd.tls-path",
			Usage: "Path on disk to LND TLS file",
		}),
	}

	start := &cli.Command{
		Name:  "start",
		Usage: "Start the main daemon",
		Action: func(c *cli.Context) error {

			zerolog.SetGlobalLevel(zerolog.InfoLevel)
			if c.Bool("debug") {
				zerolog.SetGlobalLevel(zerolog.DebugLevel)
				log.Debug().Msg("Debug logging enabled")
			}

			// Print startup message
			fmt.Printf("Starting Torq %s\n", build.Version())

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

			go commons.ManagedSettingsCache(commons.ManagedSettingsChannel, nil)
			err = settings.InitializeManagedSettingsCache(db)
			if err != nil {
				log.Error().Err(err).Msg("Failed to obtain settings for ManagedSettings cache.")
			}

			go commons.ManagedNodeCache(commons.ManagedNodeChannel, nil)
			err = settings.InitializeManagedNodeCache(db)
			if err != nil {
				log.Error().Err(err).Msg("Failed to obtain torq nodes for ManagedNode cache.")
			}

			go commons.ManagedChannelCache(commons.ManagedChannelChannel, nil)
			err = channels.InitializeManagedChannelCache(db)
			if err != nil {
				log.Error().Err(err).Msg("Failed to obtain channels for ManagedChannel cache.")
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			broadcaster := broadcast.NewBroadcastServer(ctx, eventChannel)

			// if node specified on cmd flags then check if we already know about it
			if c.String("lnd.url") != "" && c.String("lnd.macaroon-path") != "" && c.String("lnd.tls-path") != "" {
				macaroonFile, err := os.ReadFile(c.String("lnd.macaroon-path"))
				if err != nil {
					log.Error().Err(err).Msg("Reading macaroon file from disk path from config")
					return errors.Wrap(err, "Reading macaroon file from disk path from config")
				}
				tlsFile, err := os.ReadFile(c.String("lnd.tls-path"))
				if err != nil {
					log.Error().Err(err).Msg("Reading tls file from disk path from config")
					return errors.Wrap(err, "Reading tls file from disk path from config")
				}
				grpcAddress := c.String("lnd.url")
				nodeId, err := settings.GetNodeIdByGRPC(db, grpcAddress)
				if err != nil {
					log.Error().Err(err).Msg("Checking if node specified in config exists")
					return errors.Wrap(err, "Checking if node specified in config exists")
				}
				if nodeId == 0 {
					log.Debug().Msg("Node specified in config is not in DB, adding it")
					nodeConnectionDetails, err := settings.AddNodeToDB(db, commons.LND, grpcAddress, tlsFile, macaroonFile)
					if err != nil {
						log.Error().Err(err).Msg("Adding node specified in config to database")
						return errors.Wrap(err, "Adding node specified in config to database")
					}
					nodeConnectionDetails.Name = "Auto configured node"
					_, err = settings.SetNodeConnectionDetails(db, nodeConnectionDetails)
					if err != nil {
						return errors.Wrap(err, "Updating node name")
					}
				} else {
					log.Debug().Msg("Node specified in config is present, updating Macaroon and TLS files")
					if err = settings.SetNodeConnectionDetailsByConnectionDetails(db, nodeId, commons.Active, grpcAddress, tlsFile, macaroonFile); err != nil {
						log.Error().Err(err).Msg("Problem updating node files")
						return errors.Wrap(err, "Problem updating node files")
					}
				}
			}

			if !c.Bool("torq.no-sub") {
				// initialise package level var for keeping state of subsciptions
				runningSubscriptions = subscriptions{}

				// go routine that responds to start command and starts all subscriptions
				go (func() {
					for {
						<-startchan
						nodes, err := settings.GetActiveNodesConnectionDetails(db)
						if err != nil {
							log.Error().Err(errors.Wrap(err, "Getting connection details")).Send()
							return
						}

						for _, node := range nodes {
							go (func(node settings.ConnectionDetails) {

								ctx := context.Background()
								ctx, cancel := context.WithCancel(ctx)

								log.Info().Msgf("Subscribing to LND for node id: %v", node.NodeId)
								runningSubscriptions.AddSubscription(node.NodeId, cancel)
								conn, err := lnd_connect.Connect(
									node.GRPCAddress,
									node.TLSFileBytes,
									node.MacaroonFileBytes,
								)
								if err != nil {
									log.Error().Err(err).Msgf("Failed to connect to lnd for node id: %v", node.NodeId)
									runningSubscriptions.RemoveSubscription(node.NodeId)
									return
								}

								err = subscribe.Start(ctx, conn, db, node.NodeId, eventChannel)
								if err != nil {
									log.Error().Err(err).Send()
									// only log the error, don't return
								}
								log.Info().Msgf("LND Subscription stopped for node id: %v", node.NodeId)
								runningSubscriptions.RemoveSubscription(node.NodeId)
							})(node)
						}
					}

				})()

				// starts LND subscription when Torq starts
				startchan <- struct{}{}

				// go routine that looks for stop signals and cancels the context(s)
				go (func() {
					for {
						<-stopchan
						for _, cancelFunc := range runningSubscriptions.GetCancelFuncs() {
							cancelFunc()
						}
					}
				})()

			}

			if err = torqsrv.Start(c.Int("torq.port"), c.String("torq.password"), db, eventChannel, broadcaster, RestartLNDSubscription); err != nil {
				return errors.Wrap(err, "Starting torq webserver")
			}

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

// guards against running restart code whilst it's already running
var restartLock sync.RWMutex //nolint:gochecknoglobals

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
