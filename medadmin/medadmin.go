package main

import (
	"encoding/hex"
	"errors"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"time"

	admin "github.com/medchain/admin"
	cli "github.com/urfave/cli"
	"go.dedis.ch/cothority/v3"
	"go.dedis.ch/cothority/v3/byzcoin"
	"go.dedis.ch/cothority/v3/byzcoin/bcadmin/clicontracts"
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

type config struct {
	Name    string
	QueryID byzcoin.InstanceID //
}

type bcConfig struct {
	Roster    onet.Roster
	ByzCoinID skipchain.SkipBlockID
}

var cliApp = cli.NewApp()
var dataDir = ""
var gitTag = "dev" //TODO change

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

	// // get the private key from the cmdline.
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

func spawn(c *cli.Context) error {
	cl, err := getClient(c, true)
	if err != nil {
		return err
	}
	cl.SyncSignerCounter()
	// TODO broadcast the base ID
	darc, err := cl.SpawnNewAdminDarc()
	if err != nil {
		return xerrors.Errorf("spawning a new Admin Darc: %w", err)
	}
	fmt.Println("New admininistration darc spawned :")
	fmt.Println(darc)
	return bcadminlib.WaitPropagation(c, cl.Bcl)
}

func create(c *cli.Context) error {
	keys := darc.NewSignerEd25519(nil, nil)
	fmt.Println("New admin identity key pair created :")
	fmt.Println(keys.Identity().String())
	err := bcadminlib.SaveKey(keys)
	return err
}

func addAdmin(c *cli.Context) error {
	cl, err := getClient(c, true)
	if err != nil {
		return err
	}
	id := c.String("identity")
	if id == "" {
		return xerrors.New("--identity flag is required")
	}
	adidstring := c.String("adid")
	if adidstring == "" {
		return xerrors.New("--adid flag is required")
	}

	adid, err := lib.StringToDarcID(adidstring)
	if err != nil {
		return xerrors.Errorf("failed to parse darc: %v", err)
	}

	cl.SyncSignerCounter()
	// TODO broadcast the base ID
	deferredId, err := cl.AddAdminToAdminDarc(adid, id)
	if err != nil {
		return xerrors.Errorf("spawning a new Admin Darc: %w", err)
	}
	fmt.Println("Deffered transaction spawned with ID:")
	fmt.Println(deferredId)
	return bcadminlib.WaitPropagation(c, cl.Bcl)
}
func removeAdmin(c *cli.Context) error {
	cl, err := getClient(c, true)
	if err != nil {
		return err
	}
	id := c.String("identity")
	if id == "" {
		return xerrors.New("--identity flag is required")
	}
	adidstring := c.String("adid")
	if adidstring == "" {
		return xerrors.New("--adid flag is required")
	}

	adid, err := lib.StringToDarcID(adidstring)
	if err != nil {
		return xerrors.Errorf("failed to parse darc: %v", err)
	}

	cl.SyncSignerCounter()
	// TODO broadcast the base ID
	darc, err := cl.RemoveAdminFromAdminDarc(adid, id)
	if err != nil {
		return xerrors.Errorf("spawning a new Admin Darc: %w", err)
	}
	fmt.Println("Deffered transaction spawned with ID:")
	fmt.Println(darc)
	return bcadminlib.WaitPropagation(c, cl.Bcl)
}

func modifyAdminKey(c *cli.Context) error {
	cl, err := getClient(c, true)
	if err != nil {
		return err
	}
	newKey := c.String("newkey")
	if newKey == "" {
		return xerrors.New("--identity flag is required")
	}
	oldKey := c.String("oldkey")
	if oldKey == "" {
		return xerrors.New("--identity flag is required")
	}
	adidstring := c.String("adid")
	if adidstring == "" {
		return xerrors.New("--adid flag is required")
	}

	adid, err := lib.StringToDarcID(adidstring)
	if err != nil {
		return xerrors.Errorf("failed to parse darc: %v", err)
	}

	cl.SyncSignerCounter()
	// TODO broadcast the base ID
	darc, err := cl.ModifyAdminKeysFromAdminDarc(adid, oldKey, newKey)
	if err != nil {
		return xerrors.Errorf("spawning a new Admin Darc: %w", err)
	}
	fmt.Println("Deffered transaction spawned with ID:")
	fmt.Println(darc)
	return bcadminlib.WaitPropagation(c, cl.Bcl)
}

func sync(c *cli.Context) error {
	cl, err := getClient(c, true)
	if err != nil {
		return err
	}
	ids, err := cl.FetchNewDefferedInstanceIDs()
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
	instIDBuf, err := hex.DecodeString(instID)
	if err != nil {
		return err
	}
	id := byzcoin.NewInstanceID(instIDBuf)
	cl.SyncSignerCounter()
	err = cl.AddSignatureToDefferedTx(id, 0) //TODO change index as argument
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
	id := byzcoin.NewInstanceID(instIDBuf)
	cl.SyncSignerCounter()
	err = cl.ExecDefferedTx(id)
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
	fmt.Println("Deffered transaction spawned with ID:")
	fmt.Println(defId)
	fmt.Println("Project darc ID: ", pdarcID)
	return bcadminlib.WaitPropagation(c, cl.Bcl)
}

func projectCreateAccessRight(c *cli.Context) error {
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

	pdidstring := c.String("pdid")
	if pdidstring == "" {
		return xerrors.New("--pdid flag is required")
	}

	pdid, err := lib.StringToDarcID(pdidstring)
	if err != nil {
		return xerrors.Errorf("failed to parse darc: %v", err)
	}

	cl.SyncSignerCounter()
	id, err := cl.CreateAccessRight(pdid, adid)
	if err != nil {
		return err
	}
	fmt.Println("Deffered transaction spawned with ID:")
	fmt.Println(id)
	return bcadminlib.WaitPropagation(c, cl.Bcl)
}

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
	id := byzcoin.NewInstanceID(instIDBuf)

	finalID, err := cl.GetAccessRightInstanceID(id)
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
	id := byzcoin.NewInstanceID(instIDBuf)
	cl.SyncSignerCounter()
	err = cl.AttachAccessRightToProject(id)
	if err != nil {
		return err
	}
	fmt.Println("Successfully attached accessright contract instance to project darc")
	return bcadminlib.WaitPropagation(c, cl.Bcl)
}

