package admin

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.dedis.ch/cothority/v3"
	"go.dedis.ch/cothority/v3/byzcoin"
	"go.dedis.ch/cothority/v3/darc"
	"go.dedis.ch/onet/v3"
	"go.dedis.ch/onet/v3/log"
	"go.dedis.ch/protobuf"
	"golang.org/x/xerrors"
)

// A structure that hold the exepcted state of the contract to compare with the values in the global state
type AccessRightTestContext struct {
	ar      AccessRight
	signer  darc.Signer
	counter uint64
	cl      *byzcoin.Client
}

func TestAccessRight_Spawn(t *testing.T) {
	log.Lvl1("[INFO] Setting up")
	local := onet.NewTCPTest(cothority.Suite)
	defer local.CloseAll()

	signer := darc.NewSignerEd25519(nil, nil)
	_, roster, _ := local.GenTree(3, true)

	genesisMsg, err := byzcoin.DefaultGenesisMsg(byzcoin.CurrentVersion, roster,
		[]string{"spawn:accessright"}, signer.Identity())
	require.Nil(t, err)
	gDarc := &genesisMsg.GenesisDarc

	genesisMsg.BlockInterval = time.Second

	cl, _, err := byzcoin.NewLedger(genesisMsg, false)
	require.Nil(t, err)
	log.Lvl1("[INFO] Spawn access right contract")
	val := AccessRight{[]string{"1:1"}, []string{"count_per_site_shuffled:count_global"}}
	myvalue, err := protobuf.Encode(&val)
	require.NoError(t, err)
	ctx, err := cl.CreateTransaction(byzcoin.Instruction{
		InstanceID: byzcoin.NewInstanceID(gDarc.GetBaseID()),
		Spawn: &byzcoin.Spawn{
			ContractID: ContractAccessRightID,
			Args: []byzcoin.Argument{{
				Name:  "ar",
				Value: myvalue,
			}},
		},
		SignerCounter: []uint64{1},
	})
	require.NoError(t, err)
	require.Nil(t, ctx.FillSignersAndSignWith(signer))
	_, err = cl.AddTransaction(ctx)
	require.Nil(t, err)
	// Check that the contract is in the global state
	pr, err := cl.WaitProof(byzcoin.NewInstanceID(ctx.Instructions[0].DeriveID("").Slice()), 2*genesisMsg.BlockInterval, myvalue)
	require.Nil(t, err)
	require.True(t, pr.InclusionProof.Match(ctx.Instructions[0].DeriveID("").Slice()))
	v0, _, _, err := pr.Get(ctx.Instructions[0].DeriveID("").Slice())
	require.Nil(t, err)
	received := AccessRight{}
	err = protobuf.Decode(v0, &received)
	require.NoError(t, err)
	require.Equal(t, val, received) // Check that the AccessRight value is the same in the global state as the spawned one
}

func TestAccessRight_Invoke(t *testing.T) {
	log.Lvl1("[INFO] Setting up")
	local := onet.NewTCPTest(cothority.Suite)
	defer local.CloseAll()

	signer := darc.NewSignerEd25519(nil, nil)
	_, roster, _ := local.GenTree(3, true)

	genesisMsg, err := byzcoin.DefaultGenesisMsg(byzcoin.CurrentVersion, roster,
		[]string{"spawn:accessright", "invoke:accessright.add", "invoke:accessright.update", "invoke:accessright.delete"}, signer.Identity())
	require.Nil(t, err)
	gDarc := &genesisMsg.GenesisDarc

	genesisMsg.BlockInterval = time.Second

	cl, _, err := byzcoin.NewLedger(genesisMsg, false)
	require.Nil(t, err)
	// instantiate a test context to verify the global state after several interactions with the contract
	arctx := AccessRightTestContext{AccessRight{[]string{"1:1"}, []string{"count_per_site_shuffled:count_global"}}, signer, 2, cl}
	log.Lvl1("[INFO] Spawn access right contract")
	val := AccessRight{[]string{"1:1"}, []string{"count_per_site_shuffled:count_global"}}
	myvalue, err := protobuf.Encode(&val)
	require.NoError(t, err)
	ctx, err := cl.CreateTransaction(byzcoin.Instruction{
		InstanceID: byzcoin.NewInstanceID(gDarc.GetBaseID()),
		Spawn: &byzcoin.Spawn{
			ContractID: ContractAccessRightID,
			Args: []byzcoin.Argument{{
				Name:  "ar",
				Value: myvalue,
			}},
		},
		SignerCounter: []uint64{1},
	})
	require.NoError(t, err)
	require.Nil(t, ctx.FillSignersAndSignWith(signer))
	myID := ctx.Instructions[0].DeriveID("")

	_, err = cl.AddTransactionAndWait(ctx, 1)
	require.Nil(t, err)
	err = local.WaitDone(genesisMsg.BlockInterval)
	require.Nil(t, err)
	pr, err := cl.GetProof(myID.Slice())
	require.Nil(t, err)

	vv, _, _, err := pr.Proof.Get(myID.Slice())
	require.NoError(t, err)
	received := AccessRight{}
	err = protobuf.Decode(vv, &received)
	require.Equal(t, val, received)

	log.Lvl1("[INFO] Invoke accessright.add")
	err = arctx.addAccess("2:1", "patient_list,count_per_site,count_per_site_obfuscated,count_per_site_shuffled", myID)
	require.NoError(t, err)
	log.Lvl1("[INFO] Invoke accessright.add")
	err = arctx.addAccess("3:2", "count_per_site_obfuscated,count_per_site_shuffled", myID)
	require.NoError(t, err)
	log.Lvl1("[INFO] Invoke accessright.update")
	err = arctx.modifyAccess("3:2", "patient_list,count_per_site_obfuscated", myID)
	require.NoError(t, err)
	log.Lvl1("[INFO] Invoke accessright.add")
	err = arctx.addAccess("4:1", "count_per_site_obfuscated,count_global_obfuscated", myID)
	require.NoError(t, err)
	log.Lvl1("[INFO] Invoke accessright.delete")
	err = arctx.deleteAccess("3:2", myID)
	require.NoError(t, err)
	err = local.WaitDone(genesisMsg.BlockInterval)
	pr2, err := cl.GetProof(myID.Slice())
	require.Nil(t, err)
	require.True(t, pr2.Proof.InclusionProof.Match(myID.Slice()))

	v2, _, _, err := pr2.Proof.Get(myID.Slice())
	require.Nil(t, err)

	received = AccessRight{}
	err = protobuf.Decode(v2, &received)
	log.Lvl1("[INFO] Verify global state")
	require.Equal(t, arctx.ar, received) // verify that the expected state of the access right is the same as the instance in the global state
}

