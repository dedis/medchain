package contract

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.dedis.ch/cothority/v3"
	"go.dedis.ch/cothority/v3/byzcoin"
	"go.dedis.ch/cothority/v3/darc"
	"go.dedis.ch/onet/v3"
	"go.dedis.ch/protobuf"
)

func TestSpawn(t *testing.T) {
	// Create a new ledger and prepare for proper closing
	bct := newBCTest(t)
	bct.cl.UseNode(0)
	defer bct.Close()

	// Create a new instance with two key/values:
	args := byzcoin.Arguments{
		{
			Name:  "queryID1",
			Value: []byte("Approved"),
		},
		{
			Name:  "queryID2",
			Value: []byte("Rejected"),
		},
	}
	// And send it to the ledger.
	instID := bct.createInstance(t, args)

	// Get the proof from byzcoin
	reply, err := bct.cl.GetProof(instID.Slice())
	require.Nil(t, err)
	// Make sure the proof is a matching proof and not a proof of absence.
	pr := reply.Proof
	require.True(t, pr.InclusionProof.Match(instID.Slice()))

	// Get the raw values of the proof.
	_, val, _, _, err := pr.KeyValue()
	require.Nil(t, err)
	// And decode the buffer to a QueryData
	cs := QueryData{}
	err = protobuf.Decode(val, &cs)
	require.Nil(t, err)
	// Verify all values are in there.
	for i, s := range cs.Storage {
		require.Equal(t, args[i].Name, s.ID)
		require.Equal(t, args[i].Value, []byte(s.Status))
	}
}

func TestInvoke(t *testing.T) {
	// Create a new ledger and prepare for proper closing
	bct := newBCTest(t)
	bct.cl.UseNode(0)
	defer bct.Close()

	// define varibales
	var queryArgs1 = Query{}
	queryArgs1.ID = "query1"
	queryArgs1.Status = "Rejected"

	var queryArgs2 = Query{}
	queryArgs2.ID = "query2"
	queryArgs2.Status = "Approved"

	// Create a new instance with two key/values:
	args := byzcoin.Arguments{
		{
			Name:  queryArgs1.ID,
			Value: []byte(queryArgs1.Status),
		},
		{
			Name:  queryArgs2.ID,
			Value: []byte(queryArgs2.Status),
		},
	}
	// And send it to the ledger.
	instID := bct.createInstance(t, args)

	// Get the proof from byzcoin
	reply, err := bct.cl.GetProof(instID.Slice())
	require.Nil(t, err)
	pr1 := reply.Proof

	// Change "query2" and add a "query3"
	args = byzcoin.Arguments{
		{
			Name:  queryArgs2.ID,
			Value: []byte("Executed"),
		},
		{
			Name:  "query3",
			Value: []byte("Approved"),
		},
	}
	bct.updateInstance(t, instID, args)

	// Store the values of the previous proof in 'values'
	_, v1, _, _, err := pr1.KeyValue()
	require.Nil(t, err)
	var v2 []byte
	prRep2, err := bct.cl.GetProof(instID.Slice())
	require.Nil(t, err)
	_, v2, _, _, err = prRep2.Proof.KeyValue()
	require.NotEqual(t, 0, bytes.Compare(v1, v2), "didn't include new values")

	// Read the content of the instance back into a structure.
	var newArgs QueryData
	err = protobuf.Decode(v2, &newArgs)
	require.Nil(t, err)
	// Verify the content is as it is supposed to be.
	require.Equal(t, 3, len(newArgs.Storage))
	require.Equal(t, "query1", newArgs.Storage[0].ID)
	require.Equal(t, "Rejected", newArgs.Storage[0].Status)
	require.Equal(t, "query2", newArgs.Storage[1].ID)
	require.Equal(t, "Executed", newArgs.Storage[1].Status)
}

func TestUpdate(t *testing.T) {
	cs := QueryData{
		Storage: []Query{{
			ID:     "query3",
			Status: "Approved",
		}},
	}

	cs.Update(byzcoin.Arguments{{
		Name:  "query3",
		Value: []byte("Executed"),
	}})
	require.Equal(t, 1, len(cs.Storage))
	require.Equal(t, "Executed", cs.Storage[0].Status)

	// query1 does not exist, thus will be added
	cs.Update(byzcoin.Arguments{{
		Name:  "query1",
		Value: []byte("Approved"),
	}})
	require.Equal(t, 2, len(cs.Storage))

	// query2 does not exist, thus will be added
	cs.Update(byzcoin.Arguments{{
		Name:  "query2",
		Value: []byte("Executed"),
	}})
	require.Equal(t, 3, len(cs.Storage))
	require.Equal(t, "query3", cs.Storage[0].ID)
	require.Equal(t, "Executed", cs.Storage[0].Status)
	require.Equal(t, "query1", cs.Storage[1].ID)
	require.Equal(t, "Approved", cs.Storage[1].Status)
	require.Equal(t, "query2", cs.Storage[2].ID)
	require.Equal(t, "Executed", cs.Storage[2].Status)

	cs.Update(byzcoin.Arguments{{
		Name:  "query1",
		Value: []byte("Executed"),
	}})
	require.Equal(t, 3, len(cs.Storage))
	require.Equal(t, "query1", cs.Storage[1].ID)
	require.Equal(t, "Executed", cs.Storage[1].Status)
}

