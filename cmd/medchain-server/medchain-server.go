package main

import (
	"encoding/base64"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"

	s "github.com/medchain/services"
	"github.com/urfave/cli"
	"go.dedis.ch/cothority/v3/byzcoin"
	"go.dedis.ch/cothority/v3/byzcoin/bcadmin/lib"
	"go.dedis.ch/onet/v3"
	"go.dedis.ch/onet/v3/app"
	"go.dedis.ch/onet/v3/log"
	"go.dedis.ch/onet/v3/network"
	"go.dedis.ch/protobuf"
	"golang.org/x/xerrors"
)

func submitQuery(c *cli.Context) error {
	// Here is what this function does:
	//   1. Parses the stdin in order to get the proposed query
	//   2. Fires a spawn instruction for the deferred contract
	//	 3. Gets the response back from MedChain service
	//	 4. Broadcasts instanceID to all MedChain nodes
	//   5. Writes query instanceID to file

	// ---
	// 1. Parse the stdin in order to get the proposed query
	// ---
	log.Lvl1("[INFO] Starting query submission")
	log.Lvl1("[INFO] Reading query from stdin") //TODO: Read the  query from other sources?

	proposedQueryBuf, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		return xerrors.Errorf("failed to read from stdin: %v", err)
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

	dstr := c.String("darcid") //TODO: or read darc from file? admin?
	if dstr == "" {
		dstr = cfg.AdminDarc.GetIdentityString()
	}

	projectDarc, err := lib.GetDarcByString(bcl, dstr)
	if err != nil {
		return err
	}
	clientID := c.String("clientid")
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

	var list []*network.ServerIdentity
	var si *network.ServerIdentity

	address := c.String("address")
	if address != "" {
		// Contact desired server
		log.Info("[INFO] contacting server at", address)
		addr := network.Address(address)
		if !strings.HasPrefix(address, "tls://") {
			addr = network.NewAddress(network.TLS, address)
		}
		si := network.NewServerIdentity(nil, addr)
		if si.Address.Port() == "" {
			return errors.New("port not found, must provide addr:port")
		}
		list = append(list, si)
	} else {

		roster, err := readGroup(groupTomlPath)
		if err != nil {
			return errors.New("couldn't read file: " + err.Error())
		}
		list = roster.List
		log.Info("[INFO] Roster list is", list)
	}

	log.Info("[INFO] Sending request to", si)

	name := projectDarc.Description

	client, err := s.NewClient(bcl, si, clientID)
	if err != nil {
		return xerrors.Errorf("failed to init client: %v", err)
	}

	err = client.Create()
	if err != nil {
		return xerrors.Errorf("failed to create client: %v", err)
	}

	client.AllDarcIDs[string(name)] = projectDarc.GetBaseID()
	// client.DarcID = projectDarc.GetBaseID()
	req := &s.AddDeferredQueryRequest{}
	req.QueryID = proposedQuery.ID

	// ---
	// 3. Get the response back from MedChain service
	// 4. Broadcast instanceID to all MedChain nodes
	// ---
	reply, err := client.SpawnDeferredQuery(req)
	if err != nil {
		return xerrors.Errorf("failed to spawn query instance: %v", err)
	}

	if reply.OK != true {
		return xerrors.Errorf("service failed to spawn query instance: %v", err)
	}
	client.Bcl.WaitPropagation(1)

	// ---
	// 5.  Write query instance ID to file
	// ---
	instIDfilePath := c.String("idfile")
	if instIDfilePath == "" {
		err := fmt.Errorf("arguments not OK")
		log.Error(err)
		return cli.NewExitError(err, 3)
	}
	// TODO: write query ID and Instance ID
	dir, _ := path.Split(instIDfilePath)
	pathToWrite := dir + instIDfilePath
	fWrite, err := os.Create(pathToWrite)
	if err != nil {
		return err
	}
	defer fWrite.Close()

	_, err = fWrite.WriteString(base64.URLEncoding.EncodeToString([]byte(req.QueryInstID.String())))
	if err != nil {
		return err
	}

	return nil
}

