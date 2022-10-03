package main

import (
	"github.com/lncapital/torq/virtual_network"
	"github.com/urfave/cli/v2"
	"log"
	"os"
)

func main() {
	app := cli.NewApp()
	app.Name = "torq-virtual-network"
	app.EnableBashCompletion = true

	startNetwork := &cli.Command{
		Name:  "start",
		Usage: "Start the virtual network.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "network_name",
				Value:   "dev",
				Aliases: []string{"n"},
				Usage:   "The name of the virtual network (used to name the containers)",
			},
			&cli.BoolFlag{
				Name:  "db",
				Value: true,
				Usage: "Start the database as well",
			},
		},
		Action: func(c *cli.Context) error {
			err := virtual_network.StartVirtualNetwork(c.String("network_name"), c.Bool("db"))
			if err != nil {
				log.Fatal(err)
			}
			return nil
		},
	}

	stopNetwork := &cli.Command{
		Name:  "stop",
		Usage: "Stops the virtual network.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "network_name",
				Value:   "dev",
				Aliases: []string{"n"},
				Usage:   "The name of the virtual network (used to name the containers)",
			},
			&cli.BoolFlag{
				Name:  "db",
				Value: true,
				Usage: "Stops the database as well",
			},
		},
		Action: func(c *cli.Context) error {
			err := virtual_network.StopVirtualNetwork(c.String("network_name"), c.Bool("db"))
			if err != nil {
				log.Fatal(err)
			}
			return nil
		},
	}

	purgeNetwork := &cli.Command{
		Name:  "purge",
		Usage: "Purges the virtual network.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "network_name",
				Value:   "dev",
				Aliases: []string{"n"},
				Usage:   "The name of the virtual network (used to name the containers)",
			},
			&cli.BoolFlag{
				Name:  "db",
				Value: true,
				Usage: "Stops the database as well",
			},
		},
		Action: func(c *cli.Context) error {
			err := virtual_network.PurgeVirtualNetwork(c.String("network_name"), c.Bool("db"))
			if err != nil {
				log.Fatal(err)
			}
			return nil
		},
	}

	createNetwork := &cli.Command{
		Name:  "create",
		Usage: "Create the virtual network.",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "network_name",
				Value:   "dev",
				Aliases: []string{"n"},
				Usage:   "The name of the virtual network (used to name the containers)",
			},
			&cli.BoolFlag{
				Name:  "db",
				Value: true,
				Usage: "Start the database as well",
			},
			&cli.BoolFlag{
				Name:  "purge",
				Value: true,
				Usage: "Purge the old network. NB! Including the database.",
			},
		},
		Action: func(c *cli.Context) error {
			err := virtual_network.CreateNewVirtualNetwork(c.String("network_name"), c.Bool("db"), c.Bool("purge"))
			if err != nil {
				log.Fatal(err)
			}
			return nil
		},
	}

	//invfrq flag = invoice creation frequency - default to 1 time per second
	//sendc flag = create address and sencoins frequency - default to 1 time per 30 seconds
	//opchan flag = open channel frequency - default to 1 time per 10 minutes
	nodeFlow := &cli.Command{
		Name:  "flow",
		Usage: "Loop nodes activities",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "network_name",
				Value:   "dev",
				Aliases: []string{"n"},
				Usage:   "The name of the virtual network (used to name the containers)",
			},
			&cli.IntFlag{
				Name:        "virtual_network_invoice_freq",
				Usage:       "Set invoice creation frequency - seconds",
				DefaultText: "1",
			},
			&cli.IntFlag{
				Name:        "virtual_network_send_coins_freq",
				Usage:       "Create address and send coins frequency - seconds",
				DefaultText: "30",
			},
			&cli.IntFlag{
				Name:        "virtual_network_open_close_chan_freq",
				Usage:       "Open channel between random nodes frequency - minutes",
				DefaultText: "10",
			},
			//&cli.BoolFlag{
			//	Name:  "clschan",
			//	Value: true,
			//	Usage: "Close channels randomly",
			//},
		},
		Action: func(c *cli.Context) error {
			err := virtual_network.NodeFLowLoop(
				c.String("network_name"),
				c.Int("virtual_network_invoice_freq"),
				c.Int("virtual_network_send_coins_freq"),
				c.Int("virtual_network_open_close_chan_freq"),
			)
			if err != nil {
				log.Fatal(err)
			}
			return nil
		},
	}

	// Add command for
	// alice() { docker exec -it  dev-alice /bin/bash -c "lncli --macaroonpath=\"/root/.lnd/data/chain/bitcoin/simnet/admin.macaroon\" --network=simnet $@"};

	app.Commands = cli.Commands{
		createNetwork,
		startNetwork,
		stopNetwork,
		purgeNetwork,
		nodeFlow,
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}

}
