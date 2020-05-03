package admin

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.dedis.ch/cothority/v3"
	"go.dedis.ch/cothority/v3/byzcoin"
	"go.dedis.ch/cothority/v3/byzcoin/bcadmin/lib"
	"go.dedis.ch/cothority/v3/darc"
	"go.dedis.ch/onet/v3"
	"go.dedis.ch/onet/v3/log"
)

func TestAddAdminsToDarc(t *testing.T) {
	// ------------------------------------------------------------------------
	// 0. Set up
	// ------------------------------------------------------------------------
	log.Lvl1("[INFO] Create admin darc")
	local := onet.NewTCPTest(cothority.Suite)
	defer local.CloseAll()

	superAdmin := darc.NewSignerEd25519(nil, nil)
	_, roster, _ := local.GenTree(5, true)

	genesisMsg, err := byzcoin.DefaultGenesisMsg(byzcoin.CurrentVersion, roster,
		[]string{"spawn:value", "spawn:deferred", "invoke:deferred.addProof",
			"invoke:deferred.execProposedTx"}, superAdmin.Identity())
	require.NoError(t, err)
	gDarc := &genesisMsg.GenesisDarc

	genesisMsg.BlockInterval = time.Second / 5
	bcl, _, err := byzcoin.NewLedger(genesisMsg, false)
	require.NoError(t, err)
	log.Lvl1("[INFO] Create admin client")
	admcl, err := NewClientWithAuth(bcl, &superAdmin)
	require.NoError(t, err)
	require.Equal(t, superAdmin, admcl.AuthKey())

	// ------------------------------------------------------------------------
	// 1. Spawn admin darc
	// ------------------------------------------------------------------------

	log.Lvl1("[INFO] Spawn admin darc")
	adminDarc, err := admcl.SpawnNewAdminDarc()
	require.NoError(t, err)
	admcl.bcl.WaitPropagation(1)
	_, err = lib.GetDarcByID(admcl.bcl, gDarc.GetID()) // Verify that the darc is really in the global state
	require.NoError(t, err)

	// ------------------------------------------------------------------------
	// 2. Add a new admin to the admin darc
	// ------------------------------------------------------------------------

	log.Lvl1("[INFO] Create new admin client (admcl2)")
	admcl2, err := NewClient(bcl)
	require.NoError(t, err)
	log.Lvl1("[INFO] Create deffered tx to add admcl2 identity")
	id, err := admcl.AddAdminToAdminDarc(adminDarc.GetBaseID(), admcl2.AuthKey().Identity())
	require.NoError(t, err)
	admcl.bcl.WaitPropagation(1)
	_, err = admcl2.bcl.GetDeferredData(id)
	require.NoError(t, err)
	log.Lvl1("[INFO] Admin 1 sign the transaction")
	admcl.AddSignatureToDefferedTx(id)
	log.Lvl1("[INFO] Admin 1 try to exec the transaction")
	err = admcl.ExecDefferedTx(id)
	require.NoError(t, err)
	log.Lvl1("[INFO] Tx successfully executed, admin 3 has been added to the admin darc")
	_, err = lib.GetDarcByID(admcl.bcl, adminDarc.GetID()) // Verify that the darc is really in the global state
	require.NoError(t, err)

	// ------------------------------------------------------------------------
	// 2. Add a new admin to the admin darc
	// ------------------------------------------------------------------------

	log.Lvl1("[INFO] Create new admin client (admcl3)")
	admcl3, err := NewClient(bcl)
	require.NoError(t, err)
	log.Lvl1("[INFO] Create deffered tx to add admcl3 identity")
	id, err = admcl.AddAdminToAdminDarc(adminDarc.GetBaseID(), admcl3.AuthKey().Identity())
	require.NoError(t, err)
	admcl.bcl.WaitPropagation(1)
	_, err = admcl2.bcl.GetDeferredData(id)
	require.NoError(t, err)
	log.Lvl1("[INFO] Admin 1 sign the transaction")
	err = admcl.AddSignatureToDefferedTx(id)
	require.NoError(t, err)
	log.Lvl1("[INFO] Admin 1 try to exec the transaction but threshold of signature not reached")
	err = admcl.ExecDefferedTx(id)
	require.Error(t, err)
	log.Lvl1("[INFO] Admin 2 sign the transaction")
	err = admcl2.AddSignatureToDefferedTx(id)
	require.NoError(t, err)
	log.Lvl1("[INFO] Admin 2 try to exec the transaction")
	err = admcl.ExecDefferedTx(id)
	require.NoError(t, err)
	log.Lvl1("[INFO] Admin 2 try to exec the transaction")
	_, err = lib.GetDarcByID(admcl.bcl, adminDarc.GetID())
	require.NoError(t, err)
	log.Lvl1("[INFO] Tx successfully executed, admin 3 has been added to the admin darc")

	// ------------------------------------------------------------------------
	// 2. Add a new admin to the admin darc
	// ------------------------------------------------------------------------

	log.Lvl1("[INFO] Create new admin client (admcl4)")
	admcl4, err := NewClient(bcl)
	require.NoError(t, err)
	log.Lvl1("[INFO] Create deffered tx to add admcl4 identity")
	id, err = admcl.AddAdminToAdminDarc(adminDarc.GetBaseID(), admcl4.AuthKey().Identity())
	require.NoError(t, err)
	admcl.bcl.WaitPropagation(1)
	_, err = admcl2.bcl.GetDeferredData(id)
	require.NoError(t, err)
	log.Lvl1("[INFO] Admin 1 sign the transaction")
	err = admcl.AddSignatureToDefferedTx(id)
	require.NoError(t, err)
	log.Lvl1("[INFO] Admin 3 sign the transaction")
	err = admcl3.AddSignatureToDefferedTx(id)
	require.NoError(t, err)
	log.Lvl1("[INFO] Admin 2 sign the transaction")
	err = admcl2.AddSignatureToDefferedTx(id)
	require.NoError(t, err)
	log.Lvl1("[INFO] Admin 3 try to execute the transaction")
	err = admcl3.ExecDefferedTx(id)
	require.NoError(t, err)
	log.Lvl1("[INFO] Tx successfully executed, admin 4 has been added to the admin darc")
	_, err = lib.GetDarcByID(admcl.bcl, adminDarc.GetID())
	require.NoError(t, err)
}

