package main

import (
	"time"

	"github.com/BurntSushi/toml"
	medchain "github.com/medchain/client"
	"go.dedis.ch/cothority/v3/byzcoin"
	"go.dedis.ch/cothority/v3/darc"
	"go.dedis.ch/cothority/v3/darc/expression"
	"go.dedis.ch/onet/v3"
	"go.dedis.ch/onet/v3/log"
	"go.dedis.ch/onet/v3/simul/monitor"
	"golang.org/x/xerrors"
)

/*
 * Defines the simulation for the service-medchain
 TODO: remove from byzcoin simulation/coins
*/

func init() {
	onet.SimulationRegister("MedChainService", NewSimulationService)
}

// SimulationService only holds the BFTree simulation
type SimulationService struct {
	onet.SimulationBFTree
	Transactions  int
	BlockInterval string
	BatchSize     int
	Keep          bool
	Delay         int
}

// NewSimulationService returns the new simulation, where all fields are
// initialised using the config-file
func NewSimulationService(config string) (onet.Simulation, error) {
	es := &SimulationService{}
	_, err := toml.Decode(config, es)
	if err != nil {
		return nil, err
	}
	return es, nil
}

// Setup creates the tree used for that simulation
func (s *SimulationService) Setup(dir string, hosts []string) (
	*onet.SimulationConfig, error) {
	sc := &onet.SimulationConfig{}
	s.CreateRoster(sc, hosts, 2000)
	err := s.CreateTree(sc)
	if err != nil {
		return nil, err
	}
	return sc, nil
}

// Node can be used to initialize each node before it will be run
// by the server. Here we call the 'Node'-method of the
// SimulationBFTree structure which will load the roster- and the
// tree-structure to speed up the first round.
func (s *SimulationService) Node(config *onet.SimulationConfig) error {
	index, _ := config.Roster.Search(config.Server.ServerIdentity.ID)
	if index < 0 {
		log.Fatal("Didn't find this node in roster")
	}
	log.Lvl3("Initializing node-index", index)
	return s.SimulationBFTree.Node(config)
}

