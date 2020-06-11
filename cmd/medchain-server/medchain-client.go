package main

import (
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"strings"

	"io/ioutil"

	s "github.com/medchain/services"
	cli "github.com/urfave/cli"
	"go.dedis.ch/cothority/v3"
	"go.dedis.ch/cothority/v3/byzcoin"
	"go.dedis.ch/cothority/v3/byzcoin/bcadmin/lib"
	bcadminlib "go.dedis.ch/cothority/v3/byzcoin/bcadmin/lib"
	"go.dedis.ch/cothority/v3/darc"
	"go.dedis.ch/cothority/v3/skipchain"
	"go.dedis.ch/onet/v3"
	"go.dedis.ch/onet/v3/app"
	"go.dedis.ch/onet/v3/log"
	"go.dedis.ch/onet/v3/network"
	"go.dedis.ch/protobuf"
	"golang.org/x/xerrors"
)

type config struct {
	Name            string
	QueryInstanceID byzcoin.InstanceID
}

type bcConfig struct {
	Roster    onet.Roster
	ByzCoinID skipchain.SkipBlockID
}

func create(c *cli.Context) error {
	// Here is what this function does:
	//   1. Starts MedChain client
	//   2. Creates MedChain client

	// ---
	// 1. Start MedChain client
	// ---
	log.Info("[INFO] (CLI)Creating MedChain CLI client")
	mccl, err := getClient(c)
	if err != nil {
		return xerrors.Errorf("failed to get medchain client: %v", err)
	}

	// ---
	// 2. Creates MedChain client
	// ---
	err = mccl.Create()
	if err != nil {
		return err
	}
	log.Info("[INFO] Created MedChain with genesis darc ID:", mccl.GenDarc.GetIdentityString())
	return bcadminlib.WaitPropagation(c, mccl.Bcl)
}

func submitQuery(c *cli.Context) error {
	// Here is what this function does:
	//   1. Gets MedChain client
	//   2. Gets DarcID and rerives it from bzycoin
	//   3. Gets the proposed query
	//   4. Fires a spawn instruction for the deferred contract
	//	 5. Gets the response back from MedChain service
	//	 6. Broadcasts instanceID to all MedChain nodes
	//   7. Writes query instanceID to file

	// ---
	// 1. Get MedChain client
	// ---
	log.Info("[INFO] (CLI) Starting query submission")
	mccl, err := getClient(c)
	if err != nil {
		return xerrors.Errorf("failed to get medchain client: :%v", err)
	}
	// ---
	// 2. Get DarcID and retrieve it from bzycoin
	// ---
	// TODO: Broadcast all darc ID after the are created to all nodes, rea them from file
	// This implementation relies on the user to provide the right darc ID for the corresponding
	// project
	log.Info("[INFO] (CLI) Reading Darc ID")
	darcIDArg := c.String("darc")
	if darcIDArg == "" {
		return xerrors.New("--darc flag is required")
	}
	log.Info("[INFO] (CLI) Getting Darc by ID:", darcIDArg)
	projectDarc, err := bcadminlib.GetDarcByString(mccl.Bcl, darcIDArg)
	if err != nil {
		return xerrors.Errorf("failed to get project darc: %v", err)
	}

	// ---
	//  3. Get the proposed query
	// ---
	log.Info("[INFO] (CLI) Reading the query")
	queryArg := c.String("queryid")
	if queryArg == "" {
		return xerrors.New("--queryid flag is required")
	}

	proposedQuery := s.NewQuery(queryArg, " ")
	qq := strings.Split(proposedQuery.ID, ":")

	if len(qq) != 3 {
		return xerrors.New("invalid query entered")
	}

	instIDfilePath := c.String("idfile")
	if instIDfilePath == "" {
		return xerrors.New("--idfile flag is required")
	}

	projectName := qq[1]
	mccl.AllDarcs[string(projectName)] = projectDarc
	mccl.AllDarcIDs[string(projectName)] = projectDarc.GetBaseID()

	// ---
	// 5. Fire a spawn instruction for the deferred contract
	// 6. Get the response back from MedChain service
	// 7. Broadcast instanceID to all MedChain nodes
	// ---
	log.Info("[INFO] (CLI) Sending request to API")
	log.Info("[INFO] (CLI) If the query is authorized it will be sent for other users to sign")
	req := &s.AddQueryRequest{}
	req.ClientID = mccl.ClientID
	req.QueryID = proposedQuery.ID
	req.BlockID = mccl.Bcl.ID
	req.DarcID = projectDarc.GetBaseID()
	req.QueryStatus = []byte("Submitted")

	reply, err := mccl.SpawnQuery(req)
	if err != nil {
		return xerrors.Errorf("failed to spawn query instance: %v", err)
	}
	if reply.OK != true {
		return xerrors.Errorf("service failed to spawn query instance: %v", err)
	}
	mccl.Bcl.WaitPropagation(1)

	// ---
	// 8.  Write query instance ID to file
	// ---

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
	log.Info("[INFO] (CLI) Query was submitted successfully")

	return bcadminlib.WaitPropagation(c, mccl.Bcl)
}