func TestAccessRight_Set(t *testing.T) {
	log.Lvl1("[INFO] Setting up")
	local := onet.NewTCPTest(cothority.Suite)
	defer local.CloseAll()

	signer := darc.NewSignerEd25519(nil, nil)
	_, roster, _ := local.GenTree(3, true)

	genesisMsg, err := byzcoin.DefaultGenesisMsg(byzcoin.CurrentVersion, roster,
		[]string{"spawn:accessright", "invoke:accessright.add", "invoke:accessright.update", "invoke:accessright.delete"}, signer.Identity())
	require.Nil(t, err)
	gDarc := &genesisMsg.GenesisDarc

	genesisMsg.BlockInterval = time.Second

	cl, _, err := byzcoin.NewLedger(genesisMsg, false)
	require.Nil(t, err)
	// instantiate a test context to verify the global state after several interactions with the contract
	arctx := AccessRightTestContext{AccessRight{[]string{"1:1"}, []string{"count_per_site_shuffled,count_global"}}, signer, 2, cl}
	log.Lvl1("[INFO] Spawn access right contract")
	val := AccessRight{[]string{"1:1"}, []string{"count_per_site_shuffled,count_global"}}
	myvalue, err := protobuf.Encode(&val)
	require.NoError(t, err)
	ctx, err := cl.CreateTransaction(byzcoin.Instruction{
		InstanceID: byzcoin.NewInstanceID(gDarc.GetBaseID()),
		Spawn: &byzcoin.Spawn{
			ContractID: ContractAccessRightID,
			Args: []byzcoin.Argument{{
				Name:  "ar",
				Value: myvalue,
			}},
		},
		SignerCounter: []uint64{1},
	})
	require.NoError(t, err)
	require.Nil(t, ctx.FillSignersAndSignWith(signer))
	myID := ctx.Instructions[0].DeriveID("")

	_, err = cl.AddTransactionAndWait(ctx, 1)
	require.Nil(t, err)
	err = local.WaitDone(genesisMsg.BlockInterval)
	require.Nil(t, err)
	pr, err := cl.GetProof(myID.Slice())
	require.Nil(t, err)

	vv, _, _, err := pr.Proof.Get(myID.Slice())
	require.NoError(t, err)
	received := AccessRight{}
	err = protobuf.Decode(vv, &received)
	require.Equal(t, val, received)

	log.Lvl1("[INFO] Invoke accessright.add with id 1:1 already registered in the state : Expected to fail")
	err = arctx.addAccess("1:1", "patient_list,count_per_site,count_per_site_obfuscated,count_per_site_shuffled", myID)
	require.Error(t, err)
	log.Lvl1("[INFO] Invoke accessright.modify with id 3:2 not registered in the state : Expected to fail")
	err = arctx.modifyAccess("3:2", "patient_list", myID)
	require.Error(t, err)
	log.Lvl1("[INFO] Invoke accessright.modify with id 5:3 not registered in the state : Expected to fail")
	err = arctx.deleteAccess("5:3", myID)
	require.Error(t, err)
	err = local.WaitDone(genesisMsg.BlockInterval)
	pr2, err := cl.GetProof(myID.Slice())
	require.Nil(t, err)
	require.True(t, pr2.Proof.InclusionProof.Match(myID.Slice()))

	v2, _, _, err := pr2.Proof.Get(myID.Slice())
	require.Nil(t, err)

	received = AccessRight{}
	err = protobuf.Decode(v2, &received)
	log.Lvl1("[INFO] Verify global state")
	require.Equal(t, arctx.ar, received) // verify that the expected state of the access right is the same as the instance in the global state

}

