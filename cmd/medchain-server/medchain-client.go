package main

import (
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path"
	"strings"

	"io/ioutil"

	s "github.com/medchain/services"
	"go.dedis.ch/cothority/v3"
	"go.dedis.ch/cothority/v3/byzcoin"
	bcadminlib "go.dedis.ch/cothority/v3/byzcoin/bcadmin/lib"
	"go.dedis.ch/cothority/v3/darc"
	"go.dedis.ch/cothority/v3/skipchain"
	"go.dedis.ch/onet/v3"
	"go.dedis.ch/onet/v3/app"
	"go.dedis.ch/onet/v3/log"
	"go.dedis.ch/onet/v3/network"
	"go.dedis.ch/protobuf"
	"golang.org/x/xerrors"
	cli "gopkg.in/urfave/cli.v1"
)

type config struct {
	Name            string
	QueryInstanceID byzcoin.InstanceID
}

type bcConfig struct {
	Roster    onet.Roster
	ByzCoinID skipchain.SkipBlockID
}

func submitQuery(c *cli.Context) error {
	// Here is what this function does:
	//   1. Reads Byzcoin config to get config and Byzcoin client
	//   2. Gets DarcID and rerives it from bzycoin
	//   3. Gets the proposed query
	//   4. Gets cleint ID
	//   5. Fires a spawn instruction for the deferred contract
	//	 6. Gets the response back from MedChain service
	//	 7. Broadcasts instanceID to all MedChain nodes
	//   8. Writes query instanceID to file

	// ---
	// 1. Read Byzcoin config to get config and Byzcoin client
	// ---
	log.Info("[INFO] Starting query submission")
	log.Info("[INFO] Reading byzcoin config file")

	bcArg := c.String("bc")
	if bcArg == "" {
		return xerrors.New("--bc flag is required")
	}
	_, bcl, err := bcadminlib.LoadConfig(bcArg) //TODO: also return cfg
	if err != nil {
		return xerrors.Errorf("failed to read byzcoin config: %v", err)
	}
	// ---
	// 2. Get DarcID and rerives it from bzycoin
	// ---

	// TODO: Broadcast all darc ID after the are created to all nodes, rea them from file
	// This implementation relies on the user to provide the right darc ID for the corresponding
	// project
	log.Info("[INFO] Reading Darc ID")
	darcIDArg := c.String("darcid")
	if darcIDArg == "" {
		return xerrors.New("--darcid flag is required")
	}
	log.Info("[INFO] Getting Darc by ID")
	projectDarc, err := bcadminlib.GetDarcByString(bcl, darcIDArg)
	if err != nil {
		return xerrors.Errorf("failed to get project darc: %v", err)
	}

	// ---
	//  3. Get the proposed query
	// ---
	log.Info("[INFO] Reading the query") //TODO: Read the  query from other sources?
	queryArg := c.String("qid")
	if queryArg == "" {
		return xerrors.New("--qid flag is required")
	}
	log.Info("[INFO] Reading the query")
	proposedQuery := s.NewQuery(queryArg, " ")
	qq := strings.Split(proposedQuery.ID, ":")

	if len(qq) != 3 {
		return xerrors.New("invalid query entered")

	}
	projectName := qq[1]

	// ---
	//  4. Get client ID
	// ---
	log.Info("[INFO] Reading client ID")
	cidArg := c.String("cid")
	if cidArg == "" {
		return xerrors.New("--cid flag is required")
	}

	// ---
	// 5. Fire a spawn instruction for the deferred contract
	// ---

	log.Lvl1("[INFO] Reading medchain group definition")

	groupTomlPath := c.String("file")
	if groupTomlPath == "" {
		return xerrors.New("--file flag is required")
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
	log.Info("[INFO] Roster is ", list)
	log.Info("[INFO] Sending request to", si)

	client, err := s.NewClient(bcl, si, cidArg)
	if err != nil {
		return xerrors.Errorf("failed to init client: %v", err)
	}

	err = client.Create()
	if err != nil {
		return xerrors.Errorf("failed to create client: %v", err)
	}
	client.AllDarcIDs[string(projectName)] = projectDarc
	client.AllDarcIDs[string(projectName)] = projectDarc.GetBaseID()
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

	iIDStr := c.String("iid")
	if iIDStr == "" {

		return xerrors.New("--iid flag is required")
	}
	iIDBuf, err := hex.DecodeString(iIDStr)
	if err != nil {
		return err
	}
	iid := byzcoin.NewInstanceID(iIDBuf)

	bcArg := c.String("bc")
	if bcArg == "" {
		return xerrors.New("--bc flag is required")
	}

	cfg, bcl, err := bcadminlib.LoadConfig(bcArg)
	if err != nil {
		return err
	}

	dstr := c.String("darc") //TODO: Read darc from file? admin?
	if dstr == "" {
		dstr = cfg.AdminDarc.GetIdentityString()
	}

	projectDarc, err := bcadminlib.GetDarcByString(bcl, dstr)
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

	log.Info("[INFO] Sending request to", si.String()) //TODO: exact server address -> done
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

func verifyStatus(c *cli.Context) error {
	// Here is what this function does:
	//   1. Reads instanceID of query to check its status on the chain
	//   2. Retrieves the query from chain
	//	 3. Returns the status of query
	//   4. Writes query instanceID to file

	// ---
	// 1. 1. Reads instanceID of query to check its status on the chain
	// ---
	log.Lvl1("[INFO] Starting to retrieve query status")

	iIDStr := c.String("iid")
	if iIDStr == "" {

		return xerrors.New("--iid flag is required")
	}
	iIDBuf, err := hex.DecodeString(iIDStr)
	if err != nil {
		return err
	}
	iid := byzcoin.NewInstanceID(iIDBuf)

	bcArg := c.String("bc")
	if bcArg == "" {
		return xerrors.New("--bc flag is required")
	}

	cfg, bcl, err := bcadminlib.LoadConfig(bcArg)
	if err != nil {
		return err
	}

	dstr := c.String("darc") //TODO: Read darc from file? admin?
	if dstr == "" {
		dstr = cfg.AdminDarc.GetIdentityString()
	}

	projectDarc, err := bcadminlib.GetDarcByString(bcl, dstr)
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

	log.Info("[INFO] Sending request to", si.String()) //TODO: double-check server address??
	name := projectDarc.Description
	client, err := s.NewClient(bcl, si, clientID)
	if err != nil {
		return xerrors.Errorf("failed to init client: %v", err)
	}

	err = client.Create()
	if err != nil {
		return xerrors.Errorf("failed to create client: %v", err)
	}

	client.AllDarcIDs[string(name)] = projectDarc.GetBaseID() //TODo: what about other darcs?

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

// getClient will create a new MedChain.Client, given the input
// available in the commandline. If priv is false, then it will not
// look for a private key and set up the signers.
func getClient(c *cli.Context, priv bool) (*s.Client, error) {

	log.Info("[INFO] Getting MedChain CLI client")
	bc := c.String("bc")
	if bc == "" {
		return nil, xerrors.Errorf("--bc flag is required to create the client")
	}

	cfgBuf, err := ioutil.ReadFile(bc)
	if err != nil {
		return nil, err
	}
	var cfg bcConfig
	err = protobuf.DecodeWithConstructors(cfgBuf, &cfg,
		network.DefaultConstructors(cothority.Suite))
	if err != nil {
		return nil, xerrors.Errorf("failed to get byzcoin config: %v", err)
	}

	log.Info("[INFO] Reading client ID")
	cidArg := c.String("cid")
	if cidArg == "" {
		return nil, xerrors.New("--cid flag is required")
	}

	groupTomlPath := c.String("file")
	if groupTomlPath == "" {
		return nil, xerrors.New("--file flag is required")
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
			return nil, errors.New("port not found, must provide addr:port")
		}
		list = append(list, si)
	} else {

		roster, err := readGroup(groupTomlPath)
		if err != nil {
			return nil, errors.New("couldn't read file: " + err.Error())
		}
		list = roster.List
		log.Info("[INFO] Roster list is", list)
	}
	log.Info("[INFO] Roster is ", list)
	log.Info("[INFO] Sending request to", si)

	client, err := s.NewClient(byzcoin.NewClient(cfg.ByzCoinID, cfg.Roster), si, cidArg)
	if err != nil {
		return nil, xerrors.Errorf("failed to init client: %v", err)
	}

	// Initialize project Darcs hash map
	client.AllDarcs = make(map[string]*darc.Darc)
	client.AllDarcIDs = make(map[string]darc.ID)

	// get the private key from the cmdline.
	sstr := c.String("key")
	if sstr == "" {
		return nil, errors.New("--key is required")
	}
	signer, err := bcadminlib.LoadKeyFromString(sstr)
	if err != nil {
		return nil, err
	}
	client.Signers = []darc.Signer{*signer}

	return client, nil
}
