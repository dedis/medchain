package main

import (
	// Empty imports to have the init-functions called which should
	// register the protocol
	"errors"
	"os"
	"time"

	_ "github.com/medchain/protocols"
	_ "github.com/medchain/services"
	"github.com/urfave/cli"
	status "go.dedis.ch/cothority/v3/status/service"
	"go.dedis.ch/onet/v3/app"
	"go.dedis.ch/onet/v3/log"
	"go.dedis.ch/onet/v3/network"
)

var raiseFdLimit func()

func runServer(ctx *cli.Context) error {
	// first check the options
	config := ctx.GlobalString("config")
	if raiseFdLimit != nil {
		raiseFdLimit()
	}
	app.RunServer(config)
	return nil
}

func checkConfig(c *cli.Context) error {
	tomlFileName := c.String("g")
	if c.NArg() > 0 {
		tomlFileName = c.Args().First()
	}
	if tomlFileName == "" {
		log.Fatal("[-] Must give the roster file to check.")
	}

	f, err := os.Open(tomlFileName)
	if err != nil {
		return err
	}

	client := status.NewClient()

	grp, err := app.ReadGroupDescToml(f)
	if err != nil {
		return err
	}

	ro := grp.Roster
	replies := make(chan *status.Response)
	errs := make(chan error)

	// send a status request to everyone
	for _, si := range ro.List {
		go func(srvid *network.ServerIdentity) {
			reply, err := client.Request(srvid)
			if err != nil {
				errs <- err
			} else {
				replies <- reply
			}
		}(si)
	}

	counter := 0
	timeout := time.After(time.Duration(c.Int("timeout")) * time.Second)

	// ... and wait for the responses
	for counter < len(ro.List) {
		select {
		case <-replies:
			counter++
		case err := <-errs:
			return err
		case <-timeout:
			return errors.New("didn't get all the responses in time")
		}
	}

	return nil
}
