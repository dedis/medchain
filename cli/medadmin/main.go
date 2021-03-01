package main

import (
	"encoding/hex"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"strconv"
	"time"

	admin "github.com/medchain/admin"
	"github.com/medchain/contracts"
	cli "github.com/urfave/cli"
	"go.dedis.ch/cothority/v3"
	"go.dedis.ch/cothority/v3/byzcoin"
	"go.dedis.ch/cothority/v3/byzcoin/bcadmin/lib"
	bcadminlib "go.dedis.ch/cothority/v3/byzcoin/bcadmin/lib"
	"go.dedis.ch/cothority/v3/darc"
	"golang.org/x/xerrors"

	"go.dedis.ch/cothority/v3/skipchain"
	"go.dedis.ch/onet/v3"
	"go.dedis.ch/onet/v3/cfgpath"
	"go.dedis.ch/onet/v3/log"
	"go.dedis.ch/onet/v3/network"
	"go.dedis.ch/protobuf"
)

type bcConfig struct {
	Roster    onet.Roster
	ByzCoinID skipchain.SkipBlockID
}

var cliApp = cli.NewApp()
var gitTag = "dev"

func init() {
	cliApp.Name = "medadmin"
	cliApp.Usage = "Medchain administration CLI"
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
		log.SetDebugVisible(0)
		bcadminlib.ConfigPath = c.String("config")
		return nil
	}

}

func main() {
	rand.Seed(time.Now().UTC().UnixNano())
	log.ErrFatal(cliApp.Run(os.Args))
}

func getClient(c *cli.Context, priv bool) (*admin.Client, error) {
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

	// get the private key from the cmdline.
	sstr := c.String("keys")
	if sstr == "" {
		return nil, errors.New("--keys is required")
	}
	signer, err := bcadminlib.LoadKeyFromString(sstr)
	if err != nil {
		return nil, err
	}
	cl, err := admin.NewClientWithAuth(byzcoin.NewClient(cfg.ByzCoinID, cfg.Roster), signer)
	if err != nil {
		return nil, xerrors.Errorf("spawning a new Admin Darc: %w", err)
	}
	return cl, nil
}

func myqueryAdd(c *cli.Context) error {
	// identity := c.String("identity")
	// expireTimestamp := c.String("expireTimestamp")

	return nil
}

func myquerySpawnQuery(c *cli.Context) error {
	description := c.String("description")
	action := c.String("action")
	projectID := c.String("projectid")

	projectIDBuf, err := hex.DecodeString(projectID)
	if err != nil {
		return xerrors.Errorf("failed to decode instance id: %v", err)
	}

	bcArg := c.String("bc")
	if bcArg == "" {
		return xerrors.New("--bc flag is required")
	}

	cfg, cl, err := lib.LoadConfig(bcArg)
	if err != nil {
		return err
	}

	pr, err := cl.GetProofFromLatest(projectIDBuf)
	if err != nil {
		return xerrors.Errorf("couldn't get proof: %v", err)
	}
	proof := pr.Proof

	var signer *darc.Signer

	sstr := c.String("sign")
	if sstr == "" {
		signer, err = lib.LoadKey(cfg.AdminIdentity)
	} else {
		signer, err = lib.LoadKeyFromString(sstr)
	}
	if err != nil {
		return err
	}

	counters, err := cl.GetSignerCounters(signer.Identity().String())
	if err != nil {
		return fmt.Errorf("couldn't get signer counters: %v", err)
	}

	ctx, err := cl.CreateTransaction(byzcoin.Instruction{
		InstanceID: byzcoin.NewInstanceID(proof.InclusionProof.Key()),
		Spawn: &byzcoin.Spawn{
			ContractID: contracts.QueryContractID,
			Args: byzcoin.Arguments{
				{
					Name:  contracts.QueryDescriptionKey,
					Value: []byte(description),
				},
				{
					Name:  contracts.QueryActionKey,
					Value: []byte(action),
				},
			},
		},
		SignerCounter: []uint64{counters.Counters[0] + 1},
	})

	if err != nil {
		return err
	}

	err = ctx.FillSignersAndSignWith(*signer)
	if err != nil {
		return err
	}

	_, err = cl.AddTransactionAndWait(ctx, 10)
	if err != nil {
		return err
	}

	instID := ctx.Instructions[0].DeriveID("").Slice()
	fmt.Printf("Spawned a new value contract. Its instance id is:\n%x", instID)

	return lib.WaitPropagation(c, cl)
}

