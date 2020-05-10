package admin

import (
	"fmt"
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
	admcl.AddSignatureToDefferedTx(id, 0)
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
	err = admcl.AddSignatureToDefferedTx(id, 0)
	require.NoError(t, err)
	log.Lvl1("[INFO] Admin 1 try to exec the transaction but threshold of signature not reached")
	err = admcl.ExecDefferedTx(id)
	require.Error(t, err)
	log.Lvl1("[INFO] Admin 2 sign the transaction")
	err = admcl2.AddSignatureToDefferedTx(id, 0)
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
	err = admcl.AddSignatureToDefferedTx(id, 0)
	require.NoError(t, err)
	log.Lvl1("[INFO] Admin 3 sign the transaction")
	err = admcl3.AddSignatureToDefferedTx(id, 0)
	require.NoError(t, err)
	log.Lvl1("[INFO] Admin 2 sign the transaction")
	err = admcl2.AddSignatureToDefferedTx(id, 0)
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
	admcl.AddSignatureToDefferedTx(id, 0)
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
	err = admcl.AddSignatureToDefferedTx(id, 0)
	require.NoError(t, err)
	log.Lvl1("[INFO] Admin 2 sign the transaction")
	err = admcl2.AddSignatureToDefferedTx(id, 0)
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
	err = admcl.AddSignatureToDefferedTx(id, 0)
	require.NoError(t, err)
	log.Lvl1("[INFO] Admin 3 sign the transaction")
	err = admcl3.AddSignatureToDefferedTx(id, 0)
	require.NoError(t, err)
	log.Lvl1("[INFO] Admin 2 sign the transaction")
	err = admcl2.AddSignatureToDefferedTx(id, 0)
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
	admcl.AddSignatureToDefferedTx(id, 0)
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
	err = admcl.AddSignatureToDefferedTx(id, 0)
	require.NoError(t, err)
	log.Lvl1("[INFO] Admin 2 sign the transaction")
	err = admcl2.AddSignatureToDefferedTx(id, 0)
	require.NoError(t, err)
	log.Lvl1("[INFO] Admin 2 try to exec the transaction")
	err = admcl.ExecDefferedTx(id)
	require.NoError(t, err)
	log.Lvl1("[INFO] Admin 2 try to exec the transaction")
	_, err = lib.GetDarcByID(admcl.bcl, adminDarc.GetID())
	require.NoError(t, err)
	log.Lvl1("[INFO] Tx successfully executed, admin 3 has been added to the admin darc")
}

