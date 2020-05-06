package medchain

import (
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.dedis.ch/cothority/v3"
	"go.dedis.ch/cothority/v3/byzcoin"
	"go.dedis.ch/cothority/v3/darc"
	"go.dedis.ch/cothority/v3/darc/expression"
	"go.dedis.ch/cothority/v3/skipchain"
	"go.dedis.ch/kyber/v3/suites"
	"go.dedis.ch/onet/v3"
	"go.dedis.ch/onet/v3/log"
	"go.dedis.ch/protobuf"
)

var tSuite = suites.MustFind("Ed25519")

// Use this block interval for logic tests. Stress test often use a different
// block interval.
var testBlockInterval = 500 * time.Millisecond
var actionsList = "patient_list,count_per_site,count_per_site_obfuscated,count_per_site_shuffled,count_per_site_shuffled_obfuscated,count_global,count_global_obfuscated"

func TestMain(m *testing.M) {
	log.MainTest(m)
}

func TestClient_MedchainAuthorize(t *testing.T) {
	// ------------------------------------------------------------------------
	// 0. Set up and start service
	// ------------------------------------------------------------------------
	log.Lvl1("[INFO] Starting the service")
	t.Log("[INFO] Starting the service")
	s, c := newSer(t)
	leader := s.services[0]
	defer s.close()

	log.Lvl1("[INFO] Starting ByzCoin Client")
	err := c.Create()
	require.Nil(t, err)

	// ------------------------------------------------------------------------
	// 1. Add Project A Darc
	// ------------------------------------------------------------------------

	rulesA := darc.InitRules([]darc.Identity{s.owner.Identity()}, []darc.Identity{c.Signers[0].Identity()})
	actionsA := "spawn:medchain,invoke:medchain.update,invoke:medchain.patient_list,invoke:medchain.count_per_site,invoke:medchain.count_per_site_obfuscated," +
		"invoke:medchain.count_per_site_shuffled,invoke:medchain.count_per_site_shuffled_obfuscated,invoke:medchain.count_global," +
		"invoke:v.count_global_obfuscated"
	exprA := expression.InitOrExpr(c.Signers[0].Identity().String())
	c.AllDarcs["A"], _ = c.CreateDarc("Project A darc", rulesA, actionsA, exprA)

	// Add _name to Darc rule so that we can name the instances using contract_name
	c.AllDarcs["A"].Rules.AddRule("_name:"+ContractName, exprA)
	c.AllDarcs["A"].Rules.AddRule("spawn:naming", exprA)

	// Verify the darc is correct
	require.Nil(t, c.AllDarcs["A"].Verify(true))
	t.Logf("**************** Darc of Project A ******************")
	t.Log(c.AllDarcs["A"].String())

	aDarcBuf, err := c.AllDarcs["A"].ToProto()
	require.NoError(t, err)
	aDarcCopy, err := darc.NewFromProtobuf(aDarcBuf)
	require.NoError(t, err)
	require.True(t, c.AllDarcs["A"].Equal(aDarcCopy))

	ctx, err := c.ByzCoin.CreateTransaction(byzcoin.Instruction{
		InstanceID: byzcoin.NewInstanceID(s.genDarc.GetBaseID()),
		Spawn: &byzcoin.Spawn{
			ContractID: byzcoin.ContractDarcID,
			Args: byzcoin.Arguments{
				{
					Name:  "darc",
					Value: aDarcBuf,
				},
			},
		},
		SignerIdentities: []darc.Identity{c.Signers[0].Identity()},
		SignerCounter:    c.IncrementCtrs(),
	})
	require.Nil(t, err)

	err = ctx.FillSignersAndSignWith(c.Signers...)
	require.Nil(t, err)

	_, err = c.ByzCoin.AddTransactionAndWait(ctx, 10)
	require.Nil(t, err)
	c.AllDarcIDs["A"] = c.AllDarcs["A"].GetBaseID()

	//// *-*-*-*-*-*-*-*   Demo 1: Query should be Authorized *-*-*-*-*-*-*-*-*

	// ------------------------------------------------------------------------
	// 2. Spwan query instances
	// ------------------------------------------------------------------------

	//fmt.Println("[INFO] -*-*-*-*-*- DEMO 1 - AUTHORIZE -*-*-*-*-*-")
	log.Lvl1("[INFO] Spawning the query ")
	queries, ids, err := c.SpawnQuery(NewQuery("wsdf65k80h:A:patient_list", "Submitted"))
	require.Nil(t, err)
	require.Equal(t, 1, len(ids))
	require.Equal(t, 32, len(ids[0]))

	// Loop while we wait for the next block to be created.
	instaID, err := c.ByzCoin.ResolveInstanceID(c.AllDarcIDs["A"], queries[0].ID)
	require.Nil(t, err)
	waitForKey(t, leader.omni, c.ByzCoin.ID, instaID.Slice(), testBlockInterval)

	// Check consistency and # of queries.
	for i := 0; i < 10; i++ {
		leader.waitForBlock(c.ByzCoin.ID)
	}

	//Fetch the index, and check it.
	idx := checkProof(t, c, leader.omni, instaID.Slice(), c.ByzCoin.ID)
	qdata := QueryData{}
	err = protobuf.Decode(idx, &qdata)
	require.Nil(t, err)
	for _, s := range qdata.Storage {
		require.Equal(t, queries[0].ID, s.ID)
		require.Equal(t, queries[0].Status, (s.Status))
	}

	// Use the client API to get the query back
	// Resolve instance takes much time to run
	for _, query := range queries {
		instaID, err = c.ByzCoin.ResolveInstanceID(c.AllDarcIDs["A"], query.ID)
		require.Nil(t, err)
		_, err := c.GetQuery(instaID.Slice())
		require.Nil(t, err)
	}

	// ------------------------------------------------------------------------
	// 3. Check Authorizations
	// ------------------------------------------------------------------------
	log.Lvl1("[INFO] Query Authorization ")
	queries, ids, err = c.AddQuery(ids, queries...)

	require.Nil(t, err)
	require.Equal(t, 1, len(ids))
	require.Equal(t, 32, len(ids[0]))

	// Loop while we wait for the next block to be created.
	instaID, err = c.ByzCoin.ResolveInstanceID(c.AllDarcIDs["A"], queries[0].ID)
	require.Nil(t, err)
	waitForKey(t, leader.omni, c.ByzCoin.ID, instaID.Slice(), testBlockInterval)

	// Check consistency and # of queries.
	for i := 0; i < 10; i++ {
		leader.waitForBlock(c.ByzCoin.ID)
	}

	//Fetch the index, and check it.
	idx = checkProof(t, c, leader.omni, instaID.Slice(), c.ByzCoin.ID)
	qdata = QueryData{}
	err = protobuf.Decode(idx, &qdata)
	require.Nil(t, err)
	for _, s := range qdata.Storage {
		require.Equal(t, queries[0].ID, s.ID)
		require.Equal(t, "Authorized", s.Status)
	}

	// Use the client API to get the query back
	// Resolve instance takes much time to run
	for _, query := range queries {
		instaID, err = c.ByzCoin.ResolveInstanceID(c.AllDarcIDs["A"], query.ID)
		require.Nil(t, err)
		_, err := c.GetQuery(instaID.Slice())
		require.Nil(t, err)
	}
}