func myquerySpawnProject(c *cli.Context) error {
	description := c.String("description")
	name := c.String("name")
	instID := c.String("instid")

	instIDBuf, err := hex.DecodeString(instID)
	if err != nil {
		return xerrors.New("failed to decode the instid string")
	}

	bcArg := c.String("bc")
	if bcArg == "" {
		return xerrors.New("--bc flag is required")
	}

	cfg, cl, err := lib.LoadConfig(bcArg)
	if err != nil {
		return err
	}

	var signer *darc.Signer

	sstr := c.String("sign")
	if sstr == "" {
		signer, err = lib.LoadKey(cfg.AdminIdentity)
	} else {
		signer, err = lib.LoadKeyFromString(sstr)
	}
	if err != nil {
		return err
	}

	counters, err := cl.GetSignerCounters(signer.Identity().String())
	if err != nil {
		return fmt.Errorf("couldn't get signer counters: %v", err)
	}

	ctx, err := cl.CreateTransaction(byzcoin.Instruction{
		InstanceID: byzcoin.NewInstanceID(instIDBuf),
		Spawn: &byzcoin.Spawn{
			ContractID: contracts.ProjectContractID,
			Args: byzcoin.Arguments{
				{
					Name:  contracts.ProjectDescriptionKey,
					Value: []byte(description),
				},
				{
					Name:  contracts.ProjectNameKey,
					Value: []byte(name),
				},
			},
		},
		SignerCounter: []uint64{counters.Counters[0] + 1},
	})

	if err != nil {
		return err
	}

	err = ctx.FillSignersAndSignWith(*signer)
	if err != nil {
		return err
	}

	_, err = cl.AddTransactionAndWait(ctx, 10)
	if err != nil {
		return err
	}

	instIDStr := ctx.Instructions[0].DeriveID("").Slice()
	fmt.Printf("Spawned a new project contract. Its instance id is:\n%x", instIDStr)

	return lib.WaitPropagation(c, cl)
}

func myqueryAddToProject(c *cli.Context) error {
	userID := c.String("userID")
	action := c.String("action")
	instID := c.String("instid")

	instIDBuf, err := hex.DecodeString(instID)
	if err != nil {
		return xerrors.New("failed to decode the instid string")
	}

	bcArg := c.String("bc")
	if bcArg == "" {
		return xerrors.New("--bc flag is required")
	}

	cfg, cl, err := lib.LoadConfig(bcArg)
	if err != nil {
		return err
	}

	var signer *darc.Signer

	sstr := c.String("sign")
	if sstr == "" {
		signer, err = lib.LoadKey(cfg.AdminIdentity)
	} else {
		signer, err = lib.LoadKeyFromString(sstr)
	}
	if err != nil {
		return err
	}

	counters, err := cl.GetSignerCounters(signer.Identity().String())
	if err != nil {
		return fmt.Errorf("couldn't get signer counters: %v", err)
	}

	ctx, err := cl.CreateTransaction(byzcoin.Instruction{
		InstanceID: byzcoin.NewInstanceID(instIDBuf),
		Invoke: &byzcoin.Invoke{
			ContractID: contracts.ProjectContractID,
			Command:    "add",
			Args: byzcoin.Arguments{
				{
					Name:  contracts.ProjectUserIDKey,
					Value: []byte(userID),
				},
				{
					Name:  contracts.ProjectActionKey,
					Value: []byte(action),
				},
			},
		},
		SignerCounter: []uint64{counters.Counters[0] + 1},
	})

	if err != nil {
		return err
	}

	err = ctx.FillSignersAndSignWith(*signer)
	if err != nil {
		return err
	}

	_, err = cl.AddTransactionAndWait(ctx, 10)
	if err != nil {
		return err
	}

	fmt.Println("Identity added")

	return lib.WaitPropagation(c, cl)
}

