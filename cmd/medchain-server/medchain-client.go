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
	//   2. Gets DarcID of genesis darc that has invoke rules for deferred, medchain, etc contracs
	//   3. Creates MedChain client

	// ---
	// 1. Start MedChain client
	// ---
	log.Info("[INFO] Creating the MedChain CLI client:")
	mccl, err := getClient(c)
	if err != nil {
		return xerrors.Errorf("[INFO] failed to get medchain client:", err)
	}
	// ---
	// 2. GetsDarcID of genesis darc that has invoke rules for deferred, medchain, etc contracs
	// ---
	darcArg := c.String("darc")
	if darcArg == "" {
		log.Info("[INFO] GenDarcID was not given, fetching th elatest version of GenDarc from ByzCoin")
		genDarc, err := mccl.Bcl.GetGenDarc()
		if err != nil {
			return err
		}
		mccl.GenDarcID = genDarc.GetBaseID()
	} else {
		log.Info("[INFO] Retrieving GenDarcID by ID provided:", darcArg)
		darcBuf, err := bcadminlib.StringToDarcID(darcArg)
		if err != nil {
			return err
		}
		mccl.GenDarcID = darc.ID(darcBuf)
	}
	// ---
	// 3. Creates MedChain client
	// ---
	err = mccl.Create()
	if err != nil {
		return err
	}
	log.Info("[INFO] Created MedChain with genesis darc ID:", mccl.GenDarcID)
	log.Info("[INFO] Created MedChain with naming instance ID:", mccl.GenDarcID)
	return bcadminlib.WaitPropagation(c, mccl.Bcl)
}

func submitQuery(c *cli.Context) error {
	// Here is what this function does:
	//   1. Gets MedChain client
	//   2. Gets DarcID and rerives it from bzycoin
	//   3. Gets the proposed query
	//   4. Fires a spawn instruction for the deferred contract
	//	 6. Gets the response back from MedChain service
	//	 7. Broadcasts instanceID to all MedChain nodes
	//   8. Writes query instanceID to file

	// ---
	// 1. Get MedChain client
	// ---
	log.Info("[INFO] Starting query submission")
	mccl, err := getClient(c)
	if err != nil {
		return xerrors.Errorf("[INFO] failed to get medchain client:", err)
	}
	// ---
	// 2. Get DarcID and retrieve it from bzycoin
	// ---
	// TODO: Broadcast all darc ID after the are created to all nodes, rea them from file
	// This implementation relies on the user to provide the right darc ID for the corresponding
	// project
	log.Info("[INFO] Reading Darc ID")
	darcIDArg := c.String("darc")
	if darcIDArg == "" {
		return xerrors.New("--darc flag is required")
	}
	log.Info("[INFO] Getting Darc by ID:", darcIDArg)
	projectDarc, err := bcadminlib.GetDarcByString(mccl.Bcl, darcIDArg)
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
	mccl.AllDarcs[string(projectName)] = projectDarc
	mccl.AllDarcIDs[string(projectName)] = projectDarc.GetBaseID()

	// ---
	// 4. Fire a spawn instruction for the deferred contract
	// 5. Get the response back from MedChain service
	// 6. Broadcast instanceID to all MedChain nodes
	// ---
	req := &s.AddDeferredQueryRequest{}
	req.QueryID = proposedQuery.ID
	reply, err := mccl.SpawnDeferredQuery(req)
	if err != nil {
		return xerrors.Errorf("failed to spawn query instance: %v", err)
	}
	if reply.OK != true {
		return xerrors.Errorf("service failed to spawn query instance: %v", err)
	}
	mccl.Bcl.WaitPropagation(1)

	// ---
	// 7.  Write query instance ID to file
	// ---
	instIDfilePath := c.String("idfile")
	if instIDfilePath == "" {
		return xerrors.New("--idfile flag is required")
	}
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

func addSignature(c *cli.Context) error {
	// Here is what this function does:
	//   1. Starts MedChain client
	//   2. Reads instanceID of query to be signed from file from flag
	//   3. Sign proposed transaction
	//	 4. Gets the response back from MedChain service
	//   5. Reads the return back and prints it

	// ---
	// 1. Start MedChain client
	// ---
	log.Info("[INFO] Creating the MedChain CLI client:")
	mccl, err := getClient(c)
	if err != nil {
		return xerrors.Errorf("[INFO] failed to get medchain client:", err)
	}
	// ---
	// 2. Read instanceID of query to be signed from file from flag
	// ---
	log.Lvl1("[INFO] Starting adding signature to deferred query")

	iIDStr := c.String("instid")
	if iIDStr == "" {

		return xerrors.New("--instid flag is required")
	}
	iIDBuf, err := hex.DecodeString(iIDStr)
	if err != nil {
		return err
	}
	iid := byzcoin.NewInstanceID(iIDBuf)

	log.Info("[INFO] Sending request to", mccl.EntryPoint.String()) //TODO: exact server address -> done

	// ---
	// 3. Sign proposed transaction
	// ---
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
	// mccl.Bcl.WaitPropagation(1)

	// ---
	// 5. Read the return back and prints it
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

	log.Infof("Here is the deferred data: \n%s", result)
	return nil
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
		return xerrors.Errorf("[INFO] failed to get medchain client:", err)
	}
	// ---
	// 2. Read instanceID of query to be signed from file from flag
	// ---
	log.Lvl1("[INFO] Starting adding signature to deferred query")

	iIDStr := c.String("instid")
	if iIDStr == "" {

		return xerrors.New("--instid flag is required")
	}
	iIDBuf, err := hex.DecodeString(iIDStr)
	if err != nil {
		return err
	}
	iid := byzcoin.NewInstanceID(iIDBuf)

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
	log.Info("[INFO] Getting MedChain CLI client")
	log.Info("[INFO] Reading ByzCoin config file")
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
	log.Info("[INFO] Reading client ID")
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

	var list []*network.ServerIdentity
	var si *network.ServerIdentity

	// ---
	// 4. Gets the identity of server to contact to
	// ---
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

	// ---
	// 5. Init MedChain client
	// ---
	client, err := s.NewClient(byzcoin.NewClient(cfg.ByzCoinID, cfg.Roster), si, cidArg)
	if err != nil {
		return nil, xerrors.Errorf("failed to init client: %v", err)
	}

	// Initialize project Darcs hash map
	client.AllDarcs = make(map[string]*darc.Darc)
	client.AllDarcIDs = make(map[string]darc.ID)

	// ---
	// 6. Get the private key from the cmdline
	// ---
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

func createkey(c *cli.Context) error {
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