func addQuerier(c *cli.Context) error {
	cl, err := getClient(c, true)
	if err != nil {
		return err
	}

	pdidstring := c.String("pdid")
	if pdidstring == "" {
		return xerrors.New("--pdid flag is required")
	}

	pdid, err := lib.StringToDarcID(pdidstring)
	if err != nil {
		return xerrors.Errorf("failed to parse darc: %v", err)
	}
	adidstring := c.String("adid")
	if adidstring == "" {
		return xerrors.New("--adid flag is required")
	}

	qid := c.String("qid")
	if qid == "" {
		return xerrors.New("--qid flag is required")
	}

	access := c.String("access")
	if access == "" {
		return xerrors.New("--qid flag is required")
	}

	adid, err := lib.StringToDarcID(adidstring)
	if err != nil {
		return xerrors.Errorf("failed to parse darc: %v", err)
	}

	cl.SyncSignerCounter()
	id, err := cl.AddQuerierToProject(pdid, adid, qid, access)
	if err != nil {
		return err
	}
	fmt.Println("Deffered transaction spawned with ID:")
	fmt.Println(id)
	return bcadminlib.WaitPropagation(c, cl.Bcl)
}

func modifyQuerier(c *cli.Context) error {
	cl, err := getClient(c, true)
	if err != nil {
		return err
	}

	pdidstring := c.String("pdid")
	if pdidstring == "" {
		return xerrors.New("--pdid flag is required")
	}

	pdid, err := lib.StringToDarcID(pdidstring)
	if err != nil {
		return xerrors.Errorf("failed to parse darc: %v", err)
	}
	adidstring := c.String("adid")
	if adidstring == "" {
		return xerrors.New("--adid flag is required")
	}

	qid := c.String("qid")
	if qid == "" {
		return xerrors.New("--qid flag is required")
	}

	access := c.String("access")
	if access == "" {
		return xerrors.New("--access flag is required")
	}

	adid, err := lib.StringToDarcID(adidstring)
	if err != nil {
		return xerrors.Errorf("failed to parse darc: %v", err)
	}

	cl.SyncSignerCounter()
	id, err := cl.ModifyQuerierAccessRightsForProject(pdid, adid, qid, access)
	if err != nil {
		return err
	}
	fmt.Println("Deffered transaction spawned with ID:")
	fmt.Println(id)
	return bcadminlib.WaitPropagation(c, cl.Bcl)
}