func TestClient_MedchainReject(t *testing.T) {
	// ------------------------------------------------------------------------
	// 0. Set up and start service
	// ------------------------------------------------------------------------
	log.Lvl1("[INFO] Starting the service")
	s, c := newSer(t)
	leader := s.services[0]
	defer s.close()

	log.Lvl1("[INFO] Starting ByzCoin Client")
	err := c.Create()
	require.Nil(t, err)

	// ------------------------------------------------------------------------
	// 1. Add Project B Darc
	// ------------------------------------------------------------------------
	// signer can only query certain things from the database
	rulesB := darc.InitRules([]darc.Identity{s.owner.Identity()}, []darc.Identity{c.Signers[0].Identity()})
	actionsB := "spawn:medchain,invoke:medchain.update,invoke:medchain.count_global,invoke:medchain.count_global_obfuscated"
	exprB := expression.InitOrExpr(c.Signers[0].Identity().String())
	c.AllDarcs["B"], _ = c.CreateDarc("Project B darc", rulesB, actionsB, exprB)

	// Add _name to Darc rule so that we can name the instances using contract_name
	c.AllDarcs["B"].Rules.AddRule("_name:"+ContractName, exprB)
	c.AllDarcs["B"].Rules.AddRule("spawn:naming", exprB)

	// Verify the darc is correct
	require.Nil(t, c.AllDarcs["B"].Verify(true))
	t.Logf("**************** Darc of Project B ******************")
	t.Log(c.AllDarcs["B"].String())

	bDarcBuf, err := c.AllDarcs["B"].ToProto()
	require.NoError(t, err)
	bDarcCopy, err := darc.NewFromProtobuf(bDarcBuf)
	require.NoError(t, err)
	require.True(t, c.AllDarcs["B"].Equal(bDarcCopy))

	ctx, err := c.ByzCoin.CreateTransaction(byzcoin.Instruction{
		InstanceID: byzcoin.NewInstanceID(s.genDarc.GetBaseID()),
		Spawn: &byzcoin.Spawn{
			ContractID: byzcoin.ContractDarcID,
			Args: byzcoin.Arguments{
				{
					Name:  "darc",
					Value: bDarcBuf,
				},
			},
		},
		SignerIdentities: []darc.Identity{c.Signers[0].Identity()},
		SignerCounter:    c.IncrementCtrs(),
	})
	require.Nil(t, err)

	err = ctx.FillSignersAndSignWith(c.Signers...)
	require.Nil(t, err)

	_, err = c.ByzCoin.AddTransactionAndWait(ctx, 10)
	require.Nil(t, err)
	c.AllDarcIDs["B"] = c.AllDarcs["B"].GetBaseID()

	//// *-*-*-*-*-*-*-*   Demo 2: Query should be rejected *-*-*-*-*-*-*-*-*

	// ------------------------------------------------------------------------
	// 2. Spwan query instances
	// ------------------------------------------------------------------------

	//log.Lvl1("[INFO] -*-*-*-*-*- DEMO 2 - REJECT -*-*-*-*-*-")
	log.Lvl1("[INFO] Spawning the query ")
	queries, ids, err := c.SpawnQuery(NewQuery("wsdf65k80h:B:patient_list", "Submitted"))
	require.Nil(t, err)
	require.Equal(t, 1, len(ids))
	require.Equal(t, 32, len(ids[0]))

	// Loop while we wait for the next block to be created.
	instaID, err := c.ByzCoin.ResolveInstanceID(c.AllDarcIDs["B"], queries[0].ID)
	require.Nil(t, err)
	waitForKey(t, leader.omni, c.ByzCoin.ID, instaID.Slice(), testBlockInterval)

	// Check consistency and # of queries.
	for i := 0; i < 10; i++ {
		leader.waitForBlock(c.ByzCoin.ID)
	}

	//Fetch the index, and check it.
	idx := checkProof(t, c, leader.omni, instaID.Slice(), c.ByzCoin.ID)
	qdata := QueryData{}
	err = protobuf.Decode(idx, &qdata)
	require.Nil(t, err)
	for _, s := range qdata.Storage {
		require.Equal(t, queries[0].ID, s.ID)
		require.Equal(t, queries[0].Status, (s.Status))
	}

	// Use the client API to get the query back
	// Resolve instance takes much time to run
	for _, query := range queries {
		instaID, err = c.ByzCoin.ResolveInstanceID(c.AllDarcIDs["B"], query.ID)
		require.Nil(t, err)
		_, err := c.GetQuery(instaID.Slice())
		require.Nil(t, err)
	}

	// ------------------------------------------------------------------------
	// 3. Check Authorizations
	// ------------------------------------------------------------------------
	log.Lvl1("[INFO] Query Authorization ")
	queries, ids, err = c.AddQuery(ids, queries...)

	require.Nil(t, err)
	require.Equal(t, 1, len(ids))
	require.Equal(t, 32, len(ids[0]))

	// Loop while we wait for the next block to be created.
	instaID, err = c.ByzCoin.ResolveInstanceID(c.AllDarcIDs["B"], queries[0].ID)
	require.Nil(t, err)
	waitForKey(t, leader.omni, c.ByzCoin.ID, instaID.Slice(), testBlockInterval)

	// Check consistency and # of queries.
	for i := 0; i < 10; i++ {
		leader.waitForBlock(c.ByzCoin.ID)
	}

	//Fetch the index, and check it.
	idx = checkProof(t, c, leader.omni, instaID.Slice(), c.ByzCoin.ID)
	qdata = QueryData{}
	err = protobuf.Decode(idx, &qdata)
	require.Nil(t, err)
	for _, s := range qdata.Storage {
		require.Equal(t, queries[0].ID, s.ID)
		require.Equal(t, "Rejected", s.Status)
	}

	// Use the client API to get the query back
	// Resolve instance takes much time to run
	for _, query := range queries {
		instaID, err = c.ByzCoin.ResolveInstanceID(c.AllDarcIDs["B"], query.ID)
		require.Nil(t, err)
		_, err := c.GetQuery(instaID.Slice())
		require.Nil(t, err)
	}

}

