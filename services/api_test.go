package medchain

import (
	"encoding/hex"
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
	"go.dedis.ch/onet/v3"
	"go.dedis.ch/onet/v3/log"
	"go.dedis.ch/protobuf"
)

// Use this block interval for logic tests. Stress test often use a different
// block interval.
var testBlockInterval = 500 * time.Millisecond
var actionsList = "patient_list,count_per_site,count_per_site_obfuscated,count_per_site_shuffled,count_per_site_shuffled_obfuscated,count_global,count_global_obfuscated"

func TestClient_MedchainAuthorize(t *testing.T) {

	// ------------------------------------------------------------------------
	// 0. Set up and start service
	// ------------------------------------------------------------------------
	log.Lvl1("[INFO] Starting the service")
	s, _, cl := newSer(t)
	leader := s.services[0]
	defer s.close()

	log.Lvl1("[INFO] Start the client")
	err := cl.Create()
	require.Nil(t, err)

	// ------------------------------------------------------------------------
	// 1. Add Project A Darc
	// ------------------------------------------------------------------------

	rulesA := darc.InitRules([]darc.Identity{s.owner.Identity()}, []darc.Identity{cl.Signers[0].Identity()})
	actionsA := "spawn:medchain,invoke:medchain.update,invoke:medchain.patient_list,invoke:medchain.count_per_site,invoke:medchain.count_per_site_obfuscated," +
		"invoke:medchain.count_per_site_shuffled,invoke:medchain.count_per_site_shuffled_obfuscated,invoke:medchain.count_global," +
		"invoke:medchain.count_global_obfuscated"
	// The test will fail if InitAndExpr is used here since deferred transactions are not used in this test
	exprA := expression.InitOrExpr(cl.Signers[0].Identity().String())
	cl.AllDarcs["A"], _ = cl.CreateDarc("Project A darc", rulesA, actionsA, exprA)

	// Add _name to Darc rule so that we can name the instances using contract_name
	cl.AllDarcs["A"].Rules.AddRule("_name:"+ContractName, exprA)
	cl.AllDarcs["A"].Rules.AddRule("spawn:naming", exprA)

	// Verify the darc is correct
	require.Nil(t, cl.AllDarcs["A"].Verify(true))

	aDarcBuf, err := cl.AllDarcs["A"].ToProto()
	require.NoError(t, err)
	aDarcCopy, err := darc.NewFromProtobuf(aDarcBuf)
	require.NoError(t, err)
	require.True(t, cl.AllDarcs["A"].Equal(aDarcCopy))

	ctx, err := cl.Bcl.CreateTransaction(byzcoin.Instruction{
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
		SignerIdentities: []darc.Identity{cl.Signers[0].Identity()},
		SignerCounter:    cl.IncrementCtrs(),
	})
	require.Nil(t, err)

	err = ctx.FillSignersAndSignWith(cl.Signers...)
	require.Nil(t, err)

	_, err = cl.Bcl.AddTransactionAndWait(ctx, 10)
	require.Nil(t, err)
	cl.AllDarcIDs["A"] = cl.AllDarcs["A"].GetBaseID()

	// ------------------------------------------------------------------------
	// 2. Spwan query instances
	// ------------------------------------------------------------------------

	log.Lvl1("[INFO] Spawning the query ")
	query := NewQuery("wsdf65k80h:A:patient_list", "Submitted")
	id1, err := cl.SpawnQuery(query)
	require.Nil(t, err)
	cl.Bcl.WaitPropagation(1)

	// Check consistency and # of queries.
	for i := 0; i < 10; i++ {
		leader.waitForBlock(cl.Bcl.ID)
	}

	//Fetch the index, and check it.
	idx := checkProof(t, cl, leader.omni, id1.Slice(), cl.Bcl.ID)
	qdata := QueryData{}
	err = protobuf.Decode(idx, &qdata)
	require.Nil(t, err)
	for _, s := range qdata.Storage {
		require.Equal(t, query.ID, s.ID)
		require.Equal(t, query.Status, (s.Status))
		require.Equal(t, "Submitted", (s.Status))
	}

	// Use the client API to get the query back
	// Resolve instance takes much time to run
	instID1, err := cl.Bcl.ResolveInstanceID(cl.AllDarcIDs["A"], query.ID)
	require.Nil(t, err)
	_, err = cl.GetQuery(instID1.Slice())
	require.Nil(t, err)
	require.Equal(t, id1, instID1)

	// ------------------------------------------------------------------------
	// 3. Check Authorizations
	// ------------------------------------------------------------------------
	log.Lvl1("[INFO] Query Authorization ")
	id2, err := cl.AuthorizeQuery(query, id1)
	require.Nil(t, err)
	require.Equal(t, id1, id2)
	cl.Bcl.WaitPropagation(1)

	// Check consistency and # of queries.
	for i := 0; i < 10; i++ {
		leader.waitForBlock(cl.Bcl.ID)
	}

	// Use the client API to get the query back
	// Resolve instance takes much time to run
	instID2, err := cl.Bcl.ResolveInstanceID(cl.AllDarcIDs["A"], query.ID)
	require.Nil(t, err)
	_, err = cl.GetQuery(instID2.Slice())
	require.Nil(t, err)
	require.Equal(t, id2, instID2)

	//Fetch the index, and check it.
	idx = checkProof(t, cl, leader.omni, id2.Slice(), cl.Bcl.ID)
	qdata = QueryData{}
	err = protobuf.Decode(idx, &qdata)
	require.Nil(t, err)
	for _, s := range qdata.Storage {
		require.Equal(t, query.ID, s.ID)
		require.Equal(t, "Authorized", s.Status)
	}

}