func TestRemovingAdminsToDarc(t *testing.T) {
	// ------------------------------------------------------------------------
	// 0. Set up
	// ------------------------------------------------------------------------
	log.Lvl1("[INFO] Create admin darc")
	local := onet.NewTCPTest(cothority.Suite)
	defer local.CloseAll()

	superAdmin := darc.NewSignerEd25519(nil, nil)
	_, roster, _ := local.GenTree(5, true)

	genesisMsg, err := byzcoin.DefaultGenesisMsg(byzcoin.CurrentVersion, roster,
		[]string{"spawn:value", "spawn:deferred", "invoke:deferred.addProof",
			"invoke:deferred.execProposedTx"}, superAdmin.Identity())
	require.NoError(t, err)
	gDarc := &genesisMsg.GenesisDarc

	genesisMsg.BlockInterval = time.Second / 5
	bcl, _, err := byzcoin.NewLedger(genesisMsg, false)
	require.NoError(t, err)
	log.Lvl1("[INFO] Create admin client")
	admcl, err := NewClientWithAuth(bcl, &superAdmin)
	require.NoError(t, err)
	require.Equal(t, superAdmin, admcl.AuthKey())

	// ------------------------------------------------------------------------
	// 1. Spawn admin darc
	// ------------------------------------------------------------------------

	log.Lvl1("[INFO] Spawn admin darc")
	adminDarc, err := admcl.SpawnNewAdminDarc()
	require.NoError(t, err)
	admcl.bcl.WaitPropagation(1)
	_, err = lib.GetDarcByID(admcl.bcl, gDarc.GetID()) // Verify that the darc is really in the global state
	require.NoError(t, err)

	// ------------------------------------------------------------------------
	// 2. Add a new admin to the admin darc
	// ------------------------------------------------------------------------

	log.Lvl1("[INFO] Create new admin client (admcl2)")
	admcl2, err := NewClient(bcl)
	require.NoError(t, err)
	log.Lvl1("[INFO] Create deffered tx to add admcl2 identity")
	id, err := admcl.AddAdminToAdminDarc(adminDarc.GetBaseID(), admcl2.AuthKey().Identity())
	require.NoError(t, err)
	admcl.bcl.WaitPropagation(1)
	_, err = admcl2.bcl.GetDeferredData(id)
	require.NoError(t, err)
	log.Lvl1("[INFO] Admin 1 sign the transaction")
	admcl.AddSignatureToDefferedTx(id)
	log.Lvl1("[INFO] Admin 1 try to exec the transaction")
	err = admcl.ExecDefferedTx(id)
	require.NoError(t, err)
	log.Lvl1("[INFO] Tx successfully executed, admin 3 has been added to the admin darc")
	_, err = lib.GetDarcByID(admcl.bcl, adminDarc.GetID()) // Verify that the darc is really in the global state
	require.NoError(t, err)

	// ------------------------------------------------------------------------
	// 2. Add a new admin to the admin darc
	// ------------------------------------------------------------------------

	log.Lvl1("[INFO] Create new admin client (admcl3)")
	admcl3, err := NewClient(bcl)
	require.NoError(t, err)
	log.Lvl1("[INFO] Create deffered tx to add admcl3 identity")
	id, err = admcl.AddAdminToAdminDarc(adminDarc.GetBaseID(), admcl3.AuthKey().Identity())
	require.NoError(t, err)
	admcl.bcl.WaitPropagation(1)
	_, err = admcl2.bcl.GetDeferredData(id)
	require.NoError(t, err)
	log.Lvl1("[INFO] Admin 1 sign the transaction")
	err = admcl.AddSignatureToDefferedTx(id)
	require.NoError(t, err)
	log.Lvl1("[INFO] Admin 2 sign the transaction")
	err = admcl2.AddSignatureToDefferedTx(id)
	require.NoError(t, err)
	log.Lvl1("[INFO] Admin 2 try to exec the transaction")
	err = admcl.ExecDefferedTx(id)
	require.NoError(t, err)
	log.Lvl1("[INFO] Admin 2 try to exec the transaction")
	_, err = lib.GetDarcByID(admcl.bcl, adminDarc.GetID())
	require.NoError(t, err)
	log.Lvl1("[INFO] Tx successfully executed, admin 3 has been added to the admin darc")

	// ------------------------------------------------------------------------
	// 2. Remove admin 2 from the admin darc
	// ------------------------------------------------------------------------

	log.Lvl1("[INFO] Create deffered tx to remove admcl2 identity")
	id, err = admcl.RemoveAdminFromAdminDarc(adminDarc.GetBaseID(), admcl2.AuthKey().Identity())
	require.NoError(t, err)
	admcl.bcl.WaitPropagation(1)
	_, err = admcl2.bcl.GetDeferredData(id)
	require.NoError(t, err)
	log.Lvl1("[INFO] Admin 1 sign the transaction")
	err = admcl.AddSignatureToDefferedTx(id)
	require.NoError(t, err)
	log.Lvl1("[INFO] Admin 3 sign the transaction")
	err = admcl3.AddSignatureToDefferedTx(id)
	require.NoError(t, err)
	log.Lvl1("[INFO] Admin 2 sign the transaction")
	err = admcl2.AddSignatureToDefferedTx(id)
	require.NoError(t, err)
	log.Lvl1("[INFO] Admin 3 try to execute the transaction")
	err = admcl3.ExecDefferedTx(id)
	require.NoError(t, err)
	log.Lvl1("[INFO] Tx successfully executed, admin 2 has been removed from the admin darc")
	_, err = lib.GetDarcByID(admcl.bcl, adminDarc.GetID())
	require.NoError(t, err)
}

