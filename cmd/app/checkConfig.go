package main

import (
	"github.com/urfave/cli"
	"go.dedis.ch/cothority/cosi/check"
)

// checkConfig contacts all servers and verifies if it receives a valid
// signature from each.
func checkConfig(c *cli.Context) error {
	tomlFileName := c.String(optionGroupFile)
	return check.Config(tomlFileName, c.Bool("detail"))
}