func TestClient_MedchainReject(t *testing.T) {
	// ------------------------------------------------------------------------
	// 0. Set up and start service
	// ------------------------------------------------------------------------
	log.Lvl1("[INFO] Starting the service")
	s, _, cl := newSer(t)
	leader := s.services[0]
	defer s.close()

	log.Lvl1("[INFO] Starting ByzCoin Client")
	err := cl.Create()
	require.Nil(t, err)

	// ------------------------------------------------------------------------
	// 1. Add Project B Darc
	// ------------------------------------------------------------------------
	// signer can only query certain things from the database
	rulesB := darc.InitRules([]darc.Identity{s.owner.Identity()}, []darc.Identity{cl.Signers[0].Identity()})
	actionsB := "spawn:medchain,invoke:medchain.update,invoke:medchain.count_global,invoke:medchain.count_global_obfuscated"
	exprB := expression.InitOrExpr(cl.Signers[0].Identity().String())
	cl.AllDarcs["B"], _ = cl.CreateDarc("Project B darc", rulesB, actionsB, exprB)

	// Add _name to Darc rule so that we can name the instances using contract_name
	cl.AllDarcs["B"].Rules.AddRule("_name:"+ContractName, exprB)
	cl.AllDarcs["B"].Rules.AddRule("spawn:naming", exprB)

	// Verify the darc is correct
	require.Nil(t, cl.AllDarcs["B"].Verify(true))

	bDarcBuf, err := cl.AllDarcs["B"].ToProto()
	require.NoError(t, err)
	bDarcCopy, err := darc.NewFromProtobuf(bDarcBuf)
	require.NoError(t, err)
	require.True(t, cl.AllDarcs["B"].Equal(bDarcCopy))

	ctx, err := cl.Bcl.CreateTransaction(byzcoin.Instruction{
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
		SignerIdentities: []darc.Identity{cl.Signers[0].Identity()},
		SignerCounter:    cl.IncrementCtrs(),
	})
	require.Nil(t, err)

	err = ctx.FillSignersAndSignWith(cl.Signers...)
	require.Nil(t, err)

	_, err = cl.Bcl.AddTransactionAndWait(ctx, 10)
	require.Nil(t, err)
	cl.AllDarcIDs["B"] = cl.AllDarcs["B"].GetBaseID()

	// ------------------------------------------------------------------------
	// 2. Spwan query instances
	// ------------------------------------------------------------------------

	//log.Lvl1("[INFO] -*-*-*-*-*- DEMO 2 - REJECT -*-*-*-*-*-")
	log.Lvl1("[INFO] Spawning the query ")
	query := NewQuery("wsdf65k80h:B:patient_list", "Submitted")
	id1, err := cl.SpawnQuery(query)
	require.Nil(t, err)
	cl.Bcl.WaitPropagation(1)

	// Check consistency and # of queries.
	for i := 0; i < 10; i++ {
		leader.waitForBlock(cl.Bcl.ID)
	}

	//Fetch the index, and check it.
	idx := checkProof(t, cl, leader.omni, id1.Slice(), cl.Bcl.ID)
	qdata := QueryData{}
	err = protobuf.Decode(idx, &qdata)
	require.Nil(t, err)
	for _, s := range qdata.Storage {
		require.Equal(t, query.ID, s.ID)
		require.Equal(t, query.Status, (s.Status))
		require.Equal(t, "Submitted", s.Status)
	}

	// Use the client API to get the query back
	// Resolve instance takes much time to run
	instID1, err := cl.Bcl.ResolveInstanceID(cl.AllDarcIDs["B"], query.ID)
	require.Nil(t, err)
	_, err = cl.GetQuery(instID1.Slice())
	require.Nil(t, err)
	require.Equal(t, id1, instID1)

	// ------------------------------------------------------------------------
	// 3. Check Authorizations
	// ------------------------------------------------------------------------
	log.Lvl1("[INFO] Query Authorization ")
	id2, err := cl.AuthorizeQuery(query, id1)
	require.Nil(t, err)
	require.Equal(t, id1, id2)
	cl.Bcl.WaitPropagation(1)

	// Check consistency and # of queries.
	for i := 0; i < 10; i++ {
		leader.waitForBlock(cl.Bcl.ID)
	}
	// Use the client API to get the query back; takes much time to run
	instID2, err := cl.Bcl.ResolveInstanceID(cl.AllDarcIDs["B"], query.ID)
	require.Nil(t, err)
	_, err = cl.GetQuery(instID2.Slice())
	require.Nil(t, err)
	require.Equal(t, id2, instID2)

	//Fetch the index, and check it.
	idx = checkProof(t, cl, leader.omni, id2.Slice(), cl.Bcl.ID)
	qdata = QueryData{}
	err = protobuf.Decode(idx, &qdata)
	require.Nil(t, err)
	for _, s := range qdata.Storage {
		require.Equal(t, query.ID, s.ID)
		require.Equal(t, "Rejected", s.Status)
	}

}

