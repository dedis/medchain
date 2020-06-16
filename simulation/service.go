package main

import (
	"time"

	"github.com/BurntSushi/toml"
	medchain "github.com/medchain/services"
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
	Queries       int
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
	log.Lvl2("Size is:", size, "rounds:", s.Rounds, "queries:", s.Queries)
	signer := darc.NewSignerEd25519(nil, nil)

	// Create the ledger
	gm, err := byzcoin.DefaultGenesisMsg(byzcoin.CurrentVersion, config.Roster,
		[]string{"spawn:" + medchain.ContractName, "invoke:" + medchain.ContractName + "." + "update", "invoke:" + medchain.ContractName + "." + "verifystatus", "_name:" + medchain.ContractName, "spawn:deferred", "invoke:deferred.addProof",
			"invoke:deferred.execProposedTx"}, signer.Identity())
	if err != nil {
		return xerrors.Errorf("couldn't setup genesis message: %v", err)
	}

	// Set block interval from the simulation config.
	blockInterval, err := time.ParseDuration(s.BlockInterval)
	if err != nil {
		return xerrors.Errorf("parse duration of BlockInterval failed: %v", err)
	}
	gm.BlockInterval = blockInterval

	bcl, _, err := byzcoin.NewLedger(gm, s.Keep)
	if err != nil {
		return xerrors.Errorf("couldn't create genesis block: %v", err)
	}
	if err = bcl.UseNode(0); err != nil {
		return xerrors.Errorf("couldn't use the node: %v", err)
	}

	// Initialize MedChain client
	genDarc := gm.GenesisDarc
	cl, err := medchain.NewClient(bcl, config.Server.ServerIdentity, "1", signer) //TODO: or maybe use randomServerIdentity
	if err != nil {
		return xerrors.Errorf("couldn't start the client: %v", err)
	}

	cl.Signers = []darc.Signer{signer}
	cl.GenDarc = &genDarc
	log.Lvl1("Starting simulation tests")
	log.Lvl1("Bootstrapping the client")

	err = cl.Create()
	if err != nil {
		return xerrors.Errorf("couldn't start the client: %v", err)
	}

	// ------------------------------------------------------------------------
	// 1. Add Project A Darc
	// ------------------------------------------------------------------------

	rulesA := darc.InitRules([]darc.Identity{signer.Identity()}, []darc.Identity{cl.Signers[0].Identity()})
	actionsAAnd := "invoke:medchain.patient_list,invoke:medchain.count_per_site,invoke:medchain.count_per_site_obfuscated," +
		"invoke:medchain.count_per_site_shuffled,invoke:medchain.count_per_site_shuffled_obfuscated,invoke:medchain.count_global," +
		"invoke:medchain.count_global_obfuscated,invoke:darc.evolve"

	actionsAOr := "spawn:deferred,invoke:deferred.addProof,invoke:deferred.execProposedTx,spawn:medchain,invoke:medchain.update,_name:deferred,spawn:naming,_name:medchain,spawn:value,invoke:value.update,_name:value"

	// all signers need to sign
	exprAAnd := expression.InitAndExpr(cl.Signers[0].Identity().String())

	// at least one signer need to sign
	exprAOr := expression.InitOrExpr(cl.Signers[0].Identity().String())
	cl.AllDarcs["A"], err = cl.CreateDarc("Project A darc", rulesA, actionsAAnd, actionsAOr, exprAAnd, exprAOr)
	if err != nil {
		return xerrors.Errorf("couldn't create darc: %v", err)
	}

	aDarcBuf, err := cl.AllDarcs["A"].ToProto()
	if err != nil {
		return xerrors.Errorf("couldn't add darc: %v", err)
	}

	ctx, err := cl.Bcl.CreateTransaction(byzcoin.Instruction{
		InstanceID: byzcoin.NewInstanceID(cl.GenDarcID),
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
		return xerrors.Errorf("couldn't add darc: %v", err)
	}
	err = ctx.FillSignersAndSignWith(cl.Signers...)
	if err != nil {
		return xerrors.Errorf("couldn't sign transaction: %v", err)
	}
	_, err = cl.Bcl.AddTransactionAndWait(ctx, 10)
	if err != nil {
		return xerrors.Errorf("couldn't add transaction: %v", err)
	}

	cl.AllDarcIDs["A"] = cl.AllDarcs["A"].GetBaseID()

	// ------------------------------------------------------------------------
	// 1. Add Project B Darc
	// ------------------------------------------------------------------------
	// signer can only query certain things from the database
	rulesB := darc.InitRules([]darc.Identity{signer.Identity()}, []darc.Identity{cl.Signers[0].Identity()})
	// user cannot query patient_list
	actionsBAnd := "invoke:medchain.update,invoke:medchain.count_per_site,invoke:medchain.count_per_site_obfuscated," +
		"invoke:medchain.count_per_site_shuffled,invoke:medchain.count_per_site_shuffled_obfuscated,invoke:medchain.count_global," +
		"invoke:medchain.count_global_obfuscated,_name:value"

	actionsBOr := "invoke:darc.evolve,spawn:deferred,invoke:deferred.addProof,invoke:deferred.execProposedTx,spawn:medchain,_name:deferred,spawn:naming,_name:medchain,spawn:darc,spawn:value,invoke:value.update"

	// all signers need to sign
	exprBAnd := expression.InitAndExpr(cl.Signers[0].Identity().String())

	// at least one signer need to sign
	exprBOr := expression.InitOrExpr(cl.Signers[0].Identity().String())
	cl.AllDarcs["B"], _ = cl.CreateDarc("Project B darc", rulesB, actionsBAnd, actionsBOr, exprBAnd, exprBOr)

	bDarcBuf, err := cl.AllDarcs["B"].ToProto()
	if err != nil {
		return xerrors.Errorf("couldn't add darc: %v", err)
	}

	ctx, err = cl.Bcl.CreateTransaction(byzcoin.Instruction{
		InstanceID: byzcoin.NewInstanceID(cl.GenDarcID),
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
		return xerrors.Errorf("couldn't add darc: %v", err)
	}

	err = ctx.FillSignersAndSignWith(cl.Signers...)
	if err != nil {
		return xerrors.Errorf("couldn't sign tx: %v", err)
	}
	_, err = cl.Bcl.AddTransactionAndWait(ctx, 10)
	if err != nil {
		return xerrors.Errorf("couldn't add tx: %v", err)
	}
	cl.AllDarcIDs["B"] = cl.AllDarcs["B"].GetBaseID()

	// ------------------------------------------------------------------------
	// 2. Spwan instances of MedChain contract (query) and check authorizations - in this case the query is authorized (scenario 1)
	// ------------------------------------------------------------------------
	charset := "abcdefghijklmnopqrstuvwxyz" +
		"ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	qIDAuth := make([]string, s.Queries)
	for i := 0; i < s.Queries; i++ {
		qIDAuth[i] = randomString(10, charset) + ":" + randomString(1, "AB") + ":" + randomAction()
	}

	qIDRej := make([]string, s.Queries)
	for i := 0; i < s.Queries; i++ {
		qIDRej[i] = randomString(10, charset) + ":" + randomString(1, "AB") + ":" + "patient_list"
	}

	for round := 0; round < s.Rounds; round++ {
		log.Lvl1("Starting round", round)
		roundM := monitor.NewTimeMeasure("round")

		queries := s.Queries
		log.Lvlf1("Sending %d queries", s.Queries)
		var allIDs []byzcoin.InstanceID

		for q := 0; q < queries; q++ {
			//prepare := monitor.NewTimeMeasure("prepare")
			authorize := monitor.NewTimeMeasure("authorize")
			req1 := &medchain.AddQueryRequest{}
			query := medchain.NewQuery(qIDAuth[q], "Submitted")
			req1.QueryID = query.ID
			req1.ClientID = cl.ClientID
			req1.DarcID = cl.AllDarcs["A"].GetBaseID()
			//prepare.Record()

			resp1, err := cl.SpawnQuery(req1)
			if err != nil {
				return xerrors.Errorf("could not spawn query: %v", err)
			}
			if !resp1.OK {
				return xerrors.New("error in spawn query")
			}
			allIDs = append(allIDs, resp1.QueryInstID)
			authorize.Record()

			req2 := &medchain.SignDeferredTxRequest{}
			req2.Keys = cl.Signers[0]
			req2.ClientID = cl.ClientID
			req2.QueryInstID = resp1.QueryInstID
			signT := monitor.NewTimeMeasure("sign_deferred")
			resp2, err := cl.AddSignatureToDeferredQuery(req2)
			if err != nil {
				return xerrors.Errorf("could not sign deferred query: %v", err)
			}
			if !resp2.OK {
				return xerrors.New("error in signing the query")
			}
			signT.Record()
		}
		for q := 0; q < queries; q++ {
			execT := monitor.NewTimeMeasure("execute_deferred")
			req3 := &medchain.ExecuteDeferredTxRequest{}
			req3.ClientID = cl.ClientID
			req3.QueryInstID = allIDs[q]
			resp3, err := cl.ExecDefferedQuery(req3)
			if err != nil {
				return xerrors.Errorf("could not execute deferred query: %v", err)
			}
			if !resp3.OK {
				return xerrors.New("error in signing the query")
			}
			execT.Record()
		}

		for q := 0; q < queries; q++ {
			reject := monitor.NewTimeMeasure("reject")
			req4 := &medchain.AddQueryRequest{}
			query := medchain.NewQuery(qIDRej[q], "Submitted")
			req4.QueryID = query.ID
			req4.ClientID = cl.ClientID
			req4.DarcID = cl.AllDarcs["B"].GetBaseID()

			resp1, err := cl.SpawnQuery(req4)
			if err != nil {
				return xerrors.Errorf("could not spawn query: %v", err)
			}
			if !resp1.OK {
				return xerrors.New("error in spawn query")
			}

			reject.Record()
		}

		// The AddTransactionAndWait returns as soon as the transaction is included in the node, but
		// it doesn't wait until the transaction is included in all nodes. Thus this wait for
		// the new block to be propagated.
		// time.Sleep(time.Second)
		roundM.Record()

		// This sleep is needed to wait for the propagation to finish
		// on all the nodes. Otherwise the simulation manager
		// (runsimul.go in onet) might close some nodes and cause
		// skipblock propagation to fail.
		time.Sleep(blockInterval)
	}

	// This sleep is needed to wait for the propagation to finish
	// on all the nodes. Otherwise the simulation manager
	// (runsimul.go in onet) might close some nodes and cause
	// skipblock propagation to fail.
	time.Sleep(blockInterval)

	return nil
}
