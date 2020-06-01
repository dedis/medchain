package main

import (
	"errors"
	"io/ioutil"
	"math/rand"
	"os"
	"time"

	medchain "github.com/medchain/services"
	cli "github.com/urfave/cli"
	"go.dedis.ch/cothority/v3"
	"go.dedis.ch/cothority/v3/byzcoin"
	bcadminlib "go.dedis.ch/cothority/v3/byzcoin/bcadmin/lib"
	"go.dedis.ch/cothority/v3/darc"

	"go.dedis.ch/onet/v3/cfgpath"
	"go.dedis.ch/onet/v3/log"
	"go.dedis.ch/onet/v3/network"
	"go.dedis.ch/protobuf"
)

var cliApp = cli.NewApp()
var dataDir = ""
var gitTag = "dev"

func init() {
	cliApp.Name = "mc"
	cliApp.Usage = "Create and work with MedChain."
	cliApp.Version = gitTag
	cliApp.Commands = cmds
	cliApp.Flags = []cli.Flag{
		cli.IntFlag{
			Name:  "debug, d",
			Value: 0,
			Usage: "debug-level: 1 for terse, 5 for maximal",
		},
		cli.StringFlag{
			Name:  "config, c",
			Value: cfgpath.GetDataPath(cliApp.Name),
			Usage: "path to configuration-directory",
		},
	}
	cliApp.Before = func(c *cli.Context) error {
		log.SetDebugVisible(c.Int("debug"))
		bcadminlib.ConfigPath = c.String("config")
		return nil
	}

}

func main() {
	rand.Seed(time.Now().UTC().UnixNano())
	log.ErrFatal(cliApp.Run(os.Args))
}

// getClient will create a new medchain.Client, given the input
// available in the commandline. If priv is false, then it will not
// look for a private key and set up the signers. (This is used for
// searching, which does not require having a private key available
// because it does not submit transactions.)
func getClient(c *cli.Context, priv bool) (*medchain.Client, error) {
	bc := c.String("bc")
	if bc == "" {
		return nil, errors.New("--bc flag is required")
	}

	cfgBuf, err := ioutil.ReadFile(bc)
	if err != nil {
		return nil, err
	}
	var cfg bcConfig
	err = protobuf.DecodeWithConstructors(cfgBuf, &cfg,
		network.DefaultConstructors(cothority.Suite))
	if err != nil {
		return nil, err
	}

	cl := medchain.NewClient(byzcoin.NewClient(cfg.ByzCoinID, cfg.Roster))

	d, err := cl.ByzCoin.GetGenDarc()
	if err != nil {
		return nil, err
	}
	cl.DarcID = d.GetBaseID()

	// Initialize project Darcs hash map
	cl.AllDarcs = make(map[string]*darc.Darc)
	cl.AllDarcIDs = make(map[string]darc.ID)

	// The caller doesn't want/need signers.
	if !priv {
		return cl, nil
	}

	// get the private key from the cmdline.
	sstr := c.String("sign")
	if sstr == "" {
		return nil, errors.New("--sign is required")
	}
	signer, err := bcadminlib.LoadKeyFromString(sstr)
	if err != nil {
		return nil, err
	}
	cl.Signers = []darc.Signer{*signer}

	return cl, nil
}