func removeQuerier(c *cli.Context) error {
	cl, err := getClient(c, true)
	if err != nil {
		return err
	}

	pdidstring := c.String("pdid")
	if pdidstring == "" {
		return xerrors.New("--pdid flag is required")
	}

	pdid, err := lib.StringToDarcID(pdidstring)
	if err != nil {
		return xerrors.Errorf("failed to parse darc: %v", err)
	}
	adidstring := c.String("adid")
	if adidstring == "" {
		return xerrors.New("--adid flag is required")
	}

	qid := c.String("qid")
	if qid == "" {
		return xerrors.New("--qid flag is required")
	}

	adid, err := lib.StringToDarcID(adidstring)
	if err != nil {
		return xerrors.Errorf("failed to parse darc: %v", err)
	}

	cl.SyncSignerCounter()
	id, err := cl.RemoveQuerierFromProject(pdid, adid, qid)
	if err != nil {
		return err
	}
	fmt.Println("Deffered transaction spawned with ID:")
	fmt.Println(id)
	return bcadminlib.WaitPropagation(c, cl.Bcl)
}

func verifyAccess(c *cli.Context) error {
	cl, err := getClient(c, true)
	if err != nil {
		return err
	}

	pdidstring := c.String("pdid")
	if pdidstring == "" {
		return xerrors.New("--pdid flag is required")
	}

	pdid, err := lib.StringToDarcID(pdidstring)
	if err != nil {
		return xerrors.Errorf("failed to parse darc: %v", err)
	}

	qid := c.String("qid")
	if qid == "" {
		return xerrors.New("--qid flag is required")
	}

	access := c.String("access")
	if access == "" {
		return xerrors.New("--qid flag is required")
	}

	cl.SyncSignerCounter()
	bool, err := cl.VerifyAccessRights(qid, access, pdid)
	if err != nil {
		return err
	}
	fmt.Println("Access status for ", qid, " for access: ", access)
	if bool {
		fmt.Println("Granted")
	} else {
		fmt.Println("Denied")
	}
	return bcadminlib.WaitPropagation(c, cl.Bcl)
}
func showAccess(c *cli.Context) error {
	cl, err := getClient(c, true)
	if err != nil {
		return err
	}

	pdidstring := c.String("pdid")
	if pdidstring == "" {
		return xerrors.New("--pdid flag is required")
	}

	pdid, err := lib.StringToDarcID(pdidstring)
	if err != nil {
		return xerrors.Errorf("failed to parse darc: %v", err)
	}

	qid := c.String("qid")
	if qid == "" {
		return xerrors.New("--qid flag is required")
	}

	cl.SyncSignerCounter()
	accessString, err := cl.ShowAccessRights(qid, pdid)
	if err != nil {
		return err
	}
	fmt.Println("Access status for", qid)
	fmt.Println(accessString)
	return bcadminlib.WaitPropagation(c, cl.Bcl)
}

func deferredGet(c *cli.Context) error {
	return clicontracts.DeferredGet(c)
}