func TestClient_MedchainDeferredTxAuthorize(t *testing.T) {
	// ------------------------------------------------------------------------
	// 0. Set up and start service
	// ------------------------------------------------------------------------

	log.Info("[INFO] Starting the service")
	s, _, cl := newSer(t)
	require.Equal(t, s.owner, cl.Signers[0])
	leader := s.services[0]
	defer s.close()

	log.Info("[INFO] Starting ByzCoin Client")
	err := cl.Create()
	require.Nil(t, err)

	// ------------------------------------------------------------------------
	// 1. Add Project A Darc
	// ------------------------------------------------------------------------

	rulesA := darc.InitRules([]darc.Identity{s.owner.Identity()}, []darc.Identity{cl.Signers[0].Identity()})
	actionsA := "spawn:medchain,invoke:medchain.update,invoke:medchain.patient_list,invoke:medchain.count_per_site,invoke:medchain.count_per_site_obfuscated," +
		"invoke:medchain.count_per_site_shuffled,invoke:medchain.count_per_site_shuffled_obfuscated,invoke:medchain.count_global," +
		"invoke:medchain.count_global_obfuscated,spawn:deferred,invoke:deferred.addProof,invoke:deferred.execProposedTx,_name:deferred"

	// all signers need to sing
	exprA := expression.InitAndExpr(cl.Signers[0].Identity().String())
	cl.AllDarcs["A"], _ = cl.CreateDarc("Project A darc", rulesA, actionsA, exprA)

	// Add _name to Darc rule so that we can name the instances using contract_name
	cl.AllDarcs["A"].Rules.AddRule("_name:"+ContractName, exprA)
	cl.AllDarcs["A"].Rules.AddRule("spawn:naming", exprA)

	// Verify the darc is correct
	require.Nil(t, cl.AllDarcs["A"].Verify(true))

	aDarcBuf, err := cl.AllDarcs["A"].ToProto()
	require.NoError(t, err)
	aDarcCopy, err := darc.NewFromProtobuf(aDarcBuf)
	require.NoError(t, err)
	require.True(t, cl.AllDarcs["A"].Equal(aDarcCopy))

	ctx, err := cl.Bcl.CreateTransaction(byzcoin.Instruction{
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
		SignerIdentities: []darc.Identity{cl.Signers[0].Identity()},
		SignerCounter:    cl.IncrementCtrs(),
	})
	require.Nil(t, err)

	err = ctx.FillSignersAndSignWith(cl.Signers...)
	require.Nil(t, err)

	_, err = cl.Bcl.AddTransactionAndWait(ctx, 10)
	require.Nil(t, err)
	cl.AllDarcIDs["A"] = cl.AllDarcs["A"].GetBaseID()

	// ------------------------------------------------------------------------
	// 2. Spwan query instances of MedChain contract
	// ------------------------------------------------------------------------

	req1 := &AddDeferredQueryRequest{}
	query := NewQuery("wsdf65k80h:A:patient_list", "l")
	req1.QueryID = query.ID
	req1.ClientID = cl.ClientID
	resp1, err := cl.SpawnDeferredQuery(req1)
	require.NoError(t, err)
	require.NotNil(t, resp1)
	require.NotNil(t, req1.QueryInstID)
	require.True(t, resp1.OK)
	require.Equal(t, "Submitted", req1.QueryStatus)

	result, err := cl.Bcl.GetDeferredDataAfter(resp1.QueryInstID, cl.Bcl.Latest)
	require.Nil(t, err)
	// Default MaxNumExecution should be 1
	require.Equal(t, result.MaxNumExecution, uint64(1))
	require.NotEmpty(t, result.InstructionHashes)

	cl.Bcl.WaitPropagation(1)

	// Check consistency and # of queries.
	for i := 0; i < 10; i++ {
		leader.waitForBlock(cl.Bcl.ID)
	}

	//Fetch the index, and check it.
	idx := checkProof(t, cl, leader.omni, req1.QueryInstID.Slice(), cl.Bcl.ID)
	qu := QueryData{}
	err = protobuf.Decode(idx, &qu)
	require.NoError(t, err)

	dd, err := cl.Bcl.GetDeferredData(req1.QueryInstID)
	require.NoError(t, err)
	require.Equal(t, uint64(1), dd.MaxNumExecution)

	// ------------------------------------------------------------------------
	// 3. Add signature (i.e, add proof) to the deferred query instance
	// ------------------------------------------------------------------------
	req2 := &SignDeferredTxRequest{}
	req2.ClientID = cl.ClientID
	req2.QueryInstID = req1.QueryInstID
	resp2, err := cl.AddSignatureToDeferredQuery(req2)
	require.NoError(t, err)
	require.NotNil(t, resp2)
	require.True(t, resp2.OK)

	// ------------------------------------------------------------------------
	// 4. Execute the query transaction
	// ------------------------------------------------------------------------
	err = cl.ExecDefferedQuery(req1.QueryInstID)
	require.NoError(t, err)
	require.Equal(t, resp2.QueryInstID, req2.QueryInstID)
	require.Equal(t, resp2.QueryInstID, req1.QueryInstID)
	require.Equal(t, resp1.QueryInstID, resp2.QueryInstID)

	iIDStr := resp2.QueryInstID.String()
	iIDBuf, err := hex.DecodeString(iIDStr)
	require.NoError(t, err)
	require.Equal(t, resp2.QueryInstID, byzcoin.NewInstanceID(iIDBuf))

	// ------------------------------------------------------------------------
	// 5. Check Authorizations
	// ------------------------------------------------------------------------
	err = cl.GetDarcRules(req1.QueryInstID)
	require.NoError(t, err)

	id2, err := cl.AuthorizeQuery(query, req1.QueryInstID)
	require.Nil(t, err)
	require.Equal(t, 32, len(id2))
	require.NotEqual(t, req1.QueryInstID, id2)
	cl.Bcl.WaitPropagation(1)

	// Check consistency and # of queries.
	for i := 0; i < 10; i++ {
		leader.waitForBlock(cl.Bcl.ID)
	}

	//Fetch the index, and check it.
	idx = checkProof(t, cl, leader.omni, id2.Slice(), cl.Bcl.ID)
	qdata := QueryData{}
	err = protobuf.Decode(idx, &qdata)
	require.Nil(t, err)
	for _, s := range qdata.Storage {
		require.Equal(t, query.ID, s.ID)
		require.Equal(t, "Authorized", s.Status)
	}

	// Use the client API to get the query back
	// Resolve instance takes much time to run
	instaID, err := cl.Bcl.ResolveInstanceID(cl.AllDarcIDs["A"], query.ID)
	require.Nil(t, err)
	_, err = cl.GetQuery(instaID.Slice())
	require.Nil(t, err)

}