func TestClient_MedchainDeferredTxAuthorize(t *testing.T) {
	// ------------------------------------------------------------------------
	// 0. Set up and start service
	// ------------------------------------------------------------------------
	log.Lvl1("[INFO] Starting the service")
	t.Log("[INFO] Starting the service")
	s, c := newSer(t)
	require.Equal(t, s.owner, c.Signers[0])
	leader := s.services[0]
	defer s.close()

	log.Lvl1("[INFO] Starting ByzCoin Client")
	err := c.Create()
	require.Nil(t, err)

	// ------------------------------------------------------------------------
	// 1. Add Project A Darc
	// ------------------------------------------------------------------------

	rulesA := darc.InitRules([]darc.Identity{s.owner.Identity()}, []darc.Identity{c.Signers[0].Identity()})
	actionsA := "spawn:medchain,invoke:medchain.update,invoke:medchain.patient_list,invoke:medchain.count_per_site,invoke:medchain.count_per_site_obfuscated," +
		"invoke:medchain.count_per_site_shuffled,invoke:medchain.count_per_site_shuffled_obfuscated,invoke:medchain.count_global," +
		"invoke:v.count_global_obfuscated,spawn:deferred,invoke:deferred.addProof,invoke:deferred.execProposedTx,_name:deferred"
	exprA := expression.InitOrExpr(c.Signers[0].Identity().String())
	c.AllDarcs["A"], _ = c.CreateDarc("Project A darc", rulesA, actionsA, exprA)

	// Add _name to Darc rule so that we can name the instances using contract_name
	c.AllDarcs["A"].Rules.AddRule("_name:"+ContractName, exprA)
	c.AllDarcs["A"].Rules.AddRule("spawn:naming", exprA)

	// Verify the darc is correct
	require.Nil(t, c.AllDarcs["A"].Verify(true))
	t.Logf("**************** Darc of Project A ******************")
	t.Log(c.AllDarcs["A"].String())

	aDarcBuf, err := c.AllDarcs["A"].ToProto()
	require.NoError(t, err)
	aDarcCopy, err := darc.NewFromProtobuf(aDarcBuf)
	require.NoError(t, err)
	require.True(t, c.AllDarcs["A"].Equal(aDarcCopy))

	ctx, err := c.ByzCoin.CreateTransaction(byzcoin.Instruction{
		InstanceID: byzcoin.NewInstanceID(s.genDarc.GetBaseID()),
		Spawn: &byzcoin.Spawn{
			ContractID: byzcoin.ContractDarcID,
			Args: byzcoin.Arguments{
				{
					Name:  "darc",
					Value: aDarcBuf,
				},
			},
		},
		SignerIdentities: []darc.Identity{c.Signers[0].Identity()},
		SignerCounter:    c.IncrementCtrs(),
	})
	require.Nil(t, err)

	err = ctx.FillSignersAndSignWith(c.Signers...)
	require.Nil(t, err)

	_, err = c.ByzCoin.AddTransactionAndWait(ctx, 10)
	require.Nil(t, err)
	c.AllDarcIDs["A"] = c.AllDarcs["A"].GetBaseID()

	//// *-*-*-*-*-*-*-*   Demo 1: Query should be Authorized *-*-*-*-*-*-*-*-*

	// ------------------------------------------------------------------------
	// 2. Spwan query instances of MedChain contract
	// ------------------------------------------------------------------------

	//fmt.Println("[INFO] -*-*-*-*-*- DEMO 1 - AUTHORIZE -*-*-*-*-*-")
	log.Lvl1("[INFO] Spawning the query ")
	queries, ids, err := c.SpawnDeferredQuery(NewQuery("wsdf65k80h:A:patient_list", "Submitted"))
	require.Nil(t, err)
	require.Equal(t, 1, len(ids))
	require.Equal(t, 32, len(ids[0]))

	// Loop while we wait for the next block to be created.
	instaID, err := c.ByzCoin.ResolveInstanceID(c.AllDarcIDs["A"], queries[0].ID)
	require.Nil(t, err)
	waitForKey(t, leader.omni, c.ByzCoin.ID, instaID.Slice(), testBlockInterval)

	// Check consistency and # of queries.
	for i := 0; i < 10; i++ {
		leader.waitForBlock(c.ByzCoin.ID)
	}

	//Fetch the index, and check it.
	idx := checkProof(t, c, leader.omni, instaID.Slice(), c.ByzCoin.ID)
	qdata := QueryData{}
	err = protobuf.Decode(idx, &qdata)
	require.Nil(t, err)
	// for _, s := range qdata.Storage {
	// 	require.Equal(t, queries[0].ID, s.ID)
	// 	require.Equal(t, queries[0].Status, (s.Status))
	// }

	// // Use the client API to get the query back
	// // Resolve instance takes much time to run
	// for _, query := range queries {
	// 	instaID, err = c.ByzCoin.ResolveInstanceID(c.AllDarcIDs["A"], query.ID)
	// 	require.Nil(t, err)
	// 	_, err := c.GetQuery(instaID.Slice())
	// 	require.Nil(t, err)
	// }

	// ------------------------------------------------------------------------
	// 3. Sign (i.e, add proof to) the deferred query instance
	// ------------------------------------------------------------------------
	err = c.SignDeferredQuery(ids[0], c.Signers[0])
	require.Nil(t, err)
	// ------------------------------------------------------------------------
	// 4. Check Authorizations
	// ------------------------------------------------------------------------
	log.Lvl1("[INFO] Query Authorization ")
	queries, ids, err = c.AddQuery(ids, queries...)

	require.Nil(t, err)
	require.Equal(t, 1, len(ids))
	require.Equal(t, 32, len(ids[0]))

	// Loop while we wait for the next block to be created.
	instaID, err = c.ByzCoin.ResolveInstanceID(c.AllDarcIDs["A"], queries[0].ID)
	require.Nil(t, err)
	waitForKey(t, leader.omni, c.ByzCoin.ID, instaID.Slice(), testBlockInterval)

	// Check consistency and # of queries.
	for i := 0; i < 10; i++ {
		leader.waitForBlock(c.ByzCoin.ID)
	}

	//Fetch the index, and check it.
	idx = checkProof(t, c, leader.omni, instaID.Slice(), c.ByzCoin.ID)
	qdata = QueryData{}
	err = protobuf.Decode(idx, &qdata)
	require.Nil(t, err)
	for _, s := range qdata.Storage {
		require.Equal(t, queries[0].ID, s.ID)
		require.Equal(t, "Authorized", s.Status)
	}

	// Use the client API to get the query back
	// Resolve instance takes much time to run
	for _, query := range queries {
		instaID, err = c.ByzCoin.ResolveInstanceID(c.AllDarcIDs["A"], query.ID)
		require.Nil(t, err)
		_, err := c.GetQuery(instaID.Slice())
		require.Nil(t, err)
	}
}