func myqueryPrintProject(c *cli.Context) error {
	projectID := c.String("projectid")

	projectIDBuf, err := hex.DecodeString(projectID)
	if err != nil {
		return xerrors.Errorf("failed to decode instance id: %v", err)
	}

	bcArg := c.String("bc")
	if bcArg == "" {
		return xerrors.New("--bc flag is required")
	}

	_, cl, err := lib.LoadConfig(bcArg)
	if err != nil {
		return err
	}

	pr, err := cl.GetProofFromLatest(projectIDBuf)
	if err != nil {
		return xerrors.Errorf("couldn't get proof: %v", err)
	}
	proof := pr.Proof

	exist, err := proof.InclusionProof.Exists(projectIDBuf)
	if err != nil {
		return xerrors.Errorf("error while checking if proof exist: %v", err)
	}
	if !exist {
		return xerrors.New("proof not found")
	}

	match := proof.InclusionProof.Match(projectIDBuf)
	if !match {
		return xerrors.New("proof does not match")
	}

	_, resultBuf, _, _, err := proof.KeyValue()
	if err != nil {
		return xerrors.Errorf("couldn't get value out of proof: %v", err)
	}

	var project contracts.ProjectContract
	err = protobuf.Decode(resultBuf, &project)
	if err != nil {
		return xerrors.Errorf("failed to unmarshal project: %v", err)
	}

	fmt.Printf("%s\n", project)

	return nil
}

func spawn(c *cli.Context) error {
	cl, err := getClient(c, true)
	if err != nil {
		return err
	}
	cl.SyncSignerCounter()
	darc, err := cl.SpawnNewAdminDarc()
	if err != nil {
		return xerrors.Errorf("spawning a new Admin Darc: %w", err)
	}
	fmt.Println("New admininistration darc spawned :")
	fmt.Println(darc)
	fmt.Println("Admin darc base id:", darc.GetIdentityString())
	return bcadminlib.WaitPropagation(c, cl.Bcl)
}

// func getAdminList(c *cli.Context) error {
// 	cl, err := getClient(c, true)
// 	if err != nil {
// 		return err
// 	}

// 	adidstring := c.String("adid")
// 	if adidstring == "" {
// 		return xerrors.New("--adid flag is required")
// 	}

// 	adid, err := lib.StringToDarcID(adidstring)
// 	if err != nil {
// 		return xerrors.Errorf("failed to parse darc: %v", err)
// 	}

// 	listId, err := cl.Bcl.ResolveInstanceID(adid, "adminsList")
// 	if err != nil {
// 		return xerrors.Errorf("Resolving the instance id of value contract instance: %w", err)
// 	}
// 	list, err := cl.GetAdminsList(listId)
// 	if err != nil {
// 		return xerrors.Errorf("Getting admins list: %w", err)
// 	}
// 	fmt.Println("The list of admin identities in the admin darc:")
// 	fmt.Println(list.List)
// 	return bcadminlib.WaitPropagation(c, cl.Bcl)
// }

func create(c *cli.Context) error {
	keys := darc.NewSignerEd25519(nil, nil)
	fmt.Println("New admin identity key pair created :")
	fmt.Println(keys.Identity().String())
	err := bcadminlib.SaveKey(keys)
	return err
}

// func createList(c *cli.Context) error {
// 	cl, err := getClient(c, true)
// 	if err != nil {
// 		return err
// 	}
// 	adidstring := c.String("adid")
// 	if adidstring == "" {
// 		return xerrors.New("--adid flag is required")
// 	}

// 	adid, err := lib.StringToDarcID(adidstring)
// 	if err != nil {
// 		return xerrors.Errorf("failed to parse darc: %v", err)
// 	}