func addSignatureToDeferredQuery(c *cli.Context) error {
	// Here is what this function does:
	//   1. Read instanceID of query to be signed from file from flag
	//   2. Fires a spawn instruction for the deferred contract
	//	 3. Gets the response back from MedChain service
	//   4. Write query instanceID to file

	// ---
	// 1. Read instanceID of query to be signed from file from flag
	// ---
	log.Lvl1("[INFO] Starting adding signature to deferred query")

	// var iid byzcoin.InstanceID
	iidStr := c.String("iid")
	if iidStr == "" {
		err := fmt.Errorf("arguments not OK")
		return cli.NewExitError(err, 3)
	}
	iid := byzcoin.NewInstanceID([]byte(iidStr)) //TODO: double-check this is the same instanceID

	bcArg := c.String("bc")
	if bcArg == "" {
		err := fmt.Errorf("arguments not OK")
		return cli.NewExitError(err, 3)
	}

	cfg, bcl, err := lib.LoadConfig(bcArg)
	if err != nil {
		return err
	}

	dstr := c.String("darc") //TODO: Read darc from file? admin?
	if dstr == "" {
		dstr = cfg.AdminDarc.GetIdentityString()
	}

	projectDarc, err := lib.GetDarcByString(bcl, dstr)
	if err != nil {
		return err
	}
	clientID := c.String("clientid") //TODO: Read ClientID from other sources?
	if clientID == "" {
		err := fmt.Errorf("arguments not OK")
		log.Error(err)
		return cli.NewExitError(err, 3)
	}

	log.Info("[INFO] Reading medchain group definition")

	groupTomlPath := c.String("file")
	if groupTomlPath == "" {
		err := fmt.Errorf("arguments not OK")
		log.Error(err)
		return cli.NewExitError(err, 3)
	}

	var list []*network.ServerIdentity
	var si *network.ServerIdentity

	address := c.String("address")
	if address != "" {
		// Contact desired server
		log.Info("[INFO] contacting server at", address)
		addr := network.Address(address)
		if !strings.HasPrefix(address, "tls://") {
			addr = network.NewAddress(network.TLS, address)
		}
		si := network.NewServerIdentity(nil, addr)
		if si.Address.Port() == "" {
			return errors.New("port not found, must provide addr:port")
		}
		list = append(list, si)
	} else {

		roster, err := readGroup(groupTomlPath)
		if err != nil {
			return errors.New("couldn't read file: " + err.Error())
		}
		list = roster.List
		log.Info("[INFO] Roster list is", list)
	}

	log.Info("[INFO] Sending request to", si.String()) //TODO: exact server address
	name := projectDarc.Description
	client, err := s.NewClient(bcl, si, clientID)
	if err != nil {
		return xerrors.Errorf("failed to init client: %v", err)
	}

	err = client.Create()
	if err != nil {
		return xerrors.Errorf("failed to create client: %v", err)
	}

	client.AllDarcIDs[string(name)] = projectDarc.GetBaseID()
	// client.DarcID = projectDarc.GetBaseID()

	req := &s.SignDeferredTxRequest{}
	req.ClientID = clientID
	req.QueryInstID = iid
	reply, err := client.AddSignatureToDeferredQuery(req)
	if err != nil {
		return xerrors.Errorf("failed to add signature to query instance %w: %v", req.QueryInstID.String(), err)
	}

	// ---
	// 3. Get the response back from MedChain service
	// ---
	if reply.OK != true {
		return xerrors.Errorf("failed to add signature to query instance %w: %v", req.QueryInstID.String(), err)
	}
	client.Bcl.WaitPropagation(1)

	return nil
}

func readGroupArgs(c *cli.Context, pos int) *app.Group {
	if c.NArg() <= pos {
		log.Fatal("Please give the group-file as argument")
	}
	name := c.Args().Get(pos)
	return readAppGroup(name)
}

func readAppGroup(name string) *app.Group {
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

// readGroup takes a toml file name and reads the file, returning the entities
// within.
func readGroup(tomlFileName string) (*onet.Roster, error) {
	f, err := os.Open(tomlFileName)
	if err != nil {
		return nil, err
	}
	g, err := app.ReadGroupDescToml(f)
	if err != nil {
		return nil, err
	}
	if len(g.Roster.List) <= 0 {
		return nil, errors.New("Empty or invalid group file:" +
			tomlFileName)
	}
	log.Lvl3(g.Roster)
	return g.Roster, err
}