// Do not use this test - as adding queries to ledger takes much time and thus fails
// this test usually fails
func TestClient_100Query(t *testing.T) {
	if testing.Short() {
		return
	}

	s, c := newSer(t)
	leader := s.services[0]
	defer s.close()

	err := c.Create()
	require.Nil(t, err)
	waitForKey(t, leader.omni, c.ByzCoin.ID, c.NamingInstance.Slice(), time.Second)

	// ------------------------------------------------------------------------
	// 1. Add Project A Darc
	// ------------------------------------------------------------------------

	rulesA := darc.InitRules([]darc.Identity{s.owner.Identity()}, []darc.Identity{c.Signers[0].Identity()})
	actionsA := "spawn:medchain,invoke:medchain.update,invoke:medchain.patient_list,invoke:medchain.count_per_site,invoke:medchain.count_per_site_obfuscated," +
		"invoke:medchain.count_per_site_shuffled,invoke:medchain.count_per_site_shuffled_obfuscated,invoke:medchain.count_global," +
		"invoke:medchain.count_global_obfuscated"
	exprA := expression.InitOrExpr(c.Signers[0].Identity().String())
	c.AllDarcs["A"], _ = c.CreateDarc("Project A darc", rulesA, actionsA, exprA)

	// Add _name to Darc rule so that we can name the instances using contract_name
	c.AllDarcs["A"].Rules.AddRule("_name:"+ContractName, exprA)
	c.AllDarcs["A"].Rules.AddRule("spawn:naming", exprA)

	// Verify the darc is correct
	require.Nil(t, c.AllDarcs["A"].Verify(true))
	t.Logf("**************** Darc of Project A ******************")
	t.Log(c.AllDarcs["A"].String())

	aDarcBuf, err := c.AllDarcs["A"].ToProto()
	require.NoError(t, err)
	aDarcCopy, err := darc.NewFromProtobuf(aDarcBuf)
	require.NoError(t, err)
	require.True(t, c.AllDarcs["A"].Equal(aDarcCopy))

	ctx, err := c.ByzCoin.CreateTransaction(byzcoin.Instruction{
		InstanceID: byzcoin.NewInstanceID(s.genDarc.GetBaseID()),
		Spawn: &byzcoin.Spawn{
			ContractID: byzcoin.ContractDarcID,
			Args: byzcoin.Arguments{
				{
					Name:  "darc",
					Value: aDarcBuf,
				},
			},
		},
		SignerIdentities: []darc.Identity{c.Signers[0].Identity()},
		SignerCounter:    c.IncrementCtrs(),
	})
	require.Nil(t, err)

	err = ctx.FillSignersAndSignWith(c.Signers...)
	require.Nil(t, err)

	_, err = c.ByzCoin.AddTransactionAndWait(ctx, 10)
	require.Nil(t, err)
	c.AllDarcIDs["A"] = c.AllDarcs["A"].GetBaseID()

	// ------------------------------------------------------------------------
	// 2. Add Project B Darc
	// ------------------------------------------------------------------------
	// signer can only query certain things from the database
	rulesB := darc.InitRules([]darc.Identity{s.owner.Identity()}, []darc.Identity{c.Signers[0].Identity()})
	actionsB := "spawn:medchain,invoke:medchain.update,invoke:medchain.count_global,invoke:medchain.count_global_obfuscated"
	exprB := expression.InitOrExpr(c.Signers[0].Identity().String())
	c.AllDarcs["B"], _ = c.CreateDarc("Project B darc", rulesB, actionsB, exprB)

	// Add _name to Darc rule so that we can name the instances using contract_name
	c.AllDarcs["B"].Rules.AddRule("_name:"+ContractName, exprB)
	c.AllDarcs["B"].Rules.AddRule("spawn:naming", exprB)

	// Verify the darc is correct
	require.Nil(t, c.AllDarcs["B"].Verify(true))
	t.Logf("**************** Darc of Project B ******************")
	t.Log(c.AllDarcs["B"].String())

	bDarcBuf, err := c.AllDarcs["B"].ToProto()
	require.NoError(t, err)
	bDarcCopy, err := darc.NewFromProtobuf(bDarcBuf)
	require.NoError(t, err)
	require.True(t, c.AllDarcs["B"].Equal(bDarcCopy))

	ctx, err = c.ByzCoin.CreateTransaction(byzcoin.Instruction{
		InstanceID: byzcoin.NewInstanceID(s.genDarc.GetBaseID()),
		Spawn: &byzcoin.Spawn{
			ContractID: byzcoin.ContractDarcID,
			Args: byzcoin.Arguments{
				{
					Name:  "darc",
					Value: bDarcBuf,
				},
			},
		},
		SignerIdentities: []darc.Identity{c.Signers[0].Identity()},
		SignerCounter:    c.IncrementCtrs(),
	})
	require.Nil(t, err)

	err = ctx.FillSignersAndSignWith(c.Signers...)
	require.Nil(t, err)

	_, err = c.ByzCoin.AddTransactionAndWait(ctx, 10)
	require.Nil(t, err)
	c.AllDarcIDs["B"] = c.AllDarcs["B"].GetBaseID()

	qCount := 100
	// Write the queries in chunks to make sure that the verification
	// can be done in time.
	for i := 0; i < 20; i++ {
		current := s.getCurrentBlock(t)

		start := i * qCount / 5
		for ct := start; ct < start+qCount/5; ct++ {

			// ------------------------------------------------------------------------
			// 3. Spwan query instances
			// ------------------------------------------------------------------------
			_, ids, err := c.SpawnQuery(NewQuery(randomString(10, "")+":"+randomString(1, "AB")+":"+randomAction(), "Submitted"))
			require.Nil(t, err)

			// ------------------------------------------------------------------------
			// 4. Check Authorizations
			// ------------------------------------------------------------------------
			_, _, err = c.AddQuery(ids, NewQuery(randomString(10, "")+":"+randomString(1, "AB")+":"+randomAction(), "Submitted"))
		}

		s.waitNextBlock(t, current)
	}
}