// 	cl.SyncSignerCounter()
// 	id, err := cl.SpawnAdminsList(adid)
// 	if err != nil {
// 		return xerrors.Errorf("spawning a admins list: %w", err)
// 	}
// 	fmt.Println("Admins list spawned with id:")
// 	fmt.Println(id)
// 	return bcadminlib.WaitPropagation(c, cl.Bcl)
// }

func attachList(c *cli.Context) error {
	cl, err := getClient(c, true)
	if err != nil {
		return err
	}
	instID := c.String("id")
	if instID == "" {
		return xerrors.New("--id flag is required")
	}
	instIDBuf, err := hex.DecodeString(instID)
	if err != nil {
		return err
	}
	id := byzcoin.NewInstanceID(instIDBuf)
	cl.SyncSignerCounter()

	err = cl.AttachAdminsList(id)
	if err != nil {
		return xerrors.Errorf("Attaching the admins list instance id to the admin darc: %w", err)
	}
	fmt.Println("Successfully attached admins list to admin darc with name resolution : adminsList")
	return bcadminlib.WaitPropagation(c, cl.Bcl)
}

// func addAdmin(c *cli.Context) error {
// 	cl, err := getClient(c, true)
// 	if err != nil {
// 		return err
// 	}
// 	id := c.String("identity")
// 	if id == "" {
// 		return xerrors.New("--identity flag is required")
// 	}
// 	adidstring := c.String("adid")
// 	if adidstring == "" {
// 		return xerrors.New("--adid flag is required")
// 	}

// 	adid, err := lib.StringToDarcID(adidstring)
// 	if err != nil {
// 		return xerrors.Errorf("failed to parse darc: %v", err)
// 	}

// 	cl.SyncSignerCounter()
// 	deferredId, err := cl.AddAdminToAdminDarc(adid, id)
// 	if err != nil {
// 		return xerrors.Errorf("spawning a new Admin Darc: %w", err)
// 	}
// 	fmt.Println("Deferred transaction (2 instructions) spawned with ID:")
// 	fmt.Println(deferredId)
// 	return bcadminlib.WaitPropagation(c, cl.Bcl)
// }

// func removeAdmin(c *cli.Context) error {
// 	cl, err := getClient(c, true)
// 	if err != nil {
// 		return err
// 	}
// 	id := c.String("identity")
// 	if id == "" {
// 		return xerrors.New("--identity flag is required")
// 	}
// 	adidstring := c.String("adid")
// 	if adidstring == "" {
// 		return xerrors.New("--adid flag is required")
// 	}

// 	adid, err := lib.StringToDarcID(adidstring)
// 	if err != nil {
// 		return xerrors.Errorf("failed to parse darc: %v", err)
// 	}

// 	cl.SyncSignerCounter()
// 	darc, err := cl.RemoveAdminFromAdminDarc(adid, id)
// 	if err != nil {
// 		return xerrors.Errorf("spawning a new Admin Darc: %w", err)
// 	}
// 	fmt.Println("Deferred transaction (2 instructions) spawned with ID:")
// 	fmt.Println(darc)
// 	return bcadminlib.WaitPropagation(c, cl.Bcl)
// }

// func modifyAdminKey(c *cli.Context) error {
// 	cl, err := getClient(c, true)
// 	if err != nil {
// 		return err
// 	}
// 	newKey := c.String("newkey")
// 	if newKey == "" {
// 		return xerrors.New("--identity flag is required")
// 	}
// 	oldKey := c.String("oldkey")
// 	if oldKey == "" {
// 		return xerrors.New("--identity flag is required")
// 	}
// 	adidstring := c.String("adid")
// 	if adidstring == "" {
// 		return xerrors.New("--adid flag is required")
// 	}

// 	adid, err := lib.StringToDarcID(adidstring)
// 	if err != nil {
// 		return xerrors.Errorf("failed to parse darc: %v", err)
// 	}

// 	cl.SyncSignerCounter()
// 	darc, err := cl.ModifyAdminKeysFromAdminDarc(adid, oldKey, newKey)
// 	if err != nil {
// 		return xerrors.Errorf("spawning a new Admin Darc: %w", err)
// 	}
// 	fmt.Println("Deferred transaction (2 instructions) spawned with ID:")
// 	fmt.Println(darc)
// 	return bcadminlib.WaitPropagation(c, cl.Bcl)
// }