// This test usually fails due to a bottle neck
func TestClient_100Query(t *testing.T) {
	if testing.Short() {
		return
	}

	s, _, cl := newSer(t)
	leader := s.services[0]
	defer s.close()

	err := cl.Create()
	require.Nil(t, err)
	waitForKey(t, leader.omni, cl.Bcl.ID, cl.NamingInstance.Slice(), time.Second)

	// ------------------------------------------------------------------------
	// 1. Add Project A Darc
	// ------------------------------------------------------------------------

	rulesA := darc.InitRules([]darc.Identity{s.owner.Identity()}, []darc.Identity{cl.Signers[0].Identity()})
	actionsA := "spawn:medchain,invoke:medchain.update,invoke:medchain.patient_list,invoke:medchain.count_per_site,invoke:medchain.count_per_site_obfuscated," +
		"invoke:medchain.count_per_site_shuffled,invoke:medchain.count_per_site_shuffled_obfuscated,invoke:medchain.count_global," +
		"invoke:medchain.count_global_obfuscated"
	exprA := expression.InitOrExpr(cl.Signers[0].Identity().String())
	cl.AllDarcs["A"], _ = cl.CreateDarc("Project A darc", rulesA, actionsA, exprA)

	// Add _name to Darc rule so that we can name the instances using contract_name
	cl.AllDarcs["A"].Rules.AddRule("_name:"+ContractName, exprA)
	cl.AllDarcs["A"].Rules.AddRule("spawn:naming", exprA)

	// Verify the darc is correct
	require.Nil(t, cl.AllDarcs["A"].Verify(true))

	aDarcBuf, err := cl.AllDarcs["A"].ToProto()
	require.NoError(t, err)
	aDarcCopy, err := darc.NewFromProtobuf(aDarcBuf)
	require.NoError(t, err)
	require.True(t, cl.AllDarcs["A"].Equal(aDarcCopy))

	ctx, err := cl.Bcl.CreateTransaction(byzcoin.Instruction{
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
		SignerIdentities: []darc.Identity{cl.Signers[0].Identity()},
		SignerCounter:    cl.IncrementCtrs(),
	})
	require.Nil(t, err)

	err = ctx.FillSignersAndSignWith(cl.Signers...)
	require.Nil(t, err)

	_, err = cl.Bcl.AddTransactionAndWait(ctx, 10)
	require.Nil(t, err)
	cl.AllDarcIDs["A"] = cl.AllDarcs["A"].GetBaseID()

	// ------------------------------------------------------------------------
	// 2. Add Project B Darc
	// ------------------------------------------------------------------------
	// signer can only query certain things from the database
	rulesB := darc.InitRules([]darc.Identity{s.owner.Identity()}, []darc.Identity{cl.Signers[0].Identity()})
	actionsB := "spawn:medchain,invoke:medchain.update,invoke:medchain.count_global,invoke:medchain.count_global_obfuscated"
	exprB := expression.InitOrExpr(cl.Signers[0].Identity().String())
	cl.AllDarcs["B"], _ = cl.CreateDarc("Project B darc", rulesB, actionsB, exprB)

	// Add _name to Darc rule so that we can name the instances using contract_name
	cl.AllDarcs["B"].Rules.AddRule("_name:"+ContractName, exprB)
	cl.AllDarcs["B"].Rules.AddRule("spawn:naming", exprB)

	// Verify the darc is correct
	require.Nil(t, cl.AllDarcs["B"].Verify(true))

	bDarcBuf, err := cl.AllDarcs["B"].ToProto()
	require.NoError(t, err)
	bDarcCopy, err := darc.NewFromProtobuf(bDarcBuf)
	require.NoError(t, err)
	require.True(t, cl.AllDarcs["B"].Equal(bDarcCopy))

	ctx, err = cl.Bcl.CreateTransaction(byzcoin.Instruction{
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
		SignerIdentities: []darc.Identity{cl.Signers[0].Identity()},
		SignerCounter:    cl.IncrementCtrs(),
	})
	require.Nil(t, err)

	err = ctx.FillSignersAndSignWith(cl.Signers...)
	require.Nil(t, err)

	_, err = cl.Bcl.AddTransactionAndWait(ctx, 10)
	require.Nil(t, err)
	cl.AllDarcIDs["B"] = cl.AllDarcs["B"].GetBaseID()

	qCount := 100
	// Write the queries in chunks to make sure that the verification
	// can be done in time.
	for i := 0; i < 10; i++ {
		current := s.getCurrentBlock(t)

		start := i * qCount / 5
		for ct := start; ct < start+qCount/5; ct++ {
			// ------------------------------------------------------------------------
			// 3. Spwan query instances
			// ------------------------------------------------------------------------
			id1, err := cl.SpawnQuery(NewQuery(randomString(10, "")+":"+randomString(1, "AB")+":"+randomAction(), "Submitted"))
			require.Nil(t, err)

			// ------------------------------------------------------------------------
			// 4. Check Authorizations
			// ------------------------------------------------------------------------
			id2, err := cl.AuthorizeQuery(NewQuery(randomString(10, "")+":"+randomString(1, "AB")+":"+randomAction(), "Submitted"), id1)
			require.NotEqual(t, id1, id2)
		}

		s.waitNextBlock(t, current)
	}
}

