// +build routerrpc

package main

import (
	"context"
	"github.com/lightningnetwork/lnd/lnrpc/api/router"

	"github.com/urfave/cli"
)

var queryMissionControlCommand = cli.Command{
	Name:     "querymc",
	Category: "Payments",
	Usage:    "Query the internal mission control state.",
	Action:   actionDecorator(queryMissionControl),
}

func queryMissionControl(ctx *cli.Context) error {
	conn := getClientConn(ctx, false)
	defer conn.Close()

	client := router.NewRouterClient(conn)

	req := &router.QueryMissionControlRequest{}
	rpcCtx := context.Background()
	snapshot, err := client.QueryMissionControl(rpcCtx, req)
	if err != nil {
		return err
	}

	printRespJSON(snapshot)

	return nil
}
