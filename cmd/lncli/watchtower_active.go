// +build watchtowerrpc

package main

import (
	"context"
	"github.com/lightningnetwork/lnd/lnrpc/api/watchtower"

	"github.com/urfave/cli"
)

func watchtowerCommands() []cli.Command {
	return []cli.Command{
		{
			Name:     "tower",
			Usage:    "Interact with the watchtower.",
			Category: "Watchtower",
			Subcommands: []cli.Command{
				towerInfoCommand,
			},
		},
	}
}

func getWatchtowerClient(ctx *cli.Context) (watchtower.WatchtowerClient, func()) {
	conn := getClientConn(ctx, false)
	cleanup := func() {
		conn.Close()
	}
	return watchtower.NewWatchtowerClient(conn), cleanup
}

var towerInfoCommand = cli.Command{
	Name:   "info",
	Usage:  "Returns basic information related to the active watchtower.",
	Action: actionDecorator(towerInfo),
}

func towerInfo(ctx *cli.Context) error {
	if ctx.NArg() != 0 || ctx.NumFlags() > 0 {
		return cli.ShowCommandHelp(ctx, "info")
	}

	client, cleanup := getWatchtowerClient(ctx)
	defer cleanup()

	req := &watchtower.GetInfoRequest{}
	resp, err := client.GetInfo(context.Background(), req)
	if err != nil {
		return err
	}

	printRespJSON(resp)

	return nil
}