func TestClient_100QueryInOneQuery(t *testing.T) {
	if testing.Short() {
		return
	}
	// ------------------------------------------------------------------------
	// 0. Set up and start service
	// ------------------------------------------------------------------------
	s, c := newSer(t)
	leader := s.services[0]
	defer s.close()

	err := c.Create()
	require.Nil(t, err)

	// ------------------------------------------------------------------------
	// 1. Add Project A Darc
	// ------------------------------------------------------------------------

	rulesA := darc.InitRules([]darc.Identity{s.owner.Identity()}, []darc.Identity{c.Signers[0].Identity()})
	actionsA := "spawn:medchain,invoke:medchain.update,invoke:medchain.patient_list,invoke:medchain.count_per_site,invoke:medchain.count_per_site_obfuscated," +
		"invoke:medchain.count_per_site_shuffled,invoke:medchain.count_per_site_shuffled_obfuscated,invoke:medchain.count_global," +
		"invoke:medchain.count_global_obfuscated"
	exprA := expression.InitOrExpr(c.Signers[0].Identity().String())
	c.AllDarcs["A"], _ = c.CreateDarc("Project A darc", rulesA, actionsA, exprA)

	// Add _name to Darc rule so that we can name the instances using contract_name
	c.AllDarcs["A"].Rules.AddRule("_name:"+ContractName, exprA)
	c.AllDarcs["A"].Rules.AddRule("spawn:naming", exprA)

	// Verify the darc is correct
	require.Nil(t, c.AllDarcs["A"].Verify(true))
	t.Logf("**************** Darc of Project A ******************")
	t.Log(c.AllDarcs["A"].String())

	aDarcBuf, err := c.AllDarcs["A"].ToProto()
	require.NoError(t, err)
	aDarcCopy, err := darc.NewFromProtobuf(aDarcBuf)
	require.NoError(t, err)
	require.True(t, c.AllDarcs["A"].Equal(aDarcCopy))

	ctx, err := c.ByzCoin.CreateTransaction(byzcoin.Instruction{
		InstanceID: byzcoin.NewInstanceID(s.genDarc.GetBaseID()),
		Spawn: &byzcoin.Spawn{
			ContractID: byzcoin.ContractDarcID,
			Args: byzcoin.Arguments{
				{
					Name:  "darc",
					Value: aDarcBuf,
				},
			},
		},
		SignerIdentities: []darc.Identity{c.Signers[0].Identity()},
		SignerCounter:    c.IncrementCtrs(),
	})
	require.Nil(t, err)

	err = ctx.FillSignersAndSignWith(c.Signers...)
	require.Nil(t, err)

	_, err = c.ByzCoin.AddTransactionAndWait(ctx, 10)
	require.Nil(t, err)
	c.AllDarcIDs["A"] = c.AllDarcs["A"].GetBaseID()

	// ------------------------------------------------------------------------
	// 2. Add Project B Darc
	// ------------------------------------------------------------------------
	// signer can only query certain things from the database
	rulesB := darc.InitRules([]darc.Identity{s.owner.Identity()}, []darc.Identity{c.Signers[0].Identity()})
	actionsB := "spawn:medchain,invoke:medchain.update,invoke:medchain.count_global,invoke:medchain.count_global_obfuscated"
	exprB := expression.InitOrExpr(c.Signers[0].Identity().String())
	c.AllDarcs["B"], _ = c.CreateDarc("Project B darc", rulesB, actionsB, exprB)

	// Add _name to Darc rule so that we can name the instances using contract_name
	c.AllDarcs["B"].Rules.AddRule("_name:"+ContractName, exprB)
	c.AllDarcs["B"].Rules.AddRule("spawn:naming", exprB)

	// Verify the darc is correct
	require.Nil(t, c.AllDarcs["B"].Verify(true))
	t.Logf("**************** Darc of Project B ******************")
	t.Log(c.AllDarcs["B"].String())

	bDarcBuf, err := c.AllDarcs["B"].ToProto()
	require.NoError(t, err)
	bDarcCopy, err := darc.NewFromProtobuf(bDarcBuf)
	require.NoError(t, err)
	require.True(t, c.AllDarcs["B"].Equal(bDarcCopy))

	ctx, err = c.ByzCoin.CreateTransaction(byzcoin.Instruction{
		InstanceID: byzcoin.NewInstanceID(s.genDarc.GetBaseID()),
		Spawn: &byzcoin.Spawn{
			ContractID: byzcoin.ContractDarcID,
			Args: byzcoin.Arguments{
				{
					Name:  "darc",
					Value: bDarcBuf,
				},
			},
		},
		SignerIdentities: []darc.Identity{c.Signers[0].Identity()},
		SignerCounter:    c.IncrementCtrs(),
	})
	require.Nil(t, err)

	err = ctx.FillSignersAndSignWith(c.Signers...)
	require.Nil(t, err)

	_, err = c.ByzCoin.AddTransactionAndWait(ctx, 10)
	require.Nil(t, err)
	c.AllDarcIDs["B"] = c.AllDarcs["B"].GetBaseID()

	waitForKey(t, leader.omni, c.ByzCoin.ID, c.NamingInstance.Slice(), time.Second)

	qCount := 100
	// Also, one call to write a query with multiple queries in it.
	qs := make([]Query, qCount/5)
	for i := 0; i < 5; i++ {
		current := s.getCurrentBlock(t)

		for j := range qs {
			// ------------------------------------------------------------------------
			// 3. Spwan query instances
			// ------------------------------------------------------------------------
			qs[j] = NewQuery(randomString(10, "")+":"+strings.ToUpper(randomString(1, "AB"))+":"+randomAction(), "Submitted")
		}
		_, ids, err := c.SpawnQuery(qs...)
		require.Nil(t, err)
		// ------------------------------------------------------------------------
		// 4. Check Authorizations
		// ------------------------------------------------------------------------
		_, _, err = c.AddQuery(ids, qs...)
		require.Nil(t, err)

		s.waitNextBlock(t, current)
	}

	for i := 0; i < 20; i++ {
		// leader.waitForBlock isn't enough, so wait a bit longer.
		time.Sleep(s.req.BlockInterval)
		leader.waitForBlock(c.ByzCoin.ID)
	}
	require.Nil(t, err)

	// // Fetch index, and check its length.
	// instaID, err := c.ByzCoin.ResolveInstanceID(c.AllDarcIDs["A"], qs[0].ID)
	// require.Nil(t, err)
	// idx := checkProof(t, c, leader.omni, instaID.Slice(), c.ByzCoin.ID)
	// qdata := QueryData{}
	// err = protobuf.Decode(idx, &qdata)
	// require.Nil(t, err)

}