func TestUpdateAdminKeys(t *testing.T) {
	// ------------------------------------------------------------------------
	// 0. Set up
	// ------------------------------------------------------------------------
	log.Lvl1("[INFO] Create admin darc")
	local := onet.NewTCPTest(cothority.Suite)
	defer local.CloseAll()

	superAdmin := darc.NewSignerEd25519(nil, nil)
	_, roster, _ := local.GenTree(5, true)

	genesisMsg, err := byzcoin.DefaultGenesisMsg(byzcoin.CurrentVersion, roster,
		[]string{"spawn:value", "spawn:deferred", "invoke:deferred.addProof",
			"invoke:deferred.execProposedTx"}, superAdmin.Identity())
	require.NoError(t, err)
	gDarc := &genesisMsg.GenesisDarc

	genesisMsg.BlockInterval = time.Second / 5
	bcl, _, err := byzcoin.NewLedger(genesisMsg, false)
	require.NoError(t, err)
	log.Lvl1("[INFO] Create admin client")
	admcl, err := NewClientWithAuth(bcl, &superAdmin)
	require.NoError(t, err)
	require.Equal(t, superAdmin, admcl.AuthKey())

	// ------------------------------------------------------------------------
	// 1. Spawn admin darc
	// ------------------------------------------------------------------------

	log.Lvl1("[INFO] Spawn admin darc")
	adminDarc, err := admcl.SpawnNewAdminDarc()
	require.NoError(t, err)
	admcl.bcl.WaitPropagation(1)
	_, err = lib.GetDarcByID(admcl.bcl, gDarc.GetID()) // Verify that the darc is really in the global state
	require.NoError(t, err)

	// ------------------------------------------------------------------------
	// 2. Add a new admin to the admin darc
	// ------------------------------------------------------------------------

	log.Lvl1("[INFO] Create new admin client (admcl2)")
	admcl2, err := NewClient(bcl)
	require.NoError(t, err)
	log.Lvl1("[INFO] Create deffered tx to add admcl2 identity")
	id, err := admcl.AddAdminToAdminDarc(adminDarc.GetBaseID(), admcl2.AuthKey().Identity())
	require.NoError(t, err)
	admcl.bcl.WaitPropagation(1)
	_, err = admcl2.bcl.GetDeferredData(id)
	require.NoError(t, err)
	log.Lvl1("[INFO] Admin 1 sign the transaction")
	admcl.AddSignatureToDefferedTx(id)
	log.Lvl1("[INFO] Admin 1 try to exec the transaction")
	err = admcl.ExecDefferedTx(id)
	require.NoError(t, err)
	log.Lvl1("[INFO] Tx successfully executed, admin 3 has been added to the admin darc")
	_, err = lib.GetDarcByID(admcl.bcl, adminDarc.GetID()) // Verify that the darc is really in the global state
	require.NoError(t, err)

	// ------------------------------------------------------------------------
	// 2. Add a new admin to the admin darc
	// ------------------------------------------------------------------------

	log.Lvl1("[INFO] Create new admin client (admcl3)")
	newAdmcl2, err := NewClient(bcl)
	require.NoError(t, err)
	log.Lvl1("[INFO] Create deffered tx to add admcl3 identity")
	id, err = admcl.ModifyAdminKeysFromAdminDarc(adminDarc.GetBaseID(), admcl2.AuthKey().Identity(), newAdmcl2.AuthKey().Identity())
	require.NoError(t, err)
	admcl.bcl.WaitPropagation(1)
	_, err = admcl2.bcl.GetDeferredData(id)
	require.NoError(t, err)
	log.Lvl1("[INFO] Admin 1 sign the transaction")
	err = admcl.AddSignatureToDefferedTx(id)
	require.NoError(t, err)
	log.Lvl1("[INFO] Admin 2 sign the transaction")
	err = admcl2.AddSignatureToDefferedTx(id)
	require.NoError(t, err)
	log.Lvl1("[INFO] Admin 2 try to exec the transaction")
	err = admcl.ExecDefferedTx(id)
	require.NoError(t, err)
	log.Lvl1("[INFO] Admin 2 try to exec the transaction")
	_, err = lib.GetDarcByID(admcl.bcl, adminDarc.GetID())
	require.NoError(t, err)
	log.Lvl1("[INFO] Tx successfully executed, admin 3 has been added to the admin darc")
}