func addSignature(c *cli.Context) error {
	// Here is what this function does:
	//   1. Starts MedChain client
	//   2. Reads instanceID of query to be signed from file from flag
	//   3. Sign proposed transaction
	//	 4. Gets the response back from MedChain service
	//   5. Reads the deferred data and retrieves it back and prints it

	// ---
	// 1. Start MedChain client
	// ---
	log.Info("[INFO] Creating the MedChain CLI client:")
	mccl, err := getClient(c)
	if err != nil {
		return xerrors.Errorf("failed to get medchain client: %v", err)
	}
	// ---
	// 2. Read instanceID of query to be signed from file from flag
	// ---
	log.Info("[INFO] Starting adding signature to deferred query")

	iIDStr := c.String("instid")
	if iIDStr == "" {

		return xerrors.New("--instid flag is required")
	}
	iIDBuf, err := hex.DecodeString(iIDStr)
	if err != nil {
		return err
	}
	iid := byzcoin.NewInstanceID(iIDBuf)

	log.Info("[INFO] (CLI) Sending request to", mccl.EntryPoint.String())

	mccl.SyncSignerCtrs(mccl.Signers...)
	// ---
	// 3. Sign proposed transaction
	// ---
	log.Info("[INFO] (CLI) Sending signing request to API")
	req := &s.SignDeferredTxRequest{}
	req.ClientID = mccl.ClientID
	req.Keys = mccl.Signers[0]
	req.QueryInstID = iid
	reply, err := mccl.AddSignatureToDeferredQuery(req)
	if err != nil {
		return xerrors.Errorf("failed to add signature to query instance %v: %v", req.QueryInstID.String(), err)
	}

	// ---
	// 4. Get the response back from MedChain service
	// ---
	if reply.OK != true {
		return xerrors.Errorf("failed to add signature to query instance %v: %v", req.QueryInstID.String(), err)
	}
	mccl.Bcl.WaitPropagation(1)

	// ---
	// 5. Reads the deferred data and retrieves it back and prints it
	// ---
	err = bcadminlib.WaitPropagation(c, mccl.Bcl)
	if err != nil {
		return xerrors.Errorf("waiting on propagation failed: %+v", err)
	}
	pr, err := mccl.Bcl.GetProofFromLatest(iIDBuf)
	if err != nil {
		return xerrors.Errorf("couldn't get proof for admin-darc: %+v", err)
	}

	_, resultBuf, _, _, err := pr.Proof.KeyValue()
	if err != nil {
		return xerrors.Errorf("couldn't get value out of proof: %+v", err)
	}

	result := byzcoin.DeferredData{}
	err = protobuf.Decode(resultBuf, &result)
	if err != nil {
		return xerrors.Errorf("couldn't decode the result: %+v", err)
	}

	log.Infof("[INFO] (CLI) Here is the deferred data after adding signature: \n%s", result)
	return bcadminlib.WaitPropagation(c, mccl.Bcl)
}