// func TestClient_EvolveDarc(t *testing.T) {
// 	// ------------------------------------------------------------------------
// 	// 0. Set up and start service
// 	// ------------------------------------------------------------------------
// 	log.Lvl1("[INFO] Starting the service")
// 	s, c := newSer(t)
// 	leader := s.services[0]
// 	defer s.close()

// 	log.Lvl1("[INFO] Starting ByzCoin Client")
// 	err := c.Create()
// 	require.Nil(t, err)
// 	waitForKey(t, leader.omni, c.ByzCoin.ID, c.NamingInstance.Slice(), time.Second)
// 	log.Lvl1("[INFO] Evolving the Darc")
// 	newRulesB := darc.InitRules([]darc.Identity{s.owner.Identity()}, []darc.Identity{c.Signers[0].Identity(), c.Signers[1].Identity(), c.Signers[2].Identity()})
// 	c.bDarc, err = c.EvolveDarc(c.bDarc, newRulesB, "Project A Darc Evolved", c.Signers[0], c.Signers[1], c.Signers[2])
// 	require.Nil(t, err)
// 	t.Log(c.bDarc.String())

// }

func checkProof(t *testing.T, c *Client, omni *byzcoin.Service, key []byte, scID skipchain.SkipBlockID) []byte {

	req := &byzcoin.GetProof{
		Version: byzcoin.CurrentVersion,
		Key:     key,
		ID:      scID,
	}
	resp, err := omni.GetProof(req)
	require.Nil(t, err)

	p := resp.Proof
	require.True(t, p.InclusionProof.Match(key), "proof of exclusion of index")

	v0, _, _, err := p.Get(key)
	require.NoError(t, err)
	return v0
}

