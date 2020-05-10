package main

import (
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"os"
	"path"

	s "github.com/medchain/services"
	"github.com/urfave/cli"
	"go.dedis.ch/cothority/byzcoin/bcadmin/lib"
	"go.dedis.ch/onet/v3/app"
	"go.dedis.ch/onet/v3/log"
	"go.dedis.ch/protobuf"
	"golang.org/x/xerrors"
)

func submitQuery(c *cli.Context) error {
	// Here is what this function does:
	//   1. Parses the stdin in order to get the proposed query
	//   2. Fires a spawn instruction for the deferred contract
	//	 3. Gets the response back from MedChain service
	//   4. Write query instanceID to file

	// ---
	// 1. Parse the stdin in order to get the proposed query
	// ---
	log.Lvl1("[INFO] Starting query submission")
	log.Lvl1("[INFO] Reading query from stdin") //TODO: Read the  query from other sources

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
	// 2. Fire a spawn instruction for the deferred contract
	// ---
	bcArg := c.String("bc")
	if bcArg == "" {
		err := fmt.Errorf("arguments not OK")
		return cli.NewExitError(err, 3)
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
	clientID := c.String("clientid") //TODO: Read ClientID from other sources
	if clientID == "" {
		err := fmt.Errorf("arguments not OK")
		log.Error(err)
		return cli.NewExitError(err, 3)
	}

	log.Lvl1("[INFO] Reading medchain group definition")

	groupTomlPath := c.String("file")
	if groupTomlPath == "" {
		err := fmt.Errorf("arguments not OK")
		log.Error(err)
		return cli.NewExitError(err, 3)
	}

	group := readGroup(groupTomlPath)
	if err != nil {
		return err
	}
	if group == nil {
		return xerrors.Errorf("error while reading group definition file: %v", groupTomlPath)
	}
	roster := group.Roster
	if len(roster.List) <= 0 {
		return xerrors.Errorf("empty or invalid medchain group file: %v", groupTomlPath)
	}
	b, err := group.Roster.Aggregate.MarshalBinary()
	if err != nil {
		return err
	}
	log.Lvl1("[INFO] Sending query to", roster.RandomServerIdentity())
	name := projectDarc.Description
	client, err := s.NewClient(bcl, roster.RandomServerIdentity(), clientID)
	if err != nil {
		return xerrors.Errorf("failed to init client: %v", err)
	}

	err = client.Create()
	if err != nil {
		return xerrors.Errorf("failed to create client: %v", err)
	}

	client.AllDarcIDs[string(name)] = projectDarc.GetBaseID()
	client.DarcID = projectDarc.GetBaseID()
	req := &s.AddDeferredQueryRequest{}
	req.QueryID = proposedQuery.ID
	reply, err := client.SpawnDeferredQuery(req)
	if err != nil {
		return xerrors.Errorf("failed to spawn query instance: %v", err)
	}

	// ---
	// 3. Gets the response back from MedChain service
	// ---
	if reply.OK != true {
		return xerrors.Errorf("failed to spawn query instance from service: %v", err)
	}
	client.Bcl.WaitPropagation(1)

	// ---
	// 4.  Write query instance ID to file
	// ---
	dir, _ := path.Split(groupTomlPath)
	pathToWrite := dir + "instanceIDs.txt"
	fWrite, err := os.Create(pathToWrite)
	if err != nil {
		return err
	}
	defer fWrite.Close()

	_, err = fWrite.WriteString(base64.URLEncoding.EncodeToString(b))
	if err != nil {
		return err
	}

	return nil
}

func readGroupArgs(c *cli.Context, pos int) *app.Group {
	if c.NArg() <= pos {
		log.Fatal("Please give the group-file as argument")
	}
	name := c.Args().Get(pos)
	return readGroup(name)
}
func readGroup(name string) *app.Group {
	f, err := os.Open(name)
	log.ErrFatal(err, "Couldn't open group definition file")
	group, err := app.ReadGroupDescToml(f)
	log.ErrFatal(err, "Error while reading group definition file", err)
	if len(group.Roster.List) == 0 {
		log.ErrFatalf(err, "Empty entity or invalid group defintion in: %s",
			name)
	}
	return group
}