func execDefferedQuery(c *cli.Context) error {
	// Here is what this function does:
	//   1. Starts MedChain client
	//   2. Reads instanceID of query to be signed from file from flag
	//   3. Executes proposed transaction
	//	 4. Gets the response back from MedChain service
	//   5. Reads the deferred data and retrieves it back and prints it

	// ---
	// 1. Start MedChain client
	// ---
	log.Info("[INFO] Creating the MedChain CLI client:")
	mccl, err := getClient(c)
	if err != nil {
		return xerrors.Errorf("failed to get medchain client: %v", err)
	}
	// ---
	// 2. Read instanceID of query to be signed from file from flag
	// ---
	log.Info("[INFO] Starting execution of deferred query")

	iIDStr := c.String("instid")
	if iIDStr == "" {

		return xerrors.New("--instid flag is required")
	}
	iIDBuf, err := hex.DecodeString(iIDStr)
	if err != nil {
		return err
	}
	iid := byzcoin.NewInstanceID(iIDBuf)

	log.Info("[INFO] (CLI) Sending execution request to", mccl.EntryPoint.String())

	mccl.SyncSignerCtrs(mccl.Signers...)
	// ---
	// 3. execute proposed transaction
	// ---
	log.Info("[INFO] (CLI) Sending execution request to API")
	req := &s.ExecuteDeferredTxRequest{}
	req.ClientID = mccl.ClientID
	req.QueryInstID = iid
	reply, err := mccl.ExecDefferedQuery(req)
	if err != nil {
		return xerrors.Errorf("failed to execute the query instance %v: %v", req.QueryInstID.String(), err)
	}

	// ---
	// 4. Get the response back from MedChain service
	// ---
	if reply.OK != true {
		return xerrors.Errorf("failed to execute the query instance %v: %v", req.QueryInstID.String(), err)
	}
	mccl.Bcl.WaitPropagation(1)

	// ---
	// 5. Reads the deferred data and retrieves it back and prints it
	// ---
	err = bcadminlib.WaitPropagation(c, mccl.Bcl)
	if err != nil {
		return xerrors.Errorf("waiting on propagation failed: %+v", err)
	}
	pr, err := mccl.Bcl.GetProofFromLatest(iIDBuf)
	if err != nil {
		return xerrors.Errorf("couldn't get proof for admin-darc: %+v", err)
	}

	_, resultBuf, _, _, err := pr.Proof.KeyValue()
	if err != nil {
		return xerrors.Errorf("couldn't get value out of proof: %+v", err)
	}

	result := byzcoin.DeferredData{}
	err = protobuf.Decode(resultBuf, &result)
	if err != nil {
		return xerrors.Errorf("couldn't decode the result: %+v", err)
	}

	log.Infof("[INFO] (CLI) Here is the deferred data after exectution: \n%s", result)

	log.Info("[INFO] (CLI) Execution was successful")
	return bcadminlib.WaitPropagation(c, mccl.Bcl)
}

func fetchInstanceIDs(c *cli.Context) error {
	// ---
	// 1. Start MedChain client
	// ---
	log.Info("[INFO] Creating the MedChain CLI client:")
	mccl, err := getClient(c)
	if err != nil {
		return xerrors.Errorf("failed to get medchain client: %v", err)
	}

	log.Infof("[INFO] (CLI) Getting all instance IDs from the server %v", mccl.EntryPoint)
	iids, err := mccl.GetSharedData()
	if err != nil {
		xerrors.Errorf("failed to fetch instance IDs: %v", err)
	}
	for _, iid := range iids.QueryInstIDs {
		log.Infof("[INFO] Fetched instance ID from the server %v: %v", mccl.EntryPoint, iid.String())
	}
	return bcadminlib.WaitPropagation(c, mccl.Bcl)
}