// ------------------------------------------------------------------------
// Helper testing methods
// ------------------------------------------------------------------------

// This is a helper method that execute transactions and modify the local expected version of the AccessRight struct. This method locally act as the invoke:accessright.add method of the
// access right contract.
func (arctx *AccessRightTestContext) addAccess(id, access string, myID byzcoin.InstanceID) error {
	ctx2, err := arctx.cl.CreateTransaction(byzcoin.Instruction{
		InstanceID: myID,
		Invoke: &byzcoin.Invoke{
			ContractID: ContractAccessRightID,
			Command:    "add",
			Args: []byzcoin.Argument{{
				Name:  "id",
				Value: []byte(id),
			},
				{
					Name:  "ar",
					Value: []byte(access),
				}},
		},
		SignerCounter: []uint64{arctx.counter},
	})
	if err != nil {
		return xerrors.Errorf("Creating:  %w", err)
	}
	idx, _ := Find(arctx.ar.Ids, id)
	if idx == -1 {
		arctx.ar.Ids = append(arctx.ar.Ids, id)
		arctx.ar.Access = append(arctx.ar.Access, access)
	}
	err = ctx2.FillSignersAndSignWith(arctx.signer)
	if err != nil {
		return xerrors.Errorf("Signing:  %w", err)
	}
	_, err = arctx.cl.AddTransactionAndWait(ctx2, 10)
	if err != nil {
		return xerrors.Errorf("Adding transaction:  %w", err)
	}
	arctx.counter++
	return nil
}

// This is a helper method that execute transactions and modify the local expected version of the AccessRight struct. This method locally act as the invoke:accessright.update method of the
// access right contract.
func (arctx *AccessRightTestContext) modifyAccess(id, access string, myID byzcoin.InstanceID) error {
	ctx2, err := arctx.cl.CreateTransaction(byzcoin.Instruction{
		InstanceID: myID,
		Invoke: &byzcoin.Invoke{
			ContractID: ContractAccessRightID,
			Command:    "update",
			Args: []byzcoin.Argument{{
				Name:  "id",
				Value: []byte(id),
			},
				{
					Name:  "ar",
					Value: []byte(access),
				}},
		},
		SignerCounter: []uint64{arctx.counter},
	})
	idx, _ := Find(arctx.ar.Ids, id)
	if idx != -1 {
		arctx.ar.Access[idx] = access
	}
	err = ctx2.FillSignersAndSignWith(arctx.signer)
	if err != nil {
		return xerrors.Errorf("Signing:  %w", err)
	}
	_, err = arctx.cl.AddTransactionAndWait(ctx2, 10)
	if err != nil {
		return xerrors.Errorf("Adding transaction:  %w", err)
	}
	arctx.counter++
	return nil
}

// This is a helper method that execute transactions and modify the local expected version of the AccessRight struct. This method locally act as the invoke:accessright.delete method of the
// access right contract.
func (arctx *AccessRightTestContext) deleteAccess(id string, myID byzcoin.InstanceID) error {
	ctx2, err := arctx.cl.CreateTransaction(byzcoin.Instruction{
		InstanceID: myID,
		Invoke: &byzcoin.Invoke{
			ContractID: ContractAccessRightID,
			Command:    "delete",
			Args: []byzcoin.Argument{{
				Name:  "id",
				Value: []byte(id),
			},
			},
		},
		SignerCounter: []uint64{arctx.counter},
	})
	idx, _ := Find(arctx.ar.Ids, id)
	if idx != -1 {
		arctx.ar.Access = append(arctx.ar.Access[:idx], arctx.ar.Access[idx+1:]...)
		arctx.ar.Ids = append(arctx.ar.Ids[:idx], arctx.ar.Ids[idx+1:]...)
	}
	err = ctx2.FillSignersAndSignWith(arctx.signer)
	if err != nil {
		return xerrors.Errorf("Signing:  %w", err)
	}
	_, err = arctx.cl.AddTransactionAndWait(ctx2, 10)
	if err != nil {
		return xerrors.Errorf("Adding transaction:  %w", err)
	}
	arctx.counter++
	return nil
}