func waitForKey(t *testing.T, s *byzcoin.Service, scID skipchain.SkipBlockID, key []byte, interval time.Duration) {
	if len(key) == 0 {
		t.Fatal("key len", len(key))
	}
	var found bool
	var resp *byzcoin.GetProofResponse
	for ct := 0; ct < 10; ct++ {
		req := &byzcoin.GetProof{
			Version: byzcoin.CurrentVersion,
			Key:     key,
			ID:      scID,
		}
		var err error
		resp, err = s.GetProof(req)
		if err == nil {
			p := resp.Proof.InclusionProof
			if p.Match(key) {
				found = true
				break
			}
		} else {
			t.Log("wait for key")
			t.Log("err", err)
		}
		time.Sleep(interval)
	}
	if !found {
		require.Fail(t, "timeout")
	}
	_, _, _, err := resp.Proof.Get(key)
	require.NoError(t, err)
}

type ser struct {
	local    *onet.LocalTest
	hosts    []*onet.Server
	roster   *onet.Roster
	services []*Service
	id       skipchain.SkipBlockID
	owner    darc.Signer
	req      *byzcoin.CreateGenesisBlock
	genDarc  *darc.Darc // the genesis darc
}

func (s *ser) close() {
	s.local.CloseAll()
}

func newSer(t *testing.T) (*ser, *Client) {
	s := &ser{
		local: onet.NewTCPTest(tSuite),
		owner: darc.NewSignerEd25519(nil, nil),
	}
	s.hosts, s.roster, _ = s.local.GenTree(3, true)

	for _, sv := range s.local.GetServices(s.hosts, sid) {
		service := sv.(*Service)
		s.services = append(s.services, service)
	}

	var err error
	s.req, err = byzcoin.DefaultGenesisMsg(byzcoin.CurrentVersion, s.roster,
		[]string{"spawn:" + ContractName, "invoke:" + ContractName + "." + "update", "invoke:" + ContractName + "." + "verifystatus", "_name:" + ContractName, "spawn:deferred", "invoke:deferred.addProof",
			"invoke:deferred.execProposedTx"}, s.owner.Identity())
	if err != nil {
		t.Fatal(err)
	}
	s.genDarc = &s.req.GenesisDarc
	s.req.BlockInterval = testBlockInterval
	cl := onet.NewClient(cothority.Suite, byzcoin.ServiceName)

	var resp byzcoin.CreateGenesisBlockResponse
	err = cl.SendProtobuf(s.roster.List[0], s.req, &resp)
	if err != nil {
		t.Fatal(err)
	}
	s.id = resp.Skipblock.Hash

	ol := byzcoin.NewClient(s.id, *s.roster)

	c := NewClient(ol)
	c.GMsg = s.req
	c.DarcID = s.genDarc.GetBaseID()
	c.Signers = []darc.Signer{s.owner}
	c.GenDarc = s.genDarc
	log.Lvl1("[INFO] Created the services")
	return s, c
}
func (s *ser) getCurrentBlock(t *testing.T) skipchain.SkipBlockID {
	reply, err := skipchain.NewClient().GetUpdateChain(s.roster, s.id)
	require.Nil(t, err)
	return reply.Update[len(reply.Update)-1].Hash
}

func (s *ser) waitNextBlock(t *testing.T, current skipchain.SkipBlockID) {
	for i := 0; i < 20; i++ {
		reply, err := skipchain.NewClient().GetUpdateChain(s.roster, s.id)
		require.Nil(t, err)
		if !current.Equal(reply.Update[len(reply.Update)-1].Hash) {
			return
		}
		time.Sleep(s.req.BlockInterval)
	}
	require.Fail(t, "waited too long for new block to appear")
}

// waitForBlock is for use in tests; it will sleep long enough to be sure that
// a block has been created.
func (s *Service) waitForBlock(scID skipchain.SkipBlockID) {
	dur, _, err := s.omni.LoadBlockInfo(scID)
	if err != nil {
		panic(err.Error())
	}
	time.Sleep(10 * dur)
}

func stringWithCharset(length int, charset string) string {
	var seededRand *rand.Rand = rand.New(
		rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

func randomString(length int, charset string) string {
	if charset == "" {
		charset = "abcdefghijklmnopqrstuvwxyz" +
			"ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	}
	return stringWithCharset(length, charset)
}

func randomAction() string {
	actions := strings.Split(actionsList, ",")
	n := rand.Intn(len(actions))
	return actions[n]
}