func verifyStatus(c *cli.Context) error {
	// Here is what this function does:
	//   1. Reads instanceID of query to check its status on the chain
	//   2. Retrieves the query from chain
	//	 3. Returns the status of query
	//   4. Writes query instanceID to file

	// ---
	// 1. Start MedChain client
	// ---
	log.Info("[INFO] Creating the MedChain CLI client:")
	mccl, err := getClient(c)
	if err != nil {
		return xerrors.Errorf("[INFO] failed to get medchain client: %v", err)
	}
	// ---
	// 2. Read instanceID of query to be signed from file from flag
	// ---
	log.Info("[INFO] Starting adding signature to deferred query")

	iIDStr := c.String("instid")
	if iIDStr == "" {

		return xerrors.New("--instid flag is required")
	}
	// iIDBuf, err := hex.DecodeString(iIDStr)
	if err != nil {
		return err
	}
	// iid := byzcoin.NewInstanceID(iIDBuf

	log.Info("[INFO] Sending request to", mccl.EntryPoint.String())
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
func getClient(c *cli.Context) (*s.Client, error) {
	// Here is what this function does:
	//   1. Reads Byzcoin config to get config and Byzcoin client
	//   2. Gets cleint ID
	//   3. Reads group file
	//   4. Gets the identity of server to contact to
	//   5. Init MedChain client
	//   6. Get the private key from the cmdline

	// ---
	// 1. Read Byzcoin config to get config and Byzcoin client
	// ---
	log.Info("[INFO] (CLI) Getting MedChain CLI client")
	log.Info("[INFO] (CLI) Reading ByzCoin config file")
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

	// ---
	// 2. Get cleint ID
	// ---
	log.Info("[INFO] (CLI) Reading client ID")
	cidArg := c.String("cid")
	if cidArg == "" {
		return nil, xerrors.New("--cid flag is required")
	}

	// ---
	// 3. Read group file
	// ---
	groupTomlPath := c.String("file")
	if groupTomlPath == "" {
		return nil, xerrors.New("--file flag is required")
	}

	// ---
	// 4. Gets the identity of server to contact to
	// ---

	var si *network.ServerIdentity

	roster, err := readGroup(groupTomlPath)
	if err != nil {
		return nil, errors.New("couldn't read group file: " + err.Error())
	}
	list := roster.List
	log.Info("[INFO] (CLI) Roster list is", list)
	address := c.String("address")
	if address != "" {
		// Contact desired server
		addr := network.Address(address)
		log.Info("[INFO] (CLI) Network Address", addr.String())
		if !strings.HasPrefix(address, "tls://") {
			addr = network.NewAddress(network.TLS, address)
		}
		newSi := network.NewServerIdentity(nil, addr)
		if newSi.Address.Port() == "" {
			return nil, xerrors.New("port not found, must provide addr:port")
		}
		log.Infof("[INFO] (CLI) Finding server identity with address%v", newSi.Address.String())
		var found = false
		for _, id := range list {
			if id.Address == newSi.Address {
				found = true
				si = id
			}
		}
		if !found {
			return nil, xerrors.Errorf("could not find server identity at address: %v", address)
		}
	} else {
		log.Info("[INFO] (CLI) --address was not provideed. Contacting a random server... ", list)
		si := roster.RandomServerIdentity()
		log.Info("[INFO] (CLI) Roster list is", list)
		log.Infof("[INFO] (CLI) Using server %v", si.String())
	}
	// ---
	// 5. Get the private key from the cmdline
	// ---
	sstr := c.String("key")
	if sstr == "" {
		return nil, errors.New("--key is required")
	}
	signer, err := bcadminlib.LoadKeyFromString(sstr)
	if err != nil {
		return nil, err
	}

	// ---
	// 6. Init MedChain client
	// ---
	client, err := s.NewClient(byzcoin.NewClient(cfg.ByzCoinID, cfg.Roster), si, cidArg, *signer)
	if err != nil {
		return nil, xerrors.Errorf("failed to init client: %v", err)
	}

	// Initialize project Darcs hash map
	client.AllDarcs = make(map[string]*darc.Darc)
	client.AllDarcIDs = make(map[string]darc.ID)
	client.ClientID = cidArg
	client.EntryPoint = si

	return client, nil
}

func createKey(c *cli.Context) error {
	if f := c.String("print"); f != "" {
		sig, err := bcadminlib.LoadSigner(f)
		if err != nil {
			return errors.New("couldn't load signer: " + err.Error())
		}
		log.Infof("Private: %s\nPublic: %s", sig.Ed25519.Secret, sig.Ed25519.Point)
		return nil
	}
	newSigner := darc.NewSignerEd25519(nil, nil)
	err := bcadminlib.SaveKey(newSigner)
	if err != nil {
		return err
	}

	var fo io.Writer

	save := c.String("save")
	if save == "" {
		fo = os.Stdout
	} else {
		file, err := os.Create(save)
		if err != nil {
			return err
		}
		fo = file
		defer func() {
			err := file.Close()
			if err != nil {
				log.Error(err)
			}
		}()
	}
	_, err = fmt.Fprintln(fo, newSigner.Identity().String())
	return err
}

func addProjectDarc(c *cli.Context) error {
	log.Info("[INFO] (CLI) Adding project darc")
	mccl, err := getClient(c)
	if err != nil {
		return err
	}
	pname := c.String("name")
	if pname == "" {
		return errors.New("--name is required")
	}
	mccl.SyncSignerCtrs(mccl.Signers...)

	// TODO broadcast the base ID
	darc, err := mccl.AddProjectDarc(pname)
	if err != nil {
		return xerrors.Errorf("error in adding project darc: %w", err)
	}
	log.Infof("[INFO] (CLI) Created Darc for project %v with based ID %v", pname, darc.GetIdentityString())

	var fo io.Writer
	output := c.String("out_id")
	if output != "" {
		log.Infof("[INFO] (CLI) Saving darc %v id in %v", darc.GetIdentityString(), output)
		file, err := os.Create(output)
		if err != nil {
			return err
		}
		fo = file
		defer func() {
			err := file.Close()
			if err != nil {
				log.Error(err)
			}
		}()
	} else {
		fo = os.Stdout
	}
	_, err = fmt.Fprintln(fo, darc.GetIdentityString())

	return bcadminlib.WaitPropagation(c, mccl.Bcl)
}

func addSigner(c *cli.Context) error {
	log.Info("[INFO] (CLI) Adding project darc")
	mccl, err := getClient(c)
	if err != nil {
		return err
	}
	pname := c.String("name")
	if pname == "" {
		return errors.New("--name is required")
	}
	darcIDStr := c.String("id")
	if darcIDStr == "" {
		return errors.New("--id is required")
	}
	rules := c.StringSlice("rule")
	if rules == nil {
		return errors.New("--rule is required")
	}
	typeStr := c.String("type")
	if typeStr == "" {
		return errors.New("--type is required")
	}

	idStr := c.String("identity")

	if len(idStr) == 0 {
		if !c.Bool("delete") {
			return xerrors.New("--identity flag is required")
		}
	}
	mccl.SyncSignerCtrs(mccl.Signers...)
	darcBuf, err := bcadminlib.StringToDarcID(darcIDStr)
	if err != nil {
		return err
	}
	darcID := darc.ID(darcBuf)
	d, err := bcadminlib.GetDarcByString(mccl.Bcl, darcIDStr)
	if err != nil {
		return xerrors.Errorf("failed to get the darc: %v", err)
	}
	var actions []darc.Action
	for _, rule := range rules {
		actions = append(actions, darc.Action(rule))
	}
	if c.Bool("delete") {
		if len(actions) > 1 {
			return xerrors.New("single rule can be deleted at a time")
		}
		err = d.Rules.DeleteRules(actions[0])
		if err != nil {
			return xerrors.Errorf("failed to delete rule: %v", err)
		}
		log.Infof("[INFO] (CLI) Deleted rule %v from darc with ID %v ", rules, darcIDStr)

	} else {
		err = mccl.AddSignerToDarc(pname, darcID, actions, idStr, typeStr)
		if err != nil {
			return xerrors.Errorf("error in adding signer to darc: %w", err)
		}
		log.Infof("[INFO] (CLI) Added identitiy %v to darc with ID %v ", idStr, darcIDStr)
	}
	log.Info("[INFO] (CLI) Signing was successful")

	return bcadminlib.WaitPropagation(c, mccl.Bcl)
}

func getQuery(c *cli.Context) error {
	mccl, err := getClient(c)
	if err != nil {
		return err
	}
	instID := c.String("instid")
	if instID == "" {
		return xerrors.New("--id flag is required")
	}
	instIDBuf, err := hex.DecodeString(instID)
	if err != nil {
		return err
	}
	id := byzcoin.NewInstanceID(instIDBuf)
	dd, err := mccl.Bcl.GetDeferredData(id)
	log.Infof("[INFO] (CLI) Instance ID: %v ", id.String())
	log.Infof("[INFO] (CLI) Retrieved deferred data is %v ", dd)
	return bcadminlib.WaitPropagation(c, mccl.Bcl)
}

// only used in the demo
func updateGenesisDarc(c *cli.Context) error {
	mccl, err := getClient(c)
	if err != nil {
		return err
	}

	log.Info("[INFO] (CLI) check identities")
	identities := c.StringSlice("identity")
	if identities == nil {
		return xerrors.New("--identity flag is required")
	}

	mccl.SyncSignerCtrs(mccl.Signers...)
	log.Info("[INFO] (CLI) updating the genesis darc")
	err = mccl.UpdateGenesisDarc(identities)
	if err != nil {
		return err
	}
	return bcadminlib.WaitPropagation(c, mccl.Bcl)
}

func darcShow(c *cli.Context) error {
	mccl, err := getClient(c)
	if err != nil {
		return err
	}

	dstr := c.String("darc")
	if dstr == "" {
		return xerrors.New("--darc flag is required")
	}

	d, err := lib.GetDarcByString(mccl.Bcl, dstr)
	if err != nil {
		return xerrors.Errorf("could not get the darc by ID %v : %v", dstr, err)
	}

	log.Infof("[INFO] (CLI) Darc is %v ", d.String())
	return err
}