// func TestProjectDarc(t *testing.T) {
// 	// ------------------------------------------------------------------------
// 	// 0. Set up
// 	// ------------------------------------------------------------------------
// 	log.Lvl1("[INFO] Create admin darc")
// 	local := onet.NewTCPTest(cothority.Suite)
// 	defer local.CloseAll()

// 	superAdmin := darc.NewSignerEd25519(nil, nil)
// 	_, roster, _ := local.GenTree(5, true)

// 	genesisMsg, err := byzcoin.DefaultGenesisMsg(byzcoin.CurrentVersion, roster,
// 		[]string{}, superAdmin.Identity())
// 	require.NoError(t, err)
// 	gDarc := &genesisMsg.GenesisDarc

// 	genesisMsg.BlockInterval = time.Second / 5
// 	bcl, _, err := byzcoin.NewLedger(genesisMsg, false)
// 	require.NoError(t, err)
// 	spawnNamingTx, err := bcl.CreateTransaction(byzcoin.Instruction{
// 		InstanceID: byzcoin.NewInstanceID(gDarc.GetBaseID()),
// 		Spawn: &byzcoin.Spawn{
// 			ContractID: byzcoin.ContractNamingID,
// 		},
// 		SignerCounter: []uint64{1},
// 	})
// 	require.NoError(t, spawnNamingTx.FillSignersAndSignWith(superAdmin))
// 	_, err = bcl.AddTransactionAndWait(spawnNamingTx, 10)
// 	require.NoError(t, err)
// 	log.Lvl1("[INFO] Create admin client")