func sync(c *cli.Context) error {
	cl, err := getClient(c, true)
	if err != nil {
		return err
	}
	ids, err := cl.FetchNewDeferredInstanceIDs()
	if err != nil {
		return err
	}
	for _, id := range ids.Ids {
		fmt.Println(id)
	}
	return bcadminlib.WaitPropagation(c, cl.Bcl)
}

func deferredSign(c *cli.Context) error {
	cl, err := getClient(c, true)
	if err != nil {
		return err
	}
	instID := c.String("id")
	if instID == "" {
		return xerrors.New("--id flag is required")
	}
	instidx := c.String("instidx")
	if instidx == "" {
		return xerrors.New("--instidx flag is required")
	}
	idx, err := strconv.ParseUint(instidx, 10, 64)
	if err != nil {
		return err
	}
	instIDBuf, err := hex.DecodeString(instID)
	if err != nil {
		return err
	}
	id := byzcoin.NewInstanceID(instIDBuf)
	cl.SyncSignerCounter()
	err = cl.AddSignatureToDeferredTx(id, idx)
	if err != nil {
		return err
	}
	fmt.Println("Succesfully added signature to deferred transaction")
	return bcadminlib.WaitPropagation(c, cl.Bcl)
}

func deferredExec(c *cli.Context) error {
	cl, err := getClient(c, true)
	if err != nil {
		return err
	}
	instID := c.String("id")
	if instID == "" {
		return xerrors.New("--id flag is required")
	}
	instIDBuf, err := hex.DecodeString(instID)
	if err != nil {
		return err
	}
	id := byzcoin.NewInstanceID(instIDBuf)
	cl.SyncSignerCounter()
	err = cl.ExecDeferredTx(id)
	if err != nil {
		return err
	}
	fmt.Println("Succesfully executed the deferred transaction")
	return bcadminlib.WaitPropagation(c, cl.Bcl)
}

func projectCreate(c *cli.Context) error {
	cl, err := getClient(c, true)
	if err != nil {
		return err
	}

	adidstring := c.String("adid")
	if adidstring == "" {
		return xerrors.New("--adid flag is required")
	}

	adid, err := lib.StringToDarcID(adidstring)
	if err != nil {
		return xerrors.Errorf("failed to parse darc: %v", err)
	}

	pname := c.String("pname")
	if pname == "" {
		return xerrors.New("--pname flag is required")
	}

	cl.SyncSignerCounter()
	defId, _, pdarcID, err := cl.CreateNewProject(adid, pname)
	if err != nil {
		return err
	}
	fmt.Println("Deferred transaction spawned with ID:")
	fmt.Println(defId)
	fmt.Println("Project darc ID: ", pdarcID)
	return bcadminlib.WaitPropagation(c, cl.Bcl)
}

// func projectCreateAccessRight(c *cli.Context) error {
// 	cl, err := getClient(c, true)
// 	if err != nil {
// 		return err
// 	}

// 	adidstring := c.String("adid")
// 	if adidstring == "" {
// 		return xerrors.New("--adid flag is required")
// 	}

// 	adid, err := lib.StringToDarcID(adidstring)
// 	if err != nil {
// 		return xerrors.Errorf("failed to parse darc: %v", err)
// 	}

// 	pdidstring := c.String("pdid")
// 	if pdidstring == "" {
// 		return xerrors.New("--pdid flag is required")
// 	}

// 	pdid, err := lib.StringToDarcID(pdidstring)
// 	if err != nil {
// 		return xerrors.Errorf("failed to parse darc: %v", err)
// 	}

// 	cl.SyncSignerCounter()
// 	id, err := cl.CreateAccessRight(pdid, adid)
// 	if err != nil {
// 		return err
// 	}
// 	fmt.Println("Deferred transaction spawned with ID:")
// 	fmt.Println(id)
// 	return bcadminlib.WaitPropagation(c, cl.Bcl)
// }