func TestClient_100QueryInOneQuery(t *testing.T) {
	// if testing.Short() {
	// 	return
	// }
	// // ------------------------------------------------------------------------
	// // 0. Set up and start service
	// // ------------------------------------------------------------------------
	// s, c := newSer(t)
	// leader := s.services[0]
	// defer s.close()

	// err := c.Create()
	// require.Nil(t, err)

	// // ------------------------------------------------------------------------
	// // 1. Add Project A Darc
	// // ------------------------------------------------------------------------

	// rulesA := darc.InitRules([]darc.Identity{s.owner.Identity()}, []darc.Identity{c.Signers[0].Identity()})
	// actionsA := "spawn:medchain,invoke:medchain.update,invoke:medchain.patient_list,invoke:medchain.count_per_site,invoke:medchain.count_per_site_obfuscated," +
	// 	"invoke:medchain.count_per_site_shuffled,invoke:medchain.count_per_site_shuffled_obfuscated,invoke:medchain.count_global," +
	// 	"invoke:medchain.count_global_obfuscated"
	// exprA := expression.InitOrExpr(c.Signers[0].Identity().String())
	// c.AllDarcs["A"], _ = c.CreateDarc("Project A darc", rulesA, actionsA, exprA)

	// // Add _name to Darc rule so that we can name the instances using contract_name
	// c.AllDarcs["A"].Rules.AddRule("_name:"+ContractName, exprA)
	// c.AllDarcs["A"].Rules.AddRule("spawn:naming", exprA)

	// // Verify the darc is correct
	// require.Nil(t, c.AllDarcs["A"].Verify(true))

	// aDarcBuf, err := c.AllDarcs["A"].ToProto()
	// require.NoError(t, err)
	// aDarcCopy, err := darc.NewFromProtobuf(aDarcBuf)
	// require.NoError(t, err)
	// require.True(t, c.AllDarcs["A"].Equal(aDarcCopy))

	// ctx, err := c.bcl.CreateTransaction(byzcoin.Instruction{
	// 	InstanceID: byzcoin.NewInstanceID(s.genDarc.GetBaseID()),
	// 	Spawn: &byzcoin.Spawn{
	// 		ContractID: byzcoin.ContractDarcID,
	// 		Args: byzcoin.Arguments{
	// 			{
	// 				Name:  "darc",
	// 				Value: aDarcBuf,
	// 			},
	// 		},
	// 	},
	// 	SignerIdentities: []darc.Identity{c.Signers[0].Identity()},
	// 	SignerCounter:    c.IncrementCtrs(),
	// })
	// require.Nil(t, err)

	// err = ctx.FillSignersAndSignWith(c.Signers...)
	// require.Nil(t, err)

	// _, err = c.bcl.AddTransactionAndWait(ctx, 10)
	// require.Nil(t, err)
	// c.AllDarcIDs["A"] = c.AllDarcs["A"].GetBaseID()

	// // ------------------------------------------------------------------------
	// // 2. Add Project B Darc
	// // ------------------------------------------------------------------------
	// // signer can only query certain things from the database
	// rulesB := darc.InitRules([]darc.Identity{s.owner.Identity()}, []darc.Identity{c.Signers[0].Identity()})
	// actionsB := "spawn:medchain,invoke:medchain.update,invoke:medchain.count_global,invoke:medchain.count_global_obfuscated"
	// exprB := expression.InitOrExpr(c.Signers[0].Identity().String())
	// c.AllDarcs["B"], _ = c.CreateDarc("Project B darc", rulesB, actionsB, exprB)

	// // Add _name to Darc rule so that we can name the instances using contract_name
	// c.AllDarcs["B"].Rules.AddRule("_name:"+ContractName, exprB)
	// c.AllDarcs["B"].Rules.AddRule("spawn:naming", exprB)

	// // Verify the darc is correct
	// require.Nil(t, c.AllDarcs["B"].Verify(true))

	// bDarcBuf, err := c.AllDarcs["B"].ToProto()
	// require.NoError(t, err)
	// bDarcCopy, err := darc.NewFromProtobuf(bDarcBuf)
	// require.NoError(t, err)
	// require.True(t, c.AllDarcs["B"].Equal(bDarcCopy))

	// ctx, err = c.bcl.CreateTransaction(byzcoin.Instruction{
	// 	InstanceID: byzcoin.NewInstanceID(s.genDarc.GetBaseID()),
	// 	Spawn: &byzcoin.Spawn{
	// 		ContractID: byzcoin.ContractDarcID,
	// 		Args: byzcoin.Arguments{
	// 			{
	// 				Name:  "darc",
	// 				Value: bDarcBuf,
	// 			},
	// 		},
	// 	},
	// 	SignerIdentities: []darc.Identity{c.Signers[0].Identity()},
	// 	SignerCounter:    c.IncrementCtrs(),
	// })
	// require.Nil(t, err)

	// err = ctx.FillSignersAndSignWith(c.Signers...)
	// require.Nil(t, err)

	// _, err = c.bcl.AddTransactionAndWait(ctx, 10)
	// require.Nil(t, err)
	// c.AllDarcIDs["B"] = c.AllDarcs["B"].GetBaseID()

	// waitForKey(t, leader.omni, c.bcl.ID, c.NamingInstance.Slice(), time.Second)

	// qCount := 100
	// // Also, one call to write a query with multiple queries in it.
	// qu := make([]Query, qCount/5)
	// for i := 0; i < 5; i++ {
	// 	current := s.getCurrentBlock(t)

	// 	for j := range qs {
	// 		// ------------------------------------------------------------------------
	// 		// 3. Spwan query instances
	// 		// ------------------------------------------------------------------------
	// 		qu[j] = NewQuery(randomString(10, "")+":"+strings.ToUpper(randomString(1, "AB"))+":"+randomAction(), "Submitted")
	// 	}
	// 	id1, err := c.SpawnQuery(qu)
	// 	require.Nil(t, err)
	// 	// ------------------------------------------------------------------------
	// 	// 4. Check Authorizations
	// 	// ------------------------------------------------------------------------
	// 	_, _, err = c.AuthorizeQuery(id1, qu)
	// 	require.Nil(t, err)

	// 	s.waitNextBlock(t, current)
	// }

	// for i := 0; i < 20; i++ {
	// 	// leader.waitForBlock isn't enough, so wait a bit longer.
	// 	time.Sleep(s.req.BlockInterval)
	// 	leader.waitForBlock(c.bcl.ID)
	// }
	// require.Nil(t, err)
}

func TestClient_EvolveDarc(t *testing.T) {
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

}

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

func newSer(t *testing.T) (*ser, *byzcoin.Client, *Client) {
	s := &ser{
		local: onet.NewTCPTest(TSuite),
		owner: darc.NewSignerEd25519(nil, nil),
	}
	s.hosts, s.roster, _ = s.local.GenTree(3, true)
	serverID := s.roster.RandomServerIdentity()
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
	ocl := onet.NewClient(cothority.Suite, byzcoin.ServiceName)

	var resp byzcoin.CreateGenesisBlockResponse
	err = ocl.SendProtobuf(s.roster.List[0], s.req, &resp)
	if err != nil {
		t.Fatal(err)
	}
	s.id = resp.Skipblock.Hash

	bcl := byzcoin.NewClient(s.id, *s.roster)

	cl, err := NewClient(bcl, serverID, "1")
	require.NoError(t, err)

	cl.GMsg = s.req
	// cl.DarcID = s.genDarc.GetBaseID()
	cl.Signers = []darc.Signer{s.owner}
	cl.GenDarc = s.genDarc
	log.Lvl1("[INFO] Created the services")
	return s, bcl, cl
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