func TestVerifyStatus(t *testing.T) {
	// Add query1 to the ledger
	cs := QueryData{
		Storage: []Query{{
			ID:     "query1",
			Status: "Approved",
		}},
	}

	// query2 does not exist, thus will be added
	cs.Update(byzcoin.Arguments{{
		Name:  "query2",
		Value: []byte("Rejected"),
	}})

	// check the skipchain (items in the ledger)
	require.Equal(t, 2, len(cs.Storage))
	require.Equal(t, "query1", cs.Storage[0].ID)
	require.Equal(t, "Approved", cs.Storage[0].Status)
	require.Equal(t, "query2", cs.Storage[1].ID)
	require.Equal(t, "Rejected", cs.Storage[1].Status)

	// Check the status of query1
	err := cs.VerifyStatus(byzcoin.Arguments{{
		Name: "query1",
	}})
	//The status of query1 is Approved, so should return nil
	assert.Nil(t, err)

	// Check the status of query2
	err = cs.VerifyStatus(byzcoin.Arguments{{
		Name: "query2",
	}})

	//The status of query2 is not Approved, so should return some error (i.e., not nil)
	assert.NotNil(t, err)

	// Check the status of query3
	err = cs.VerifyStatus(byzcoin.Arguments{{
		Name: "query3",
	}})

	//query3 does not exist, so chould return error
	assert.NotNil(t, err)

}

// bcTest is used here to provide some simple test structure for different
// tests.
type bcTest struct {
	local   *onet.LocalTest
	signer  darc.Signer
	servers []*onet.Server
	roster  *onet.Roster
	cl      *byzcoin.Client
	gMsg    *byzcoin.CreateGenesisBlock
	gDarc   *darc.Darc
	ct      uint64
}

func newBCTest(t *testing.T) (out *bcTest) {
	out = &bcTest{}
	// First create a local test environment with three nodes.
	out.local = onet.NewTCPTest(cothority.Suite)

	out.signer = darc.NewSignerEd25519(nil, nil)
	out.servers, out.roster, _ = out.local.GenTree(3, true)

	// Then create a new ledger with the genesis darc having the right
	// to create and update key-value contracts.
	var err error
	out.gMsg, err = byzcoin.DefaultGenesisMsg(byzcoin.CurrentVersion, out.roster,
		[]string{"spawn:queryContract", "invoke:queryContract.update"}, out.signer.Identity())
	require.Nil(t, err)
	out.gDarc = &out.gMsg.GenesisDarc

	// This BlockInterval is good for testing, but in real world applications this
	// should be more like 5 seconds.
	out.gMsg.BlockInterval = time.Second / 2

	out.cl, _, err = byzcoin.NewLedger(out.gMsg, false)
	require.Nil(t, err)
	out.ct = 1

	return out
}

func (bct *bcTest) Close() {
	bct.local.CloseAll()
}

func (bct *bcTest) createInstance(t *testing.T, args byzcoin.Arguments) byzcoin.InstanceID {
	ctx, err := bct.cl.CreateTransaction(byzcoin.Instruction{
		InstanceID:    byzcoin.NewInstanceID(bct.gDarc.GetBaseID()),
		SignerCounter: []uint64{bct.ct},
		Spawn: &byzcoin.Spawn{
			ContractID: MedchainContractID,
			Args:       args,
		},
	})
	require.NoError(t, err)

	bct.ct++
	// And we need to sign the instruction with the signer that has his
	// public key stored in the darc.
	require.NoError(t, ctx.FillSignersAndSignWith(bct.signer))

	// Sending this transaction to ByzCoin does not directly include it in the
	// global state - first we must wait for the new block to be created.
	_, err = bct.cl.AddTransactionAndWait(ctx, 5)
	require.Nil(t, err)
	return ctx.Instructions[0].DeriveID("")
}

func (bct *bcTest) updateInstance(t *testing.T, instID byzcoin.InstanceID, args byzcoin.Arguments) {
	ctx, err := bct.cl.CreateTransaction(byzcoin.Instruction{
		InstanceID:    instID,
		SignerCounter: []uint64{bct.ct},
		Invoke: &byzcoin.Invoke{
			ContractID: MedchainContractID,
			Command:    "update",
			Args:       args,
		},
	})
	require.NoError(t, err)

	bct.ct++
	// And we need to sign the instruction with the signer that has his
	// public key stored in the darc.
	require.NoError(t, ctx.FillSignersAndSignWith(bct.signer))

	// Sending this transaction to ByzCoin does not directly include it in the
	// global state - first we must wait for the new block to be created.
	_, err = bct.cl.AddTransactionAndWait(ctx, 5)
	require.Nil(t, err)
}
