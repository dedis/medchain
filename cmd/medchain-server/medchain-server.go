package main

import (
	// Empty imports to have the init-functions called which should
	// register the protocol
	_ "github.com/medchain/protocols"
	_ "github.com/medchain/services"
	"go.dedis.ch/onet/v3/app"
	cli "gopkg.in/urfave/cli.v1"
)

func runServer(ctx *cli.Context) error {
	// first check the options
	config := ctx.String("config")
	app.RunServer(config)
	return nil
}