// 	admcl, err := NewClientWithAuth(bcl, &superAdmin)
// 	admcl.incrementSignerCounter() // TODO manage the creation of the genesis block (the naming contract should be created in the genesis block like above)
// 	require.NoError(t, err)
// 	require.Equal(t, superAdmin, admcl.AuthKey())

// 	// ------------------------------------------------------------------------
// 	// 1. Spawn admin darc
// 	// ------------------------------------------------------------------------

// 	log.Lvl1("[INFO] Spawn admin darc")
// 	adminDarc, err := admcl.SpawnNewAdminDarc()
// 	require.NoError(t, err)
// 	admcl.bcl.WaitPropagation(1)
// 	_, err = lib.GetDarcByID(admcl.bcl, gDarc.GetID())
// 	require.NoError(t, err)

// 	// ------------------------------------------------------------------------
// 	// 2. Create a new project named project A
// 	// ------------------------------------------------------------------------

// 	log.Lvl1("[INFO] Spawn a new project darc")
// 	pdarcID, err := admcl.CreateNewProject(adminDarc.GetBaseID(), "Project A")
// 	require.NoError(t, err)
// 	_, err = lib.GetDarcByID(admcl.bcl, pdarcID)
// 	require.NoError(t, err)
// 	log.Lvl1("[INFO] Check if he access right value contract instance is set")
// 	arid, err := admcl.bcl.ResolveInstanceID(pdarcID, "AR") // check that the access right value contract is correctly named
// 	require.NoError(t, err)
// 	log.Lvl1("[INFO] The access right value contract is set")

// 	log.Lvl1("[INFO] Get the value of the access right contract")
// 	pr, err := admcl.bcl.WaitProof(byzcoin.NewInstanceID(arid.Slice()), 2*genesisMsg.BlockInterval, nil)
// 	require.Nil(t, err)
// 	v0, _, _, err := pr.Get(arid.Slice())
// 	require.Nil(t, err)
// 	ar := AccessRight{}
// 	err = protobuf.Decode(v0, &ar)
// 	require.Nil(t, err)
// 	log.Lvl1("[INFO] Access rights for querier with id : q1:h1 should not be set")
// 	_, ok := ar.AccessRightsMap["q1:h1"]
// 	require.Equal(t, ok, false)
// 	log.Lvl1("[INFO] Setting the access rights for querier with id : q1:h1")
// 	err = admcl.AddQuerierToProject(pdarcID, "q1:h1", "count_per_site")
// 	require.Nil(t, err)
// 	pr, err = admcl.bcl.WaitProof(byzcoin.NewInstanceID(arid.Slice()), 2*genesisMsg.BlockInterval, nil)
// 	require.Nil(t, err)
// 	v1, _, _, err := pr.Get(arid.Slice())
// 	require.Nil(t, err)
// 	ar = AccessRight{}
// 	err = protobuf.Decode(v1, &ar)
// 	require.Nil(t, err)
// 	v, ok := ar.AccessRightsMap["q1:h1"]
// 	require.Equal(t, ok, true)
// 	require.Equal(t, v, "count_per_site")
// 	log.Lvl1("[INFO] Access rights for querier with id : q1:h1 are set as expected")
// 	log.Lvl1("[INFO] Remove the access rights for querier with id : q1:h1")
// 	err = admcl.RemoveQuerierFromProject(pdarcID, "q1:h1")
// 	require.Nil(t, err)
// 	pr, err = admcl.bcl.WaitProof(byzcoin.NewInstanceID(arid.Slice()), 2*genesisMsg.BlockInterval, nil)
// 	require.Nil(t, err)
// 	v1, _, _, err = pr.Get(arid.Slice())
// 	require.Nil(t, err)
// 	ar = AccessRight{}
// 	err = protobuf.Decode(v1, &ar)
// 	require.Nil(t, err)
// 	v, ok = ar.AccessRightsMap["q1:h1"]
// 	require.Equal(t, ok, false)
// 	log.Lvl1("[INFO] Access rights for querier with id : q1:h1 have been removed")
// }