func getExecResult(c *cli.Context) error {
	cl, err := getClient(c, true)
	if err != nil {
		return err
	}

	instID := c.String("id")
	if instID == "" {
		return xerrors.New("--id flag is required")
	}
	instIDBuf, err := hex.DecodeString(instID)
	if err != nil {
		return err
	}
	id := byzcoin.NewInstanceID(instIDBuf)

	finalID, err := cl.GetContractInstanceID(id, 0)
	if err != nil {
		return err
	}
	fmt.Println("Instance ID after execution:")
	fmt.Println(finalID)
	return bcadminlib.WaitPropagation(c, cl.Bcl)
}

func attach(c *cli.Context) error {
	cl, err := getClient(c, true)
	if err != nil {
		return err
	}

	instID := c.String("id")
	if instID == "" {
		return xerrors.New("--id flag is required")
	}
	instIDBuf, err := hex.DecodeString(instID)
	if err != nil {
		return err
	}
	id := byzcoin.NewInstanceID(instIDBuf)
	cl.SyncSignerCounter()
	err = cl.AttachAccessRightToProject(id)
	if err != nil {
		return err
	}
	fmt.Println("Successfully attached accessright contract instance to project darc with name resolution AR")
	return bcadminlib.WaitPropagation(c, cl.Bcl)
}

// func addQuerier(c *cli.Context) error {
// 	cl, err := getClient(c, true)
// 	if err != nil {
// 		return err
// 	}

// 	pdidstring := c.String("pdid")
// 	if pdidstring == "" {
// 		return xerrors.New("--pdid flag is required")
// 	}

// 	pdid, err := lib.StringToDarcID(pdidstring)
// 	if err != nil {
// 		return xerrors.Errorf("failed to parse darc: %v", err)
// 	}
// 	adidstring := c.String("adid")
// 	if adidstring == "" {
// 		return xerrors.New("--adid flag is required")
// 	}

// 	qid := c.String("qid")
// 	if qid == "" {
// 		return xerrors.New("--qid flag is required")
// 	}

// 	access := c.String("access")
// 	if access == "" {
// 		return xerrors.New("--qid flag is required")
// 	}

// 	adid, err := lib.StringToDarcID(adidstring)
// 	if err != nil {
// 		return xerrors.Errorf("failed to parse darc: %v", err)
// 	}

// 	cl.SyncSignerCounter()
// 	id, err := cl.AddQuerierToProject(pdid, adid, qid, access)
// 	if err != nil {
// 		return err
// 	}
// 	fmt.Println("Deferred transaction spawned with ID:")
// 	fmt.Println(id)
// 	return bcadminlib.WaitPropagation(c, cl.Bcl)
// }

// func modifyQuerier(c *cli.Context) error {
// 	cl, err := getClient(c, true)
// 	if err != nil {
// 		return err
// 	}

// 	pdidstring := c.String("pdid")
// 	if pdidstring == "" {
// 		return xerrors.New("--pdid flag is required")
// 	}

// 	pdid, err := lib.StringToDarcID(pdidstring)
// 	if err != nil {
// 		return xerrors.Errorf("failed to parse darc: %v", err)
// 	}
// 	adidstring := c.String("adid")
// 	if adidstring == "" {
// 		return xerrors.New("--adid flag is required")
// 	}

// 	qid := c.String("qid")
// 	if qid == "" {
// 		return xerrors.New("--qid flag is required")
// 	}

// 	access := c.String("access")
// 	if access == "" {
// 		return xerrors.New("--access flag is required")
// 	}

// 	adid, err := lib.StringToDarcID(adidstring)
// 	if err != nil {
// 		return xerrors.Errorf("failed to parse darc: %v", err)
// 	}

// 	cl.SyncSignerCounter()
// 	id, err := cl.ModifyQuerierAccessRightsForProject(pdid, adid, qid, access)
// 	if err != nil {
// 		return err
// 	}
// 	fmt.Println("Deferred transaction spawned with ID:")
// 	fmt.Println(id)
// 	return bcadminlib.WaitPropagation(c, cl.Bcl)
// }