func TestProjectDarc(t *testing.T) {
	// log.SetDebugVisible(2)
	// ------------------------------------------------------------------------
	// 0. Set up
	// ------------------------------------------------------------------------
	log.Lvl1("[INFO] Create admin darc")
	local := onet.NewTCPTest(cothority.Suite)
	defer local.CloseAll()

	superAdmin := darc.NewSignerEd25519(nil, nil)
	_, roster, _ := local.GenTree(5, true)

	genesisMsg, err := byzcoin.DefaultGenesisMsg(byzcoin.CurrentVersion, roster,
		[]string{}, superAdmin.Identity())
	require.NoError(t, err)
	gDarc := &genesisMsg.GenesisDarc

	genesisMsg.BlockInterval = time.Second / 5
	bcl, _, err := byzcoin.NewLedger(genesisMsg, false)
	require.NoError(t, err)
	spawnNamingTx, err := bcl.CreateTransaction(byzcoin.Instruction{
		InstanceID: byzcoin.NewInstanceID(gDarc.GetBaseID()),
		Spawn: &byzcoin.Spawn{
			ContractID: byzcoin.ContractNamingID,
		},
		SignerCounter: []uint64{1},
	})
	require.NoError(t, spawnNamingTx.FillSignersAndSignWith(superAdmin))
	_, err = bcl.AddTransactionAndWait(spawnNamingTx, 10)
	require.NoError(t, err)
	log.Lvl1("[INFO] Create admin client")

	admcl, err := NewClientWithAuth(bcl, &superAdmin)
	admcl.incrementSignerCounter() // TODO manage the creation of the genesis block (the naming contract should be created in the genesis block like above)
	require.NoError(t, err)
	require.Equal(t, superAdmin, admcl.AuthKey())

	// ------------------------------------------------------------------------
	// 1. Spawn admin darc
	// ------------------------------------------------------------------------

	log.Lvl1("[INFO] Spawn admin darc")
	adminDarc, err := admcl.SpawnNewAdminDarc()
	require.NoError(t, err)
	admcl.bcl.WaitPropagation(1)
	_, err = lib.GetDarcByID(admcl.bcl, gDarc.GetID())
	require.NoError(t, err)

	log.Lvl1("[INFO] Create new admin client (admcl2)")
	admcl2, err := NewClient(bcl)
	require.NoError(t, err)
	log.Lvl1("[INFO] Create deffered tx to add admcl2 identity")
	id, err := admcl.AddAdminToAdminDarc(adminDarc.GetBaseID(), admcl2.AuthKey().Identity())
	require.NoError(t, err)
	err = admcl.AddSignatureToDefferedTx(id, 0)
	require.NoError(t, err)
	err = admcl.ExecDefferedTx(id)
	require.NoError(t, err)

	err = local.WaitDone(genesisMsg.BlockInterval)
	require.Nil(t, err)

	adminDarc, err = lib.GetDarcByID(admcl.bcl, adminDarc.GetID())
	require.NoError(t, err)
	fmt.Println(adminDarc)

	// ------------------------------------------------------------------------
	// 2. Create a new project named project A
	// ------------------------------------------------------------------------

	log.Lvl1("[INFO] Spawn a new project darc")
	instID, pdarcID, err := admcl.CreateNewProject(adminDarc.GetBaseID(), "Project A")
	admcl.AddSignatureToDefferedTx(instID, 0)
	require.NoError(t, err)
	admcl2.AddSignatureToDefferedTx(instID, 0)
	require.NoError(t, err)
	admcl.ExecDefferedTx(instID)
	err = local.WaitDone(genesisMsg.BlockInterval)
	require.Nil(t, err)
	pDarc, err := lib.GetDarcByID(admcl.bcl, pdarcID)
	require.NoError(t, err)
	fmt.Println(pDarc)
	log.Lvl1("[INFO] Check if he access right value contract instance is set")
	id, _, err = admcl.CreateAccessRight(adminDarc.GetID(), pdarcID)
	require.NoError(t, err)
	admcl.AddSignatureToDefferedTx(id, 0)
	require.NoError(t, err)
	admcl2.AddSignatureToDefferedTx(id, 0)
	require.NoError(t, err)
	err = admcl.ExecDefferedTx(id)
	require.NoError(t, err)

	// err = admcl.AttachAccessRightToProject(arID)
	// require.NoError(t, err)
	// id, err = admcl.bcl.ResolveInstanceID(pdarcID, "AR") // check that the access right value contract is correctly named
	// require.Equal(t, id, arID)
	// log.Lvl1("[INFO] The access right value contract is set")

	// log.Lvl1("[INFO] Get the value of the access right contract")
	// pr, err := admcl.bcl.WaitProof(byzcoin.NewInstanceID(arid.Slice()), 2*genesisMsg.BlockInterval, nil)
	// require.Nil(t, err)
	// v0, _, _, err := pr.Get(arid.Slice())
	// require.Nil(t, err)
	// ar := AccessRight{}
	// err = protobuf.Decode(v0, &ar)
	// require.Nil(t, err)
	// log.Lvl1("[INFO] Add the querier 1:1 to the project")
	// err = admcl.AddQuerierToProject(pdarcID, "1:1", "count_per_site_shuffled,count_global")
	// require.NoError(t, err)
	// err = admcl.ModifyQuerierAccessRightsForProject(pdarcID, "1:1", "count_per_site_shuffled")
	// require.NoError(t, err)
	// err = local.WaitDone(genesisMsg.BlockInterval)
	// require.Nil(t, err)
	// pr2, err := admcl.bcl.GetProof(arid.Slice())
	// v0, _, _, err = pr2.Proof.Get(arid.Slice())
	// require.Nil(t, err)
	// ar = AccessRight{}
	// err = protobuf.Decode(v0, &ar)
	// // fmt.Println(ar)
	// log.Lvl1("[INFO] Verify access right of querier 1:1 for action : count_per_site_shuffled")
	// authorization, err := admcl.VerifyAccessRights("1:1", "count_per_site_shuffled", pdarcID)
	// require.NoError(t, err)
	// require.True(t, authorization)
	// log.Lvl1("[INFO] Verify access right of querier 2:1 (not registered) for action :count_per_site_shuffled (Expected to fail)")
	// authorization, err = admcl.VerifyAccessRights("2:1", "count_per_site_shuffled", pdarcID)
	// require.Error(t, err)
	// require.False(t, authorization)
	// err = admcl.AddQuerierToProject(pdarcID, "1:4", "count_global")
	// require.NoError(t, err)
	// err = admcl.AddQuerierToProject(pdarcID, "1:3", "count_per_site_shuffled,patient_list")
	// require.NoError(t, err)
	// err = admcl.RemoveQuerierFromProject(pdarcID, "1:3")
	// require.NoError(t, err)
	// err = local.WaitDone(genesisMsg.BlockInterval)
	// require.Nil(t, err)

	// authorization, err = admcl.VerifyAccessRights("1:4", "count_global", pdarcID)
	// require.NoError(t, err)
	// require.True(t, authorization)
	// authorization, err = admcl.VerifyAccessRights("1:3", "patient_list", pdarcID)
	// require.Error(t, err)
	// require.False(t, authorization)
}
