package medchain

import (
	"encoding/hex"
	"fmt"
	"math/rand"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.dedis.ch/cothority/v3"
	"go.dedis.ch/cothority/v3/byzcoin"
	"go.dedis.ch/cothority/v3/byzcoin/bcadmin/lib"
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
	actionsAAnd := "invoke:medchain.patient_list,invoke:medchain.count_per_site,invoke:medchain.count_per_site_obfuscated," +
		"invoke:medchain.count_per_site_shuffled,invoke:medchain.count_per_site_shuffled_obfuscated,invoke:medchain.count_global," +
		"invoke:medchain.count_global_obfuscated,invoke:darc.evolve"

	actionsAOr := "spawn:deferred,invoke:deferred.addProof,invoke:deferred.execProposedTx,spawn:medchain,invoke:medchain.update,_name:deferred,spawn:naming,_name:medchain"

	// all signers need to sign
	exprAAnd := expression.InitAndExpr(cl.Signers[0].Identity().String())

	// at least one signer need to sign
	exprAOr := expression.InitOrExpr(cl.Signers[0].Identity().String())
	cl.AllDarcs["A"], _ = cl.CreateDarc("Project A darc", rulesA, actionsAAnd, actionsAOr, exprAAnd, exprAOr)

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
	log.Info("[INFO] Darc for Project A is:", cl.AllDarcs["A"].String())

	// ------------------------------------------------------------------------
	// 2. Spwan query instances
	// ------------------------------------------------------------------------

	log.Lvl1("[INFO] Spawning the query ")
	query := NewQuery("wsdf65k80h:A:patient_list", "Submitted")
	id1, err := cl.SpawnQuery(query)
	require.Nil(t, err)
	cl.Bcl.WaitPropagation(1)

	//Fetch the index, and check it.
	idx := checkProof(t, cl, leader.omni, id1.Slice(), cl.Bcl.ID)
	qdata := QueryData{}
	err = protobuf.Decode(idx, &qdata)
	require.Nil(t, err)
	for _, s := range qdata.Storage {
		require.Equal(t, query.ID, s.ID)
		require.Equal(t, query.Status, (s.Status))
		require.Equal(t, []byte("Submitted"), (s.Status))
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
	req := &AuthorizeQueryRequest{}
	req.QueryID = query.ID
	req.QueryStatus = query.Status
	req.QueryInstID = id1
	req.DarcID = cl.AllDarcs["A"].GetBaseID()
	resp, err := cl.AuthorizeQuery(req)
	require.Nil(t, err)
	require.True(t, resp.OK)
	require.Equal(t, 32, len(resp.QueryInstID))
	require.Equal(t, req.QueryInstID, resp.QueryInstID)
	cl.Bcl.WaitPropagation(1)

	// Use the client API to get the query back
	// Resolve instance takes much time to run
	instID2, err := cl.Bcl.ResolveInstanceID(cl.AllDarcIDs["A"], query.ID)
	require.Nil(t, err)
	qu1, err := cl.GetQuery(instID2.Slice())
	require.Nil(t, err)
	require.NotEqual(t, resp.QueryInstID, instID2)
	require.NotEqual(t, resp.QueryInstID, req.QueryID)
	require.Equal(t, []byte("Submitted"), qu1.Storage[0].Status)
	require.Equal(t, "wsdf65k80h:A:patient_list", qu1.Storage[0].ID)

	// Use the client API to get the query back
	// Resolve instance takes much time to run
	instID3, err := cl.Bcl.ResolveInstanceID(cl.GenDarcID, query.ID+"_auth")
	require.Nil(t, err)
	qu, err := cl.GetQuery(instID3.Slice())
	require.Nil(t, err)
	require.Equal(t, 1, len(qu.Storage))
	require.Equal(t, resp.QueryInstID, instID3)
	require.Equal(t, []byte("Authorized"), qu.Storage[0].Status)
	require.Equal(t, "wsdf65k80h:A:patient_list", qu.Storage[0].ID)

	//Fetch the index, and check it.
	idx = checkProof(t, cl, leader.omni, resp.QueryInstID.Slice(), cl.Bcl.ID)
	qdata = QueryData{}
	err = protobuf.Decode(idx, &qdata)
	require.Nil(t, err)
	require.Equal(t, 1, len(qdata.Storage))

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
	actionsBAnd := "invoke:medchain.count_per_site,invoke:medchain.count_per_site_obfuscated," +
		"invoke:medchain.count_per_site_shuffled,invoke:medchain.count_per_site_shuffled_obfuscated,invoke:medchain.count_global," +
		"invoke:medchain.count_global_obfuscated,invoke:darc.evolve"

	actionsBOr := "spawn:deferred,invoke:deferred.addProof,invoke:deferred.execProposedTx,spawn:medchain,invoke:medchain.update,_name:deferred,spawn:naming,_name:medchain"

	// all signers need to sign
	exprBAnd := expression.InitAndExpr(cl.Signers[0].Identity().String())

	// at least one signer need to sign
	exprBOr := expression.InitOrExpr(cl.Signers[0].Identity().String())
	cl.AllDarcs["B"], _ = cl.CreateDarc("Project B darc", rulesB, actionsBAnd, actionsBOr, exprBAnd, exprBOr)
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
	log.Info("[INFO] Darc for Project B is:", cl.AllDarcs["B"].String())

	// ------------------------------------------------------------------------
	// 2. Spwan query instances
	// ------------------------------------------------------------------------

	//log.Lvl1("[INFO] -*-*-*-*-*- DEMO 2 - REJECT -*-*-*-*-*-")
	log.Lvl1("[INFO] Spawning the query ")
	query := NewQuery("wsdf65k80h:B:patient_list", "Submitted")
	id1, err := cl.SpawnQuery(query)
	require.Nil(t, err)
	cl.Bcl.WaitPropagation(1)

	// // Check consistency and # of queries.
	// for i := 0; i < 10; i++ {
	// 	leader.waitForBlock(cl.Bcl.ID)
	// }

	//Fetch the index, and check it.
	idx := checkProof(t, cl, leader.omni, id1.Slice(), cl.Bcl.ID)
	qdata := QueryData{}
	err = protobuf.Decode(idx, &qdata)
	require.Nil(t, err)
	for _, s := range qdata.Storage {
		require.Equal(t, query.ID, s.ID)
		require.Equal(t, []byte(query.Status), (s.Status))
		require.Equal(t, []byte("Submitted"), s.Status)
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
	req := &AuthorizeQueryRequest{}
	req.QueryID = query.ID
	req.QueryStatus = query.Status
	req.QueryInstID = id1
	req.DarcID = cl.AllDarcs["B"].GetBaseID()
	resp, err := cl.AuthorizeQuery(req)
	require.Nil(t, err)
	require.True(t, resp.OK)
	require.Equal(t, 32, len(resp.QueryInstID))
	require.Equal(t, req.QueryInstID, resp.QueryInstID)
	cl.Bcl.WaitPropagation(1)

	// Use the client API to get the query back; takes much time to run
	instID2, err := cl.Bcl.ResolveInstanceID(cl.AllDarcIDs["B"], query.ID)
	require.Nil(t, err)
	qu, err := cl.GetQuery(instID2.Slice())
	require.Nil(t, err)
	require.Equal(t, 1, len(qu.Storage))
	require.Equal(t, resp.QueryInstID, instID2)
	require.Equal(t, []byte("Submitted"), qu.Storage[0].Status)
	require.Equal(t, "wsdf65k80h:A:patient_list", qu.Storage[0].ID)

	instID3, err := cl.Bcl.ResolveInstanceID(cl.AllDarcIDs["B"], query.ID+"_auth")
	require.Nil(t, err)
	qu2, err := cl.GetQuery(instID3.Slice())
	require.Nil(t, err)
	require.Equal(t, 2, len(qu2.Storage))
	require.Equal(t, resp.QueryInstID, instID2)
	require.Equal(t, []byte("Submitted"), qu.Storage[0].Status)
	require.Equal(t, []byte("Rejected"), qu.Storage[1].Status)
	require.Equal(t, "wsdf65k80h:A:patient_list", qu.Storage[1].ID)

	//Fetch the index, and check it.
	idx = checkProof(t, cl, leader.omni, resp.QueryInstID.Slice(), cl.Bcl.ID)
	qdata = QueryData{}
	err = protobuf.Decode(idx, &qdata)
	require.Nil(t, err)
	require.Equal(t, 2, len(qu.Storage))

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
	actionsAAnd := "invoke:medchain.patient_list,invoke:medchain.count_per_site,invoke:medchain.count_per_site_obfuscated," +
		"invoke:medchain.count_per_site_shuffled,invoke:medchain.count_per_site_shuffled_obfuscated,invoke:medchain.count_global," +
		"invoke:medchain.count_global_obfuscated,invoke:darc.evolve"

	actionsAOr := "spawn:deferred,invoke:deferred.addProof,invoke:deferred.execProposedTx,spawn:medchain,invoke:medchain.update,_name:deferred,spawn:naming,_name:medchain,_config"

	// all signers need to sign
	exprAAnd := expression.InitAndExpr(cl.Signers[0].Identity().String())

	// at least one signer need to sign
	exprAOr := expression.InitOrExpr(cl.Signers[0].Identity().String())
	cl.AllDarcs["A"], err = cl.CreateDarc("Project A darc", rulesA, actionsAAnd, actionsAOr, exprAAnd, exprAOr)
	require.NoError(t, err)

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
	log.Info("[INFO] Darc for Project A is:", cl.AllDarcs["A"].String())

	// ------------------------------------------------------------------------
	// 2. Spwan query instances of MedChain contract
	// ------------------------------------------------------------------------

	req1 := &AddDeferredQueryRequest{}
	query := NewQuery("wsdf65k80h:A:patient_list", "Submitted")
	req1.QueryID = query.ID
	req1.ClientID = cl.ClientID
	req1.DarcID = cl.AllDarcs["A"].GetBaseID()
	resp1, err := cl.SpawnDeferredQuery(req1)
	require.NoError(t, err)
	require.NotNil(t, resp1)
	require.NotNil(t, req1.QueryInstID)
	require.Equal(t, req1.QueryInstID, resp1.QueryInstID)
	require.True(t, resp1.OK)
	require.Equal(t, []byte("Submitted"), req1.QueryStatus)

	result, err := cl.Bcl.GetDeferredDataAfter(resp1.QueryInstID, cl.Bcl.Latest)
	require.Nil(t, err)
	// Default MaxNumExecution should be 1
	require.Equal(t, result.MaxNumExecution, uint64(1))
	require.NotEmpty(t, result.InstructionHashes)
	require.Empty(t, result.ExecResult)

	cl.Bcl.WaitPropagation(1)

	//Fetch the index, and check it.
	idx := checkProof(t, cl, leader.omni, req1.QueryInstID.Slice(), cl.Bcl.ID)
	qu := byzcoin.DeferredData{}
	err = protobuf.Decode(idx, &qu)
	require.NoError(t, err)
	require.Equal(t, 1, len(qu.ProposedTransaction.Instructions))
	log.Info("[INFO] spawned deferred data retrieved", qu)
	log.Info("[INFO] spawned deferred data args retrieved", qu.ProposedTransaction.Instructions)

	dd1, err := cl.Bcl.GetDeferredData(resp1.QueryInstID)
	log.Infof("Here is the deferred data after spawning it: \n%s", dd1)
	require.NoError(t, err)
	require.NotEmpty(t, dd1.InstructionHashes)
	require.Equal(t, uint64(1), dd1.MaxNumExecution)
	require.Empty(t, dd1.ExecResult)

	// ------------------------------------------------------------------------
	// 3. Add signature (i.e, add proof) to the deferred query instance
	// ------------------------------------------------------------------------
	req2 := &SignDeferredTxRequest{}
	req2.Keys = cl.Signers[0]
	req2.ClientID = cl.ClientID
	req2.QueryInstID = req1.QueryInstID
	resp2, err := cl.AddSignatureToDeferredQuery(req2)
	require.NoError(t, err)
	require.NotNil(t, resp2)
	require.True(t, resp2.OK)
	require.Equal(t, req1.QueryInstID, resp2.QueryInstID)

	dd2, err := cl.Bcl.GetDeferredData(resp2.QueryInstID)
	log.Infof("Here is the deferred data after adding signature: \n%s", dd2)
	require.NoError(t, err)
	require.NotEmpty(t, dd2.InstructionHashes)
	require.Equal(t, uint64(1), dd2.MaxNumExecution)
	require.Empty(t, dd2.ExecResult)

	// ------------------------------------------------------------------------
	// 4. Execute the query transaction
	// ------------------------------------------------------------------------
	req3 := &ExecuteDeferredTxRequest{}
	req3.ClientID = cl.ClientID
	req3.QueryInstID = req1.QueryInstID
	req3.QueryStatus = req1.QueryStatus
	resp3, err := cl.ExecDefferedQuery(req3)
	require.NoError(t, err)
	require.True(t, resp3.OK)
	require.Equal(t, resp2.QueryInstID, req3.QueryInstID)
	require.Equal(t, resp3.QueryInstID, req3.QueryInstID)
	require.Equal(t, resp1.QueryInstID, resp3.QueryInstID)

	iIDStr := resp3.QueryInstID.String()
	iIDBuf, err := hex.DecodeString(iIDStr)
	require.NoError(t, err)
	require.Equal(t, resp3.QueryInstID, byzcoin.NewInstanceID(iIDBuf))

	result2, err := cl.Bcl.GetDeferredDataAfter(resp1.QueryInstID, cl.Bcl.Latest)
	require.Nil(t, err)
	// Default MaxNumExecution should now be decremented to 0
	require.Equal(t, result2.MaxNumExecution, uint64(0))
	require.NotEmpty(t, result2.InstructionHashes)
	require.NotEmpty(t, result2.ExecResult)
	log.Infof("Here is the deferred data after execution: \n%s", result)

	// ------------------------------------------------------------------------
	// 5. Check Authorizations
	// ------------------------------------------------------------------------
	err = cl.GetDarcRules(req1.QueryInstID)
	require.NoError(t, err)

	req4 := &AuthorizeQueryRequest{}
	req4.QueryID = query.ID
	req4.QueryInstID = req1.QueryInstID
	req4.DarcID = cl.AllDarcs["A"].GetBaseID()
	resp4, err := cl.AuthorizeQuery(req4)
	require.Nil(t, err)
	require.True(t, resp4.OK)
	require.Equal(t, 32, len(resp4.QueryInstID))
	require.NotEqual(t, req1.QueryInstID, resp4.QueryInstID)
	cl.Bcl.WaitPropagation(5)

	instID := checkProof(t, cl, leader.omni, req4.QueryInstID.Slice(), cl.Bcl.ID)
	qu2 := QueryData{}
	err = protobuf.Decode(instID, &qu2)
	require.NoError(t, err)
	require.Equal(t, 1, len(qu.ProposedTransaction.Instructions))
	log.Info("[INFO] deferred data retrieved in the end", qu)
	log.Info("[INFO] deferred data args retrieved in the end", qu.ProposedTransaction.Instructions)

	log.Info("[INFO] deffered instance ID", req4.QueryInstID.String())
	log.Info("[INFO] normal instance ID", resp4.QueryInstID.String())

	instID2 := checkProof(t, cl, leader.omni, resp4.QueryInstID.Slice(), cl.Bcl.ID)
	qu3 := QueryData{}
	err = protobuf.Decode(instID2, &qu3)
	require.NoError(t, err)
	log.Info("[INFO] Authorized query in the end", qu3)

	// instID3, err := cl.Bcl.ResolveInstanceID(cl.AllDarcIDs["A"], query.ID+"_auth")
	// require.Nil(t, err)
	// qu3, err := cl.GetQuery(instID3.Slice())
	// require.Nil(t, err)
	// require.Equal(t, 1, len(qu3.Storage))
	// require.Equal(t, resp4.QueryInstID, instID3)
	// require.Equal(t, []byte("Authorized"), qu3.Storage[1].Status)
	// require.Equal(t, "wsdf65k80h:A:patient_list", qu3.Storage[1].ID)

}

func TestClient_MedchainDeferredTxReject(t *testing.T) {
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
	// 1. Add Project B Darc
	// ------------------------------------------------------------------------
	// signer can only query certain things from the database
	rulesB := darc.InitRules([]darc.Identity{s.owner.Identity()}, []darc.Identity{cl.Signers[0].Identity()})
	actionsBAnd := "invoke:medchain.update,invoke:medchain.count_per_site,invoke:medchain.count_per_site_obfuscated," +
		"invoke:medchain.count_per_site_shuffled,invoke:medchain.count_per_site_shuffled_obfuscated,invoke:medchain.count_global," +
		"invoke:medchain.count_global_obfuscated"

	actionsBOr := "invoke:darc.evolve,spawn:deferred,invoke:deferred.addProof,invoke:deferred.execProposedTx,spawn:medchain,_name:deferred,spawn:naming,_name:medchain,spawn:darc"

	// all signers need to sign
	exprBAnd := expression.InitAndExpr(cl.Signers[0].Identity().String())

	// at least one signer need to sign
	exprBOr := expression.InitOrExpr(cl.Signers[0].Identity().String())
	cl.AllDarcs["B"], _ = cl.CreateDarc("Project B darc", rulesB, actionsBAnd, actionsBOr, exprBAnd, exprBOr)

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
	log.Info("[INFO] Darc for Project B is:", cl.AllDarcs["B"].String())
	// ------------------------------------------------------------------------
	// 2. Spwan query instances of MedChain contract
	// ------------------------------------------------------------------------

	req1 := &AddDeferredQueryRequest{}
	query := NewQuery("wsdf65k80h:B:patient_list", " ")
	req1.QueryID = query.ID
	req1.ClientID = cl.ClientID
	req1.DarcID = cl.AllDarcs["B"].GetBaseID()
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
	req2.Keys = cl.Signers[0]
	req2.ClientID = cl.ClientID
	req2.QueryInstID = req1.QueryInstID
	resp2, err := cl.AddSignatureToDeferredQuery(req2)
	require.NoError(t, err)
	require.NotNil(t, resp2)
	require.True(t, resp2.OK)

	// ------------------------------------------------------------------------
	// 4. Execute the query transaction
	// ------------------------------------------------------------------------
	req3 := &ExecuteDeferredTxRequest{}
	req3.ClientID = cl.ClientID
	req3.QueryInstID = req1.QueryInstID
	req3.QueryStatus = req1.QueryStatus
	resp3, err := cl.ExecDefferedQuery(req3)
	require.NoError(t, err)
	require.True(t, resp3.OK)
	require.Equal(t, resp2.QueryInstID, req3.QueryInstID)
	require.Equal(t, resp3.QueryInstID, req3.QueryInstID)
	require.Equal(t, resp1.QueryInstID, resp3.QueryInstID)

	iIDStr := resp3.QueryInstID.String()
	iIDBuf, err := hex.DecodeString(iIDStr)
	require.NoError(t, err)
	require.Equal(t, resp3.QueryInstID, byzcoin.NewInstanceID(iIDBuf))

	// ------------------------------------------------------------------------
	// 5. Check Authorizations
	// ------------------------------------------------------------------------
	err = cl.GetDarcRules(req1.QueryInstID)
	require.NoError(t, err)

	req4 := &AuthorizeQueryRequest{}
	req4.QueryID = query.ID
	req4.QueryInstID = req1.QueryInstID
	req4.DarcID = cl.AllDarcs["B"].GetBaseID()
	resp4, err := cl.AuthorizeQuery(req4)
	require.Nil(t, err)
	require.True(t, resp4.OK)
	require.Equal(t, 32, len(resp4.QueryInstID))
	require.NotEqual(t, req1.QueryInstID, resp4.QueryInstID)
	cl.Bcl.WaitPropagation(5)

	// Use the client API to get the query back
	// Resolve instance takes much time to run
	instaID, err := cl.Bcl.ResolveInstanceID(cl.AllDarcIDs["B"], query.ID+"_auth")
	require.Nil(t, err)
	_, err = cl.GetQuery(instaID.Slice())
	require.Nil(t, err)

}

func TestClient_MedchainDeferredMultiSigners(t *testing.T) {
	// ------------------------------------------------------------------------
	// 0. Set up and start service
	// ------------------------------------------------------------------------

	log.Info("[INFO] Starting the service")
	s, bcl, cl := newSer(t)
	require.Equal(t, s.owner, cl.Signers[0])
	leader := s.services[0]
	defer s.close()

	log.Info("[INFO] Starting ByzCoin Client")
	err := cl.Create()
	require.Nil(t, err)
	log.Info("[INFO] Signer counter", cl.signerCtrs)

	// ------------------------------------------------------------------------
	// 0. Add Project A Darc
	// ------------------------------------------------------------------------

	rulesA := darc.InitRules([]darc.Identity{s.owner.Identity()}, []darc.Identity{cl.Signers[0].Identity()})
	actionsAAnd := "spawn:medchain,invoke:medchain.patient_list,invoke:medchain.count_per_site,invoke:medchain.count_per_site_obfuscated," +
		"invoke:medchain.count_per_site_shuffled,invoke:medchain.count_per_site_shuffled_obfuscated,invoke:medchain.count_global," +
		"invoke:medchain.count_global_obfuscated"

	actionsAOr := "invoke:medchain.update,spawn:deferred,invoke:deferred.addProof,invoke:deferred.execProposedTx,spawn:darc,invoke:darc.evolve,_name:deferred,spawn:naming,_name:medchain,_config"

	// all signers need to sign
	exprAAnd := expression.InitAndExpr(cl.Signers[0].Identity().String())

	// at least one signer needs to sign
	exprAOr := expression.InitOrExpr(cl.Signers[0].Identity().String())
	cl.AllDarcs["A"], err = cl.CreateDarc("Project A darc", rulesA, actionsAAnd, actionsAOr, exprAAnd, exprAOr)
	require.NoError(t, err)

	// Verify the darc is correct
	require.Nil(t, cl.AllDarcs["A"].Verify(true))

	aDarcBuf, err := cl.AllDarcs["A"].ToProto()
	require.NoError(t, err)
	aDarcCopy, err := darc.NewFromProtobuf(aDarcBuf)
	require.NoError(t, err)
	require.True(t, cl.AllDarcs["A"].Equal(aDarcCopy))

	// Add darc to byzcoin
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
	log.Info("[INFO] Darc for Project is:", cl.AllDarcs["A"].String())

	// ------------------------------------------------------------------------
	// 1.  add new client and add new signer to darc
	// ------------------------------------------------------------------------
	log.Info("[INFO] Updating Genesis Darc")
	cl2, err := NewClient(bcl, s.roster.RandomServerIdentity(), "2")
	expr := expression.InitOrExpr(cl.Signers[0].Identity().String(), cl2.Signers[0].Identity().String())

	rGenDarc, err := lib.GetDarcByID(cl.Bcl, cl.GenDarcID)
	require.NoError(t, err)
	newDarc1, err := cl.AddRuleToDarc(rGenDarc, "spawn:"+ContractName, expr)
	require.NoError(t, err)
	newDarc1, err = cl.AddRuleToDarc(newDarc1, "invoke:"+ContractName+"."+"update", expr)
	require.NoError(t, err)
	newDarc1, err = cl.AddRuleToDarc(newDarc1, "invoke:"+ContractName+"."+"verifystatus", expr)
	require.NoError(t, err)
	newDarc1, err = cl.AddRuleToDarc(newDarc1, "_name:"+ContractName, expr)
	require.NoError(t, err)
	newDarc1, err = cl.AddRuleToDarc(newDarc1, "spawn:deferred", expr)
	require.NoError(t, err)
	newDarc1, err = cl.AddRuleToDarc(newDarc1, "invoke:deferred.addProof", expr)
	require.NoError(t, err)
	newDarc2, err := cl.AddRuleToDarc(newDarc1, "invoke:deferred.execProposedTx", expr)
	require.NoError(t, err)

	log.Info("[INFO] New genesis darc", newDarc2.String())

	log.Info("[INFO] Starting ByzCoin Client 2")
	cl2.GenDarcID = newDarc2.GetBaseID()
	err = cl2.Create()
	require.NoError(t, err)
	require.Equal(t, cl2.signerCtrs, []uint64([]uint64{0x1}))
	cl2.Bcl.WaitPropagation(1)
	require.Nil(t, err)
	require.NoError(t, err)

	darcActionsAAnd, err := cl.getDarcActions(actionsAAnd)
	require.NoError(t, err)
	darcActionsAOr, err := cl.getDarcActions(actionsAOr)
	require.NoError(t, err)
	err = cl.AddSignerToDarc("A", cl.AllDarcIDs["A"], darcActionsAAnd, cl2.Signers[0], 0)
	require.NoError(t, err)
	err = cl.AddSignerToDarc("A", cl.AllDarcIDs["A"], darcActionsAOr, cl2.Signers[0], 1)
	require.NoError(t, err)

	// ------------------------------------------------------------------------
	// 2. Spwan query instances of MedChain contract
	// ------------------------------------------------------------------------

	req1 := &AddDeferredQueryRequest{}
	query := NewQuery("wsdf65k80h:A:patient_list", " ")
	req1.QueryID = query.ID
	req1.ClientID = cl.ClientID
	req1.DarcID = cl.AllDarcIDs["A"]
	resp1, err := cl.SpawnDeferredQuery(req1)
	require.NoError(t, err)
	require.NotNil(t, resp1)
	require.NotNil(t, req1.QueryInstID)
	require.True(t, resp1.OK)
	require.Equal(t, []byte("Submitted"), req1.QueryStatus)

	result, err := cl.Bcl.GetDeferredDataAfter(resp1.QueryInstID, cl.Bcl.Latest)
	require.Nil(t, err)
	// Default MaxNumExecution should be 1
	require.Equal(t, result.MaxNumExecution, uint64(1))
	require.NotEmpty(t, result.InstructionHashes)

	cl.Bcl.WaitPropagation(1)

	//Fetch the index, and check it.
	idx := checkProof(t, cl, leader.omni, req1.QueryInstID.Slice(), cl.Bcl.ID)
	qu := byzcoin.DeferredData{}
	err = protobuf.Decode(idx, &qu)
	require.NoError(t, err)
	log.Info("[INFO] retrieved deferred data:", qu)

	dd, err := cl.Bcl.GetDeferredData(req1.QueryInstID)
	require.NoError(t, err)
	require.Equal(t, uint64(1), dd.MaxNumExecution)

	dd1, err := cl.Bcl.GetDeferredData(resp1.QueryInstID)
	log.Infof("Here is the deferred data after spawning it: \n%s", dd1)
	require.NoError(t, err)
	require.NotEmpty(t, dd1.InstructionHashes)
	require.Equal(t, uint64(1), dd1.MaxNumExecution)
	require.Empty(t, dd1.ExecResult)

	// ------------------------------------------------------------------------
	// 3. Add signature (i.e, add proof) to the deferred query instance
	// ------------------------------------------------------------------------
	req2 := &SignDeferredTxRequest{}
	req2.Keys = cl.Signers[0]
	req2.ClientID = cl.ClientID
	req2.QueryInstID = req1.QueryInstID

	resp2, err := cl.AddSignatureToDeferredQuery(req2)
	require.NoError(t, err)
	require.NotNil(t, resp2)
	require.True(t, resp2.OK)

	dd2, err := cl.Bcl.GetDeferredData(resp1.QueryInstID)
	log.Infof("Here is the deferred data after adding the first signature: \n%s", dd2)
	require.NoError(t, err)
	require.NotEmpty(t, dd2.InstructionHashes)
	require.Equal(t, uint64(1), dd2.MaxNumExecution)
	require.Empty(t, dd2.ExecResult)

	// ------------------------------------------------------------------------
	// 4. Execute the query transaction. This one should fail as one of the
	//    signers has not signed the instruction yet
	// ------------------------------------------------------------------------
	req3 := &ExecuteDeferredTxRequest{}
	req3.ClientID = cl.ClientID
	req3.QueryInstID = req1.QueryInstID
	req3.QueryStatus = req1.QueryStatus
	resp3, err := cl.ExecDefferedQuery(req3)
	log.Info("[INFO] resp3:", resp3)
	require.Error(t, err)
	require.Contains(t, err.Error(), "instruction verification: evaluating darc: expression evaluated to false")
	// require.True(t, resp3.OK)
	// require.Equal(t, resp3.QueryInstID, req3.QueryInstID)
	// require.Equal(t, resp3.QueryInstID, req3.QueryInstID)
	// require.Equal(t, resp1.QueryInstID, resp3.QueryInstID)

	// iIDStr := resp3.QueryInstID.String()
	// iIDBuf, err := hex.DecodeString(iIDStr)
	// require.NoError(t, err)
	// require.Equal(t, resp2.QueryInstID, byzcoin.NewInstanceID(iIDBuf))

	dd3, err := cl.Bcl.GetDeferredData(resp1.QueryInstID)
	log.Infof("Here is the deferred data after first execution: \n%s", dd3)
	require.NoError(t, err)
	require.NotEmpty(t, dd3.InstructionHashes)
	require.Equal(t, uint64(1), dd3.MaxNumExecution) //since the query has not been executed yet
	require.Empty(t, dd3.ExecResult)

	// ------------------------------------------------------------------------
	// 4. Add signature of user2 (i.e, add proof) to the deferred query instance.
	// ------------------------------------------------------------------------
	// namingTx, err := cl2.Bcl.CreateTransaction(
	// 	byzcoin.Instruction{
	// 		InstanceID: byzcoin.NewInstanceID(cl2.GenDarcID),
	// 		Spawn: &byzcoin.Spawn{
	// 			ContractID: byzcoin.ContractNamingID,
	// 		},
	// 		SignerCounter: cl2.IncrementCtrs(),
	// 	},
	// )
	// require.NoError(t, err)
	// log.Info("[INFO] (API) Spawning the instance of naming contract")
	// err = cl2.spawnTx(namingTx)
	// require.NoError(t, err)
	// log.Info("[INFO] (API) contract_name instance was added to the ledger")

	req4 := &SignDeferredTxRequest{}
	req4.Keys = cl2.Signers[0]
	req4.ClientID = cl2.ClientID
	req4.QueryInstID = req1.QueryInstID
	require.Equal(t, cl2.signerCtrs, []uint64([]uint64{0x1}))
	resp4, err := cl2.AddSignatureToDeferredQuery(req4)
	require.NoError(t, err)
	require.NotNil(t, resp4)
	require.True(t, resp4.OK)

	dd4, err := cl2.Bcl.GetDeferredData(resp1.QueryInstID)
	log.Infof("Here is the deferred data after adding second signature: \n%s", dd4)
	require.NoError(t, err)
	require.NotEmpty(t, dd4.InstructionHashes)
	require.Equal(t, uint64(1), dd4.MaxNumExecution)
	require.Empty(t, dd4.ExecResult)

	// ------------------------------------------------------------------------
	// 5. Execute the query transaction. This one should succeed
	// ------------------------------------------------------------------------
	req5 := &ExecuteDeferredTxRequest{}
	req5.ClientID = cl.ClientID
	req5.QueryInstID = req1.QueryInstID
	req5.QueryStatus = req1.QueryStatus
	resp5, err := cl.ExecDefferedQuery(req5)
	require.NoError(t, err)
	require.True(t, resp5.OK)
	require.Equal(t, resp2.QueryInstID, req5.QueryInstID)
	require.Equal(t, resp4.QueryInstID, req5.QueryInstID)
	require.Equal(t, resp1.QueryInstID, resp5.QueryInstID)

	dd5, err := cl.Bcl.GetDeferredData(resp1.QueryInstID)
	log.Infof("Here is the deferred data after final execution: \n%s", dd5)
	require.NoError(t, err)
	require.NotEmpty(t, dd5.InstructionHashes)
	require.Equal(t, uint64(0), dd5.MaxNumExecution)
	require.NotEmpty(t, dd5.ExecResult) //since the tranasction has been executed

	// ------------------------------------------------------------------------
	// 6. Check Authorizations
	// ------------------------------------------------------------------------
	err = cl.GetDarcRules(req1.QueryInstID)
	require.NoError(t, err)

	req6 := &AuthorizeQueryRequest{}
	req6.QueryID = query.ID
	req6.QueryInstID = req1.QueryInstID
	req6.DarcID = cl.AllDarcs["A"].GetBaseID()
	resp6, err := cl.AuthorizeQuery(req6)
	require.Nil(t, err)
	require.True(t, resp6.OK)
	require.Equal(t, 32, len(resp5.QueryInstID))
	require.NotEqual(t, req1.QueryInstID, resp6.QueryInstID)
	cl.Bcl.WaitPropagation(5)

	// Use the client API to get the query back
	// Resolve instance takes much time to run
	instaID, err := cl.Bcl.ResolveInstanceID(cl.AllDarcIDs["A"], query.ID+"_auth")
	require.Nil(t, err)
	_, err = cl.GetQuery(instaID.Slice())
	require.Nil(t, err)

}

func TestClient_IDSharing(t *testing.T) {

	// ------------------------------------------------------------------------
	// 0. Set up and start service
	// ------------------------------------------------------------------------

	log.Info("[INFO] Starting the service")
	s, _, cl := newSer(t)
	require.Equal(t, s.owner, cl.Signers[0])
	// leader := s.services[0]
	defer s.close()

	log.Info("[INFO] Starting ByzCoin Client")
	err := cl.Create()
	require.Nil(t, err)

	// ------------------------------------------------------------------------
	// 1. Add Project A Darc
	// ------------------------------------------------------------------------

	rulesA := darc.InitRules([]darc.Identity{s.owner.Identity()}, []darc.Identity{cl.Signers[0].Identity()})
	actionsAAnd := "invoke:medchain.patient_list,invoke:medchain.count_per_site,invoke:medchain.count_per_site_obfuscated," +
		"invoke:medchain.count_per_site_shuffled,invoke:medchain.count_per_site_shuffled_obfuscated,invoke:medchain.count_global," +
		"invoke:medchain.count_global_obfuscated,invoke:darc.evolve"

	actionsAOr := "spawn:deferred,invoke:deferred.addProof,invoke:deferred.execProposedTx,spawn:medchain,invoke:medchain.update,_name:deferred,spawn:naming,_name:medchain,_config"

	// all signers need to sign
	exprAAnd := expression.InitAndExpr(cl.Signers[0].Identity().String())

	// at least one signer need to sign
	exprAOr := expression.InitOrExpr(cl.Signers[0].Identity().String())
	cl.AllDarcs["A"], err = cl.CreateDarc("Project A darc", rulesA, actionsAAnd, actionsAOr, exprAAnd, exprAOr)
	require.NoError(t, err)

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
	log.Info("[INFO] Darc for Project A is:", cl.AllDarcs["A"].String())

	// ------------------------------------------------------------------------
	// 4. Propagate data and then Request shared data
	// ------------------------------------------------------------------------

	services := s.local.GetServices(s.hosts, sid)
	for _, s := range services {
		log.Info("[INFO] sending PropagateIDRequest 1 to", s)
		resp, err := s.(*Service).HandleGetSharedData(
			&GetSharedDataRequest{},
		)
		require.Empty(t, err)
		log.Info("[INFO] response is", resp.QueryInstIDs)
	}

	rep1 := PropagateIDReply{}
	err = cl.onetcl.SendProtobuf(s.roster.List[1], &PropagateIDRequest{ctx.Instructions[0].DeriveID(""), s.roster}, &rep1)
	require.NoError(t, err)
	services = s.local.GetServices(s.hosts, sid)
	for _, s := range services {
		log.Info("[INFO] sending PropagateIDRequest 1 to", s)
		resp, err := s.(*Service).HandleGetSharedData(
			&GetSharedDataRequest{},
		)
		require.NoError(t, err)
		require.NotEmpty(t, resp.QueryInstIDs)
		log.Info("[INFO] response is", resp.QueryInstIDs)
		fmt.Println(resp.QueryInstIDs)
	}

	type Client struct {
		*onet.Client
	}

	newCl := Client{Client: onet.NewClient(TSuite, ServiceName)}

	err = s.local.WaitDone(defaultBlockInterval)
	require.NoError(t, err)
	rep2 := PropagateIDReply{}
	err = newCl.SendProtobuf(s.roster.List[1], &PropagateIDRequest{ctx.Instructions[0].DeriveID(""), s.roster}, &rep2)
	require.NoError(t, err)
	err = newCl.SendProtobuf(s.roster.List[1], &PropagateIDRequest{byzcoin.InstanceID{}, s.roster}, &rep2)
	require.NoError(t, err)
	services = s.local.GetServices(s.hosts, sid)
	for _, s := range services {
		log.Info("[INFO] sending PropagateIDRequest 2 to service", s)
		resp, err := s.(*Service).HandleGetSharedData(
			&GetSharedDataRequest{},
		)
		require.Nil(t, err)
		require.NotEmpty(t, resp.QueryInstIDs)
		// require.Equal(t, 3, len(resp.QueryInstIDs))
		log.Info("[INFO] response is", resp.QueryInstIDs)
		fmt.Println(resp.QueryInstIDs)
	}

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
	actionsAAnd := "spawn:medchain,invoke:medchain.patient_list,invoke:medchain.count_per_site,invoke:medchain.count_per_site_obfuscated," +
		"invoke:medchain.count_per_site_shuffled,invoke:medchain.count_per_site_shuffled_obfuscated,invoke:medchain.count_global," +
		"invoke:medchain.count_global_obfuscated,invoke:darc.evolve"

	actionsAOr := "spawn:deferred,invoke:deferred.addProof,invoke:deferred.execProposedTx,invoke:medchain.update,_name:deferred,spawn:naming,_name:medchain"

	// all signers need to sign
	exprAAnd := expression.InitAndExpr(cl.Signers[0].Identity().String())

	// at least one signer need to sign
	exprAOr := expression.InitOrExpr(cl.Signers[0].Identity().String())
	cl.AllDarcs["A"], _ = cl.CreateDarc("Project A darc", rulesA, actionsAAnd, actionsAOr, exprAAnd, exprAOr)

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
	log.Info("[INFO] Darc for Project is:", cl.AllDarcs["A"].String())

	// ------------------------------------------------------------------------
	// 2. Add Project B Darc
	// ------------------------------------------------------------------------
	// signer can only query certain things from the database
	rulesB := darc.InitRules([]darc.Identity{s.owner.Identity()}, []darc.Identity{cl.Signers[0].Identity()})
	actionsBAnd := "invoke:medchain.count_per_site,invoke:medchain.count_per_site_obfuscated," +
		"invoke:medchain.count_per_site_shuffled,invoke:medchain.count_per_site_shuffled_obfuscated,invoke:medchain.count_global," +
		"invoke:medchain.count_global_obfuscated,invoke:darc.evolve"

	actionsBOr := "spawn:deferred,invoke:deferred.addProof,invoke:deferred.execProposedTx,spawn:medchain,invoke:medchain.update,_name:deferred,spawn:naming,_name:medchain"

	// all signers need to sign
	exprBAnd := expression.InitAndExpr(cl.Signers[0].Identity().String())

	// at least one signer need to sign
	exprBOr := expression.InitOrExpr(cl.Signers[0].Identity().String())
	cl.AllDarcs["B"], _ = cl.CreateDarc("Project B darc", rulesB, actionsBAnd, actionsBOr, exprBAnd, exprBOr)
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
	log.Info("[INFO] Darc for Project B is:", cl.AllDarcs["B"].String())

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
			project := randomString(1, "AB")
			qu := NewQuery(randomString(10, "")+":"+project+":"+randomAction(), "Submitted")
			req := &AuthorizeQueryRequest{}
			if project == "A" {
				req.DarcID = cl.AllDarcs["A"].GetBaseID()
			} else {
				req.DarcID = cl.AllDarcs["B"].GetBaseID()
			}
			req.QueryInstID = id1
			req.QueryID = qu.ID
			req.QueryStatus = qu.Status
			resp, err := cl.AuthorizeQuery(req)
			require.NoError(t, err)
			require.True(t, resp.OK)
			require.NotEqual(t, id1, resp.QueryInstID)
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

	// cl.GMsg = s.req
	cl.Signers = []darc.Signer{s.owner}
	cl.GenDarcID = s.genDarc.GetBaseID()
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