// func removeQuerier(c *cli.Context) error {
// 	cl, err := getClient(c, true)
// 	if err != nil {
// 		return err
// 	}

// 	pdidstring := c.String("pdid")
// 	if pdidstring == "" {
// 		return xerrors.New("--pdid flag is required")
// 	}

// 	pdid, err := lib.StringToDarcID(pdidstring)
// 	if err != nil {
// 		return xerrors.Errorf("failed to parse darc: %v", err)
// 	}
// 	adidstring := c.String("adid")
// 	if adidstring == "" {
// 		return xerrors.New("--adid flag is required")
// 	}

// 	qid := c.String("qid")
// 	if qid == "" {
// 		return xerrors.New("--qid flag is required")
// 	}

// 	adid, err := lib.StringToDarcID(adidstring)
// 	if err != nil {
// 		return xerrors.Errorf("failed to parse darc: %v", err)
// 	}

// 	cl.SyncSignerCounter()
// 	id, err := cl.RemoveQuerierFromProject(pdid, adid, qid)
// 	if err != nil {
// 		return err
// 	}
// 	fmt.Println("Deferred transaction spawned with ID:")
// 	fmt.Println(id)
// 	return bcadminlib.WaitPropagation(c, cl.Bcl)
// }

// func verifyAccess(c *cli.Context) error {
// 	cl, err := getClient(c, true)
// 	if err != nil {
// 		return err
// 	}

// 	pdidstring := c.String("pdid")
// 	if pdidstring == "" {
// 		return xerrors.New("--pdid flag is required")
// 	}

// 	pdid, err := lib.StringToDarcID(pdidstring)
// 	if err != nil {
// 		return xerrors.Errorf("failed to parse darc: %v", err)
// 	}

// 	qid := c.String("qid")
// 	if qid == "" {
// 		return xerrors.New("--qid flag is required")
// 	}

// 	access := c.String("access")
// 	if access == "" {
// 		return xerrors.New("--qid flag is required")
// 	}

// 	cl.SyncSignerCounter()
// 	bool, err := cl.VerifyAccessRights(qid, access, pdid)
// 	if err != nil {
// 		return err
// 	}
// 	fmt.Println("Access status for ", qid, " for access: ", access)
// 	if bool {
// 		fmt.Println("Granted")
// 	} else {
// 		fmt.Println("Denied")
// 	}
// 	return bcadminlib.WaitPropagation(c, cl.Bcl)
// }

// func showAccess(c *cli.Context) error {
// 	cl, err := getClient(c, true)
// 	if err != nil {
// 		return err
// 	}

// 	pdidstring := c.String("pdid")
// 	if pdidstring == "" {
// 		return xerrors.New("--pdid flag is required")
// 	}

// 	pdid, err := lib.StringToDarcID(pdidstring)
// 	if err != nil {
// 		return xerrors.Errorf("failed to parse darc: %v", err)
// 	}

// 	qid := c.String("qid")
// 	if qid == "" {
// 		return xerrors.New("--qid flag is required")
// 	}

// 	cl.SyncSignerCounter()
// 	accessString, err := cl.ShowAccessRights(qid, pdid)
// 	if err != nil {
// 		return err
// 	}
// 	fmt.Println("Access status for", qid)
// 	fmt.Println(accessString)
// 	return bcadminlib.WaitPropagation(c, cl.Bcl)
// }

func deferredGet(c *cli.Context) error {
	cl, err := getClient(c, true)
	if err != nil {
		return err
	}
	instID := c.String("id")
	if instID == "" {
		return xerrors.New("--id flag is required")
	}
	instIDBuf, err := hex.DecodeString(instID)
	if err != nil {
		return err
	}
	id := byzcoin.NewInstanceID(instIDBuf)
	tx, err := cl.Bcl.GetDeferredData(id)
	fmt.Println("Transaction", id, ":")
	fmt.Println(tx.ProposedTransaction)
	return nil
}
