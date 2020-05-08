package main

import (
	"io/ioutil"
	"os"

	s "github.com/medchain/services"
	"github.com/urfave/cli"
	"go.dedis.ch/cothority/byzcoin/bcadmin/lib"
	"go.dedis.ch/protobuf"
	"golang.org/x/xerrors"
)

func submitQuery(c *cli.Context) error {
	// Here is what this function does:
	//   1. Parses the stdin in order to get the proposed query
	//   2. Fires a spawn instruction for the deferred contract
	//	 3. Write query instanceID to file
	//   4. Gets the response back

	// ---
	// 1.
	// ---
	proposedQueryBuf, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		return xerrors.Errorf("failed to read from stding: %v", err)
	}

	proposedQuery := s.Query{}
	err = protobuf.Decode(proposedQueryBuf, &proposedQuery)
	if err != nil {
		return xerrors.Errorf("failed to decode query, did you use --export ?: %v", err)
	}

	// ---
	// 2.
	// ---
	bcArg := c.String("bc")
	if bcArg == "" {
		return xerrors.New("--bc flag is required")
	}

	cfg, bcl, err := lib.LoadConfig(bcArg)
	if err != nil {
		return err
	}

	dstr := c.String("darc")
	if dstr == "" {
		dstr = cfg.AdminDarc.GetIdentityString()
	}

	projectDarc, err := lib.GetDarcByString(bcl, dstr)
	if err != nil {
		return err
	}

	name := projectDarc.Description
	client, err := s.NewClient(bcl)
	if err != nil {
		return xerrors.Errorf("failed to init client: %v", err)
	}

	err = client.Create()
	if err != nil {
		return xerrors.Errorf("failed to create client: %v", err)
	}

	client.AllDarcIDs[string(name)] = projectDarc.GetBaseID()
	client.DarcID = projectDarc.GetBaseID()
	instID, err := client.SpawnDeferredQuery(proposedQuery)
	if err != nil {
		return xerrors.Errorf("failed to spawn query instance: %v", err)
	}
	client.Bcl.WaitPropagation(1)
	//TODO: write instID to file

	return nil
}