// Run is used on the destination machines and runs a number of
// rounds
func (s *SimulationService) Run(config *onet.SimulationConfig) error {
	size := config.Tree.Size()
	log.Lvl2("Size is:", size, "rounds:", s.Rounds, "transactions:", s.Transactions)
	signer := darc.NewSignerEd25519(nil, nil)

	// Create the ledger
	req, err := byzcoin.DefaultGenesisMsg(byzcoin.CurrentVersion, config.Roster,
		[]string{"spawn:" + medchain.ContractName, "invoke:" + medchain.ContractName + "." + "update", "invoke:" + medchain.ContractName + "." + "verifystatus", "_name:" + medchain.ContractName}, signer.Identity())
	if err != nil {
		return xerrors.Errorf("couldn't setup genesis message: %v", err)
	}

	// Set block interval from the simulation config.
	blockInterval, err := time.ParseDuration(s.BlockInterval)
	if err != nil {
		return xerrors.Errorf("parse duration of BlockInterval failed: %v", err)
	}
	req.BlockInterval = blockInterval

	c, _, err := byzcoin.NewLedger(req, s.Keep)
	if err != nil {
		return xerrors.Errorf("couldn't create genesis block: %v", err)
	}
	if err = c.UseNode(0); err != nil {
		return xerrors.Errorf("couldn't use the node: %v", err)
	}

	// Initialize MedChain client
	genDarc := req.GenesisDarc
	cl := medchain.NewClient(c)
	cl.DarcID = genDarc.GetBaseID()
	cl.Signers = []darc.Signer{signer}
	cl.GenDarc = &genDarc
	log.Lvl1("Starting simulation tests")

	log.Lvl1("Bootstrapping the client")
	err = cl.Create()
	if err != nil {
		return xerrors.Errorf("couldn't start the client: %v", err)
	}

	// ------------------------------------------------------------------------
	// 1.1 Add Project A Darc
	// ------------------------------------------------------------------------

	// Measure the time it takes to add the darc every project as well as all project darcs
	addDarcA := monitor.NewTimeMeasure("addDarcA")
	log.Lvl1("Adding Darc of project A")

	addProjectDarcs := monitor.NewTimeMeasure("addProjectDarcs")
	rulesA := darc.InitRules([]darc.Identity{signer.Identity()}, []darc.Identity{cl.Signers[0].Identity()})
	actionsA := "spawn:medchain,invoke:medchain.update,invoke:medchain.patient_list,invoke:medchain.count_per_site,invoke:medchain.count_per_site_obfuscated," +
		"invoke:medchain.count_per_site_shuffled,invoke:medchain.count_per_site_shuffled_obfuscated,invoke:medchain.count_global," +
		"invoke:v.count_global_obfuscated"
	exprA := expression.InitOrExpr(cl.Signers[0].Identity().String())
	cl.AllDarcs["A"], _ = cl.CreateDarc("Project A darc", rulesA, actionsA, exprA)

	// Add _name to Darc rule so that we can name the instances using contract_name
	cl.AllDarcs["A"].Rules.AddRule("_name:"+medchain.ContractName, exprA)
	cl.AllDarcs["A"].Rules.AddRule("spawn:naming", exprA)
	// Verify the darc is correct
	err = cl.AllDarcs["A"].Verify(true)
	if err != nil {
		return xerrors.Errorf("could not verify the darc: %v", err)
	}

	aDarcBuf, err := cl.AllDarcs["A"].ToProto()
	if err != nil {
		return err
	}

	ctx, err := cl.ByzCoin.CreateTransaction(byzcoin.Instruction{
		InstanceID: byzcoin.NewInstanceID(cl.GenDarc.GetBaseID()),
		Spawn: &byzcoin.Spawn{
			ContractID: byzcoin.ContractDarcID,
			Args: byzcoin.Arguments{
				{
					Name:  "darc",
					Value: aDarcBuf,
				},
			},
		},
		SignerIdentities: []darc.Identity{cl.Signers[0].Identity()},
		SignerCounter:    cl.IncrementCtrs(),
	})
	if err != nil {
		return err
	}

	err = ctx.FillSignersAndSignWith(cl.Signers...)
	if err != nil {
		return err
	}

	_, err = cl.ByzCoin.AddTransactionAndWait(ctx, 10)
	if err != nil {
		return err
	}
	cl.AllDarcIDs["A"] = cl.AllDarcs["A"].GetBaseID()
	addDarcA.Record()

	// ------------------------------------------------------------------------
	// 1.2 Add Project B Darc
	// ------------------------------------------------------------------------
	// Measure the time it takes to add the darc every project as well as all project darcs
	addDarcB := monitor.NewTimeMeasure("addDarcB")
	log.Lvl1("Adding the Darc for project B")
	// signer can only query certain things from the database
	rulesB := darc.InitRules([]darc.Identity{signer.Identity()}, []darc.Identity{cl.Signers[0].Identity()})
	actionsB := "spawn:medchain,invoke:medchain.update,invoke:medchain.count_global,invoke:medchain.count_global_obfuscated"
	exprB := expression.InitOrExpr(cl.Signers[0].Identity().String())
	cl.AllDarcs["B"], _ = cl.CreateDarc("Project B darc", rulesB, actionsB, exprB)

	// Add _name to Darc rule so that we can name the instances using contract_name
	cl.AllDarcs["B"].Rules.AddRule("_name:"+medchain.ContractName, exprB)
	cl.AllDarcs["B"].Rules.AddRule("spawn:naming", exprB)

	// Verify the darc is correct
	err = cl.AllDarcs["B"].Verify(true)
	if err != nil {
		return err
	}

	bDarcBuf, err := cl.AllDarcs["B"].ToProto()
	if err != nil {
		return err
	}

	ctx, err = cl.ByzCoin.CreateTransaction(byzcoin.Instruction{
		InstanceID: byzcoin.NewInstanceID(cl.GenDarc.GetBaseID()),
		Spawn: &byzcoin.Spawn{
			ContractID: byzcoin.ContractDarcID,
			Args: byzcoin.Arguments{
				{
					Name:  "darc",
					Value: bDarcBuf,
				},
			},
		},
		SignerIdentities: []darc.Identity{cl.Signers[0].Identity()},
		SignerCounter:    cl.IncrementCtrs(),
	})
	if err != nil {
		return err
	}

	err = ctx.FillSignersAndSignWith(cl.Signers...)
	if err != nil {
		return err
	}

	_, err = cl.ByzCoin.AddTransactionAndWait(ctx, 10)
	if err != nil {
		return err
	}
	cl.AllDarcIDs["B"] = cl.AllDarcs["B"].GetBaseID()
	addDarcB.Record()
	addProjectDarcs.Record()

	// ------------------------------------------------------------------------
	// 2. Spwan query instances
	// ------------------------------------------------------------------------

	// Measure the time it takes to spawn a query and authorize a query
	spawnQuery := monitor.NewTimeMeasure("spawnQuery")
	authQuery := monitor.NewTimeMeasure("authQuery")
	log.Lvl1("Spawning the query")

	queries, ids, err := cl.SpawnQuery(medchain.NewQuery("wsdf65k80h:A:patient_list", "Submitted"))
	if err != nil {
		return err
	}
	spawnQuery.Record()

	// ------------------------------------------------------------------------
	// 3. Check Authorizations
	// ------------------------------------------------------------------------
	log.Lvl1("Authorizing the query")

	queries, ids, err = cl.WriteQueries(ids, queries...)
	authQuery.Record()

	// // Measure the time for ResolveInstanceID
	// resolveID := monitor.NewTimeMeasure("resolveID")
	// _, err = cl.ByzCoin.ResolveInstanceID(cl.AllDarcIDs["B"], queries[0].ID)
	// if err != nil {
	// 	return err
	// }
	// resolveID.Record()

	// This sleep is needed to wait for the propagation to finish
	// on all the nodes. Otherwise the simulation manager
	// (runsimul.go in onet) might close some nodes and cause
	// skipblock propagation to fail.
	time.Sleep(blockInterval)

	return nil
}
