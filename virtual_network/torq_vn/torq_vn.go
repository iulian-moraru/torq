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

	// Add command for
	// alice() { docker exec -it  dev-alice /bin/bash -c "lncli --macaroonpath=\"/root/.lnd/data/chain/bitcoin/simnet/admin.macaroon\" --network=simnet $@"};

	app.Commands = cli.Commands{
		createNetwork,
		startNetwork,
		stopNetwork,
		purgeNetwork,
	}

	err := app.Run(os.Args)
	if err != nil {
		log.Fatal(err)
	}

}
