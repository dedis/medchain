package admin

import (
	"encoding/binary"
	"errors"

	"github.com/medchain/admin/service"
	"go.dedis.ch/cothority/v3"
	"go.dedis.ch/cothority/v3/byzcoin"
	"go.dedis.ch/cothority/v3/byzcoin/bcadmin/lib"
	"go.dedis.ch/cothority/v3/darc"
	"go.dedis.ch/cothority/v3/darc/expression"
	"go.dedis.ch/onet/v3"
	"go.dedis.ch/protobuf"
	"golang.org/x/xerrors"
)

type Client struct {
	Bcl               *byzcoin.Client
	onetCl            *onet.Client // Onet Client to the ShareID service
	adminkeys         darc.Signer
	genDarc           darc.Darc
	signerCounter     uint64
	pendingDeferredTx map[byzcoin.InstanceID]bool // Set of deferred transactions IDs
}

// Map an action in the admin darc to the signature requirements.
// 1 : One admin can sign a transaction to perform the action
// 0 : The multisignature rule defined is necessary for the action in the transaction to be executed
var adminActions = map[darc.Action]uint{
	"invoke:darc.evolve":             0,
	"spawn:deferred":                 1,
	"invoke:deferred.addProof":       1,
	"invoke:deferred.execProposedTx": 1,
	"spawn:darc":                     0,
	"spawn:value":                    0,
	"invoke:value.update":            0,
	"_name:value":                    0,
}

// Instantiate a new admin client to perform administrators actions. This method create a new identity.
func NewClient(bcl *byzcoin.Client) (*Client, error) {
	if bcl == nil {
		return nil, errors.New("A Byzcoin Client is required")
	}
	cl := &Client{
		Bcl:               bcl,
		adminkeys:         darc.NewSignerEd25519(nil, nil),
		signerCounter:     1,
		onetCl:            onet.NewClient(cothority.Suite, "ShareID"),
		pendingDeferredTx: make(map[byzcoin.InstanceID]bool),
	}
	if genDarc, err := bcl.GetGenDarc(); err == nil {
		cl.genDarc = *genDarc
		return cl, nil
	} else {
		return nil, xerrors.Errorf("getting genesis darc from chain: %w", err)
	}
}

// Instantiate a new admin client to perform administrators actions. This method take an existing identity as argument.
func NewClientWithAuth(bcl *byzcoin.Client, keys *darc.Signer) (*Client, error) {
	if keys == nil {
		return nil, errors.New("Keys are required")
	}
	cl, err := NewClient(bcl)
	cl.adminkeys = *keys
	return cl, err
}

func (cl *Client) GetKeys() darc.Signer {
	return cl.adminkeys
}

// Spawn a new admin darc. This method create the admin darc and the transaction to spawn it.
func (cl *Client) SpawnNewAdminDarc() (*darc.Darc, error) {
	adminDarc, err := cl.createAdminDarc()
	if err != nil {
		return nil, xerrors.Errorf("Creating the admin darc: %w", err)
	}
	buf, err := adminDarc.ToProto()
	if err != nil {
		return nil, xerrors.Errorf("Marshalling: %w", err)
	}
	ctx, err := cl.Bcl.CreateTransaction(byzcoin.Instruction{
		InstanceID: byzcoin.NewInstanceID(cl.genDarc.GetBaseID()),
		Spawn: &byzcoin.Spawn{
			ContractID: byzcoin.ContractDarcID,
			Args: byzcoin.Arguments{
				{
					Name:  "darc",
					Value: buf,
				},
			},
		},
		SignerCounter: []uint64{cl.signerCounter},
	})
	if err != nil {
		return nil, xerrors.Errorf("Creating the transaction: %w", err)
	}
	err = cl.spawnTransaction(ctx)
	if err != nil {
		return nil, xerrors.Errorf("Adding transaction to the ledger: %w", err)
	}
	return adminDarc, err

}

// Create the admin list. The admin list is a value contract that hold the list of all registered admins in the admin darc.
// All operations that modify the administrators identities registered in the admin darc also modify this list.
// func (cl *Client) SpawnAdminsList(adid darc.ID) (byzcoin.InstanceID, error) {
// 	list := AdminsList{[]string{cl.GetKeys().Identity().String()}}
// 	buf, err := protobuf.Encode(&list)
// 	if err != nil {
// 		return byzcoin.InstanceID{}, xerrors.Errorf("Marshalling: %w", err)
// 	}
// 	ctx, err := cl.Bcl.CreateTransaction(byzcoin.Instruction{
// 		InstanceID: byzcoin.NewInstanceID(adid),
// 		Spawn: &byzcoin.Spawn{
// 			ContractID: contracts.ContractValueID,
// 			Args: byzcoin.Arguments{
// 				{
// 					Name:  "value",
// 					Value: buf,
// 				},
// 			},
// 		},
// 		SignerCounter: []uint64{cl.signerCounter},
// 	})
// 	if err != nil {
// 		return byzcoin.InstanceID{}, xerrors.Errorf("Creating the transaction: %w", err)
// 	}
// 	err = cl.spawnTransaction(ctx)
// 	if err != nil {
// 		return byzcoin.InstanceID{}, xerrors.Errorf("Adding transaction to the ledger: %w", err)
// 	}
// 	return ctx.Instructions[0].DeriveID(""), nil

// }

// Add a name resolver to the admin list (to adminsList) from the admin darc.
// To work, it is required that the naming contract is spawned (when setting up the Byzcoin chain).
func (cl *Client) AttachAdminsList(id byzcoin.InstanceID) error {
	ctx, err := cl.Bcl.CreateTransaction(byzcoin.Instruction{
		InstanceID: byzcoin.NamingInstanceID,
		Invoke: &byzcoin.Invoke{
			ContractID: byzcoin.ContractNamingID,
			Command:    "add",
			Args: byzcoin.Arguments{
				{
					Name:  "instanceID",
					Value: id.Slice(),
				},
				{
					Name:  "name",
					Value: []byte("adminsList"),
				},
			},
		},
		SignerCounter: []uint64{cl.signerCounter},
	})
	if err != nil {
		return xerrors.Errorf("Creating the transaction: %w", err)
	}
	return cl.spawnTransaction(ctx)
}

// Append a new admin to the admin list. It get the value of the contract using the name resolution
// to adminsList. It append the new admin identity, and create the instruction to update the value contract.
// func (cl *Client) addAdminToList(adid darc.ID, id string) (byzcoin.Instruction, []string, error) {
// 	listId, err := cl.Bcl.ResolveInstanceID(adid, "adminsList")
// 	if err != nil {
// 		return byzcoin.Instruction{}, []string{}, xerrors.Errorf("Resolving the instance id of value contract instance: %w", err)
// 	}
// 	list, err := cl.GetAdminsList(listId)
// 	if err != nil {
// 		return byzcoin.Instruction{}, []string{}, xerrors.Errorf("Getting admins list: %w", err)
// 	}
// 	err = Add(&list.List, id)
// 	if err != nil {
// 		return byzcoin.Instruction{}, []string{}, xerrors.Errorf("Adding admin to admins list: %w", err)
// 	}
// 	buf, err := protobuf.Encode(&list)
// 	if err != nil {
// 		return byzcoin.Instruction{}, []string{}, xerrors.Errorf("Marshalling: %w", err)
// 	}
// 	inst := byzcoin.Instruction{
// 		InstanceID: listId,
// 		Invoke: &byzcoin.Invoke{
// 			ContractID: contracts.ContractValueID,
// 			Command:    "update",
// 			Args: byzcoin.Arguments{
// 				{
// 					Name:  "value",
// 					Value: buf,
// 				},
// 			},
// 		},
// 	}
// 	return inst, list.List, nil

// }

// Remove an admin from the admin list. It get the value of the contract using the name resolution
// to adminsList. It removes the admin identity, and create the instruction to update the value contract.
// func (cl *Client) removeAdminFromList(adid darc.ID, id string) (byzcoin.Instruction, []string, error) {
// 	listId, err := cl.Bcl.ResolveInstanceID(adid, "adminsList")
// 	if err != nil {
// 		return byzcoin.Instruction{}, []string{}, xerrors.Errorf("Resolving the instance id of value contract instance: %w", err)
// 	}
// 	list, err := cl.GetAdminsList(listId)
// 	if err != nil {
// 		return byzcoin.Instruction{}, []string{}, xerrors.Errorf("Getting admins list: %w", err)
// 	}
// 	err = Remove(&list.List, id)
// 	if err != nil {
// 		return byzcoin.Instruction{}, []string{}, xerrors.Errorf("Removing admin from admins list: %w", err)
// 	}
// 	buf, err := protobuf.Encode(&list)
// 	if err != nil {
// 		return byzcoin.Instruction{}, []string{}, xerrors.Errorf("Marshalling: %w", err)
// 	}
// 	inst := byzcoin.Instruction{
// 		InstanceID: listId,
// 		Invoke: &byzcoin.Invoke{
// 			ContractID: contracts.ContractValueID,
// 			Command:    "update",
// 			Args: byzcoin.Arguments{
// 				{
// 					Name:  "value",
// 					Value: buf,
// 				},
// 			},
// 		},
// 	}
// 	return inst, list.List, nil

// }

// Update the identity of an admin in the list. It get the value of the contract using the name resolution
// to adminsList. It modify the admin identity, and create the instruction to update the value contract.
// func (cl *Client) updateAdminKeyInList(adid darc.ID, oldId, newId string) (byzcoin.Instruction, []string, error) {
// 	adminListInstanceId, err := cl.Bcl.ResolveInstanceID(adid, "adminsList")
// 	if err != nil {
// 		return byzcoin.Instruction{}, []string{}, xerrors.Errorf("Resolving the instance id of value contract instance: %w", err)
// 	}
// 	adminsList, err := cl.GetAdminsList(adminListInstanceId)
// 	if err != nil {
// 		return byzcoin.Instruction{}, []string{}, xerrors.Errorf("Getting admins adminsList: %w", err)
// 	}
// 	err = Update(&adminsList.List, oldId, newId)
// 	if err != nil {
// 		return byzcoin.Instruction{}, []string{}, xerrors.Errorf("Removing admin from admins adminsList: %w", err)
// 	}
// 	buf, err := protobuf.Encode(&adminsList)
// 	if err != nil {
// 		return byzcoin.Instruction{}, []string{}, xerrors.Errorf("Marshalling: %w", err)
// 	}
// 	inst := byzcoin.Instruction{
// 		InstanceID: adminListInstanceId,
// 		Invoke: &byzcoin.Invoke{
// 			ContractID: contracts.ContractValueID,
// 			Command:    "update",
// 			Args: byzcoin.Arguments{
// 				{
// 					Name:  "value",
// 					Value: buf,
// 				},
// 			},
// 		},
// 	}
// 	return inst, adminsList.List, nil

// }

// func (cl *Client) GetAdminsList(listId byzcoin.InstanceID) (AdminsList, error) {
// 	pr, err := cl.Bcl.GetProof(listId.Slice())
// 	if err != nil {
// 		return AdminsList{}, xerrors.Errorf("Resolving the proof of the value contract instance: %w", err)
// 	}
// 	buf, _, _, err := pr.Proof.Get(listId.Slice())
// 	if err != nil {
// 		return AdminsList{}, xerrors.Errorf("Getting the proof of value contract: %w", err)
// 	}
// 	list := AdminsList{}
// 	err = protobuf.Decode(buf, &list)
// 	if err != nil {
// 		return AdminsList{}, xerrors.Errorf("Encoding: %w", err)
// 	}
// 	return list, nil
// }

// Create the multisignature rule expression from the list of identities.
func createMultisigRuleExpression(identities []string, min int) expression.Expr {
	// this method from the bcadmin library return all the different combinations of identities for the given threshold
	// of required signatures
	andGroups := lib.CombinationAnds(identities, min)
	groupExpr := expression.InitOrExpr(andGroups...)
	return groupExpr
}

func (cl *Client) createAdminDarc() (*darc.Darc, error) {
	description := "Admin darc guards medchain project darcs"
	rules := darc.InitRules([]darc.Identity{cl.adminkeys.Identity()}, []darc.Identity{cl.adminkeys.Identity()})
	adminDarc := darc.NewDarc(rules, []byte(description))
	adminDarcActions := "invoke:darc.evolve,spawn:deferred,invoke:deferred.addProof,invoke:deferred.execProposedTx,spawn:darc,spawn:value,_name:value,invoke:value.update"
	identities := []string{cl.adminkeys.Identity().String()}
	adminDarcExpr := createMultisigRuleExpression(identities, len(identities)) // All identities are for now required to sign (len(identities))
	err := addActionsToDarc(adminDarc, adminDarcActions, adminDarcExpr)
	return adminDarc, err
}

// Create and spawn a deferred transaction with the transaction given as argument. Share the id of the spawned deferred
// transaction to all cothority nodes
func (cl *Client) addDeferredTransaction(tx byzcoin.ClientTransaction, adid darc.ID) (byzcoin.InstanceID, error) {
	txBuf, err := protobuf.Encode(&tx)
	if err != nil {
		return *new(byzcoin.InstanceID), xerrors.Errorf("Marshalling the transaction: %w", err)
	}
	deferredInstanceID, err := cl.spawnDeferredInstance(txBuf, adid)
	if err != nil {
		return *new(byzcoin.InstanceID), xerrors.Errorf("Creating the deferred transaction: %w", err)
	}
	res := service.ShareDeferredIDReply{}
	// Send the instance id of the deferred transaction to cothority nodes
	err = cl.onetCl.SendProtobuf(cl.Bcl.Roster.RandomServerIdentity(), &service.ShareDeferredID{deferredInstanceID, cl.Bcl.Roster}, &res)
	if err != nil {
		return *new(byzcoin.InstanceID), xerrors.Errorf("Sharing the id of the deferred transaction instance: %w", err)
	}
	return deferredInstanceID, nil
}

// Spawn a deferred transaction to add a new admin to admin darc and admins list.
// func (cl *Client) AddAdminToAdminDarc(adid darc.ID, newAdmin string) (byzcoin.InstanceID, error) {
// 	adminDarc, err := lib.GetDarcByID(cl.Bcl, adid)
// 	if err != nil {
// 		return *new(byzcoin.InstanceID), xerrors.Errorf("Getting the admin darc from chain: %w", err)
// 	}

// 	adminsListInstruction, adminsList, err := cl.addAdminToList(adid, newAdmin)
// 	if err != nil {
// 		return byzcoin.InstanceID{}, xerrors.Errorf("Adding admin to list: %w", err)
// 	}
// 	adminDarcInstruction, err := cl.evolveAdminDarc(adminsList, adminDarc)
// 	if err != nil {
// 		return *new(byzcoin.InstanceID), xerrors.Errorf("Evolving the admin darc: %w", err)
// 	}

// 	proposedTransaction, err := cl.Bcl.CreateTransaction(adminsListInstruction, adminDarcInstruction)
// 	if err != nil {
// 		return byzcoin.InstanceID{}, xerrors.Errorf("Creating the transaction: %w", err)
// 	}

// 	return cl.addDeferredTransaction(proposedTransaction, adid)
// }

// Spawn a deferred transaction to remove an admin from the admin darc and admins list.
// func (cl *Client) RemoveAdminFromAdminDarc(adid darc.ID, adminId string) (byzcoin.InstanceID, error) {
// 	adminDarc, err := lib.GetDarcByID(cl.Bcl, adid)
// 	if err != nil {
// 		return *new(byzcoin.InstanceID), xerrors.Errorf("Getting the admin darc from chain: %w", err)
// 	}

// 	adminsListInstruction, adminsList, err := cl.removeAdminFromList(adid, adminId)
// 	if err != nil {
// 		return byzcoin.InstanceID{}, xerrors.Errorf("Removing admin from list: %w", err)
// 	}

// 	adminDarcInstruction, err := cl.evolveAdminDarc(adminsList, adminDarc)
// 	if err != nil {
// 		return *new(byzcoin.InstanceID), xerrors.Errorf("Evolving the admin darc: %w", err)
// 	}

// 	proposedTransaction, err := cl.Bcl.CreateTransaction(adminsListInstruction, adminDarcInstruction)
// 	if err != nil {
// 		return byzcoin.InstanceID{}, xerrors.Errorf("Creating the transaction: %w", err)
// 	}
// 	return cl.addDeferredTransaction(proposedTransaction, adid)
// }

// Spawn a deferred transaction to remove an admin from the admin darc and admins list.
// func (cl *Client) ModifyAdminKeysFromAdminDarc(adid darc.ID, oldkey, newkey string) (byzcoin.InstanceID, error) {
// 	adminDarc, err := lib.GetDarcByID(cl.Bcl, adid)
// 	if err != nil {
// 		return *new(byzcoin.InstanceID), xerrors.Errorf("Getting the admin darc from chain: %w", err)
// 	}

// 	adminsListInstruction, adminsList, err := cl.updateAdminKeyInList(adid, oldkey, newkey)
// 	if err != nil {
// 		return byzcoin.InstanceID{}, xerrors.Errorf("Removing admin from list: %w", err)
// 	}

// 	adminDarcInstruction, err := cl.evolveAdminDarc(adminsList, adminDarc)
// 	if err != nil {
// 		return *new(byzcoin.InstanceID), xerrors.Errorf("Evolving the admin darc: %w", err)
// 	}

// 	proposedTransaction, err := cl.Bcl.CreateTransaction(adminsListInstruction, adminDarcInstruction)
// 	if err != nil {
// 		return byzcoin.InstanceID{}, xerrors.Errorf("Creating the transaction: %w", err)
// 	}
// 	return cl.addDeferredTransaction(proposedTransaction, adid)
// }

// Update all actions in the darc. The newAdminExpr slice holds the two kind of expression
// 0 : The multisignature rule
// 1 : Only one signature is required
func (cl *Client) updateAdminRules(evolvedAdminDarc *darc.Darc, newAdminExpr []expression.Expr) error {
	err := evolvedAdminDarc.Rules.UpdateEvolution(newAdminExpr[0])
	if err != nil {
		return xerrors.Errorf("Updating the _evolve expression in admin darc: %w", err)
	}
	err = evolvedAdminDarc.Rules.UpdateSign(newAdminExpr[0])
	if err != nil {
		return xerrors.Errorf("Updating the _sign expression in admin darc: %w", err)
	}
	// set the right expression for each action defined in the adminActions map
	for k, v := range adminActions {
		err = evolvedAdminDarc.Rules.UpdateRule(k, newAdminExpr[v])
		if err != nil {
			return xerrors.Errorf("Updating the %s expression in admin darc: %w", k, err)
		}
	}
	return nil
}

// Create the transaction that evolve the admin darc expressions with the new admins list
func (cl *Client) evolveAdminDarc(adminsList []string, olddarc *darc.Darc) (byzcoin.Instruction, error) {
	newdarc := olddarc.Copy()
	// All identities are for now required to sign (len(adminsList))
	newAdminExpr := []expression.Expr{createMultisigRuleExpression(adminsList, len(adminsList)), expression.InitOrExpr(adminsList...)}
	err := cl.updateAdminRules(newdarc, newAdminExpr)
	if err != nil {
		return byzcoin.Instruction{}, xerrors.Errorf("Updating admin rules: %w", err)
	}
	err = newdarc.EvolveFrom(olddarc)
	if err != nil {
		return byzcoin.Instruction{}, xerrors.Errorf("Evolving the admin darc: %w", err)
	}
	_, darc2Buf, err := newdarc.MakeEvolveRequest(cl.GetKeys())
	if err != nil {
		return byzcoin.Instruction{}, xerrors.Errorf("Creating the evolution request: %w", err)
	}

	proposedTransaction := byzcoin.Instruction{
		InstanceID: byzcoin.NewInstanceID(olddarc.GetBaseID()),
		Invoke: &byzcoin.Invoke{
			ContractID: byzcoin.ContractDarcID,
			Command:    "evolve",
			Args: []byzcoin.Argument{{
				Name:  "darc",
				Value: darc2Buf,
			}},
		},
	}
	return proposedTransaction, nil
}

// Take a marshalled proposed transaction as argument and spawn a deferred transaction.
func (cl *Client) spawnDeferredInstance(proposedTransactionBuf []byte, adid darc.ID) (byzcoin.InstanceID, error) {
	// The deferred transaction without ExpireBlockIndex given as argument is valid for 50 blocks
	ctx, err := cl.Bcl.CreateTransaction(byzcoin.Instruction{
		InstanceID: byzcoin.NewInstanceID(adid),
		Spawn: &byzcoin.Spawn{
			ContractID: byzcoin.ContractDeferredID,
			Args: []byzcoin.Argument{
				{
					Name:  "proposedTransaction",
					Value: proposedTransactionBuf,
				},
			},
		},
		SignerCounter: []uint64{cl.signerCounter},
	})
	if err != nil {
		return *new(byzcoin.InstanceID), xerrors.Errorf("Creating the deferred transaction: %w", err)
	}
	err = cl.spawnTransaction(ctx)
	if err != nil {
		return *new(byzcoin.InstanceID), xerrors.Errorf("Adding transaction to the ledger: %w", err)
	}
	return ctx.Instructions[0].DeriveID(""), err
}

func (cl *Client) SynchronizeDeferredInstanceIDs() error {
	deferredIDs, err := cl.FetchNewDeferredInstanceIDs()
	if err != nil {
		return xerrors.Errorf("Fetching Instance IDs: %w", err)
	}
	for _, id := range deferredIDs.Ids {
		cl.pendingDeferredTx[id] = false
	}
	return nil
}

// Get the last deferred transaction instance ids, known by the different conodes
func (cl *Client) FetchNewDeferredInstanceIDs() (service.GetDeferredIDsReply, error) {
	res := service.GetDeferredIDsReply{}
	err := cl.onetCl.SendProtobuf(cl.Bcl.Roster.RandomServerIdentity(), &service.GetDeferredIDs{}, &res)
	if err != nil {
		return service.GetDeferredIDsReply{}, xerrors.Errorf("Sending the GetDeferredIDs request to the service : %w", err)
	}
	return res, nil
}

func (cl *Client) AddSignatureToDeferredTx(instID byzcoin.InstanceID, instIdx uint64) error {
	if _, ok := cl.pendingDeferredTx[instID]; !ok {
		cl.pendingDeferredTx[instID] = false
	}
	deferredData, err := cl.Bcl.GetDeferredData(instID)
	if err != nil {
		return xerrors.Errorf("Getting the deferred instance from chain: %w", err)
	}
	rootHash := deferredData.InstructionHashes
	identity := cl.GetKeys().Identity()
	identityBuf, err := protobuf.Encode(&identity)
	if err != nil {
		return xerrors.Errorf("Encoding the identity of signer: %w", err)
	}
	signature, err := cl.GetKeys().Sign(rootHash[instIdx])
	if err != nil {
		return xerrors.Errorf("Signing the deferred transaction: %w", err)
	}
	index := uint32(instIdx) // The index of the instruction to sign in the transaction
	indexBuf := make([]byte, 4)
	binary.LittleEndian.PutUint32(indexBuf, index)

	ctx, err := cl.Bcl.CreateTransaction(byzcoin.Instruction{
		InstanceID: instID,
		Invoke: &byzcoin.Invoke{
			ContractID: byzcoin.ContractDeferredID,
			Command:    "addProof",
			Args: []byzcoin.Argument{
				{
					Name:  "identity",
					Value: identityBuf,
				},
				{
					Name:  "signature",
					Value: signature,
				},
				{
					Name:  "index",
					Value: indexBuf,
				},
			},
		},
		SignerCounter: []uint64{cl.signerCounter},
	})
	if err != nil {
		return xerrors.Errorf("Creating the transaction: %w", err)
	}
	cl.pendingDeferredTx[instID] = true
	return cl.spawnTransaction(ctx)
}

// Sign for the exectution of the deferred transaction
func (cl *Client) ExecDeferredTx(instID byzcoin.InstanceID) error {
	ctx, err := cl.Bcl.CreateTransaction(byzcoin.Instruction{
		InstanceID: instID,
		Invoke: &byzcoin.Invoke{
			ContractID: byzcoin.ContractDeferredID,
			Command:    "execProposedTx",
		},
		SignerCounter: []uint64{cl.signerCounter},
	})
	if err != nil {
		return xerrors.Errorf("Creating the transaction: %w", err)
	}
	return cl.spawnTransaction(ctx)
}

// Spawn a deferred transaction that hold the spawn transaction of a new project darc
func (cl *Client) CreateNewProject(adid darc.ID, pname string) (byzcoin.InstanceID, darc.ID, string, error) {
	proposedTransaction, pdarcID, pdarcIDString, err := cl.createProjectDarc(pname, adid)
	if err != nil {
		return byzcoin.InstanceID{}, darc.ID{}, "", xerrors.Errorf("Crafting the project darc spawn transaction: %w", err)
	}
	id, err := cl.addDeferredTransaction(proposedTransaction, adid)
	if err != nil {
		return byzcoin.InstanceID{}, darc.ID{}, "", xerrors.Errorf("Spawning the deferred transaction: %w", err)
	}
	return id, pdarcID, pdarcIDString, err
}

func (cl *Client) createProjectDarc(pname string, adid darc.ID) (byzcoin.ClientTransaction, darc.ID, string, error) {
	pdarcDescription := pname
	adminDarc, err := lib.GetDarcByID(cl.Bcl, adid)
	if err != nil {
		return byzcoin.ClientTransaction{}, darc.ID{}, "", xerrors.Errorf("Getting the admin darc from chain: %w", err)
	}
	// All rules that involves accessright are delegated to the admin darc (the signatures needs to satisfy the _sign
	// expression of the admin darc).
	// The other actions by default (_evolve, _sign) require only the admin that create the darc to sign.
	// This is not desirable in practice but evolving these rules in all project darc for every change in the admin darc is not easy to manage
	rules := darc.InitRules([]darc.Identity{cl.adminkeys.Identity()}, []darc.Identity{cl.adminkeys.Identity()})
	pdarc := darc.NewDarc(rules, []byte(pdarcDescription))
	pdarcActions := "spawn:accessright,invoke:accessright.add,invoke:accessright.update,invoke:accessright.delete"
	identities := []string{adminDarc.GetIdentityString()}
	// All identities are for now required to sign (len(identities))
	pdarcExpr := createMultisigRuleExpression(identities, len(identities))
	err = addActionsToDarc(pdarc, pdarcActions, pdarcExpr)
	if err != nil {
		return byzcoin.ClientTransaction{}, darc.ID{}, "", xerrors.Errorf("Adding rule to darc: %w", err)
	}

	err = pdarc.Rules.UpdateEvolution(pdarcExpr)
	if err != nil {
		return byzcoin.ClientTransaction{}, darc.ID{}, "", xerrors.Errorf("Updating the _evolve expression in darc: %w", err)
	}
	err = pdarc.Rules.UpdateSign(pdarcExpr)
	if err != nil {
		return byzcoin.ClientTransaction{}, darc.ID{}, "", xerrors.Errorf("Updating the _sign expression in darc: %w", err)
	}

	// The naming contract doesn't support deferred transactions, therefore the one spawning the project darc needs to
	// set up the name resolver
	pdarcActions = "_name:accessright"
	pdarcExpr = expression.InitOrExpr(cl.adminkeys.Identity().String())
	err = addActionsToDarc(pdarc, pdarcActions, pdarcExpr)
	if err != nil {
		return byzcoin.ClientTransaction{}, darc.ID{}, "", xerrors.Errorf("Adding rules to darc: %w", err)
	}
	pdarcProto, err := pdarc.ToProto()
	if err != nil {
		return byzcoin.ClientTransaction{}, darc.ID{}, "", xerrors.Errorf("Marshalling: %w", err)
	}
	ctx, err := cl.Bcl.CreateTransaction(byzcoin.Instruction{
		InstanceID: byzcoin.NewInstanceID(adid),
		Spawn: &byzcoin.Spawn{
			ContractID: byzcoin.ContractDarcID,
			Args: byzcoin.Arguments{
				{
					Name:  "darc",
					Value: pdarcProto,
				},
			},
		},
	})
	if err != nil {
		return byzcoin.ClientTransaction{}, darc.ID{}, "", xerrors.Errorf("Creating the transaction: %w", err)
	}
	return ctx, pdarc.GetBaseID(), pdarc.GetIdentityString(), nil
}

// func (cl *Client) CreateAccessRight(pdarc, adid darc.ID) (byzcoin.InstanceID, error) {
// 	emptyAccess := AccessRight{[]string{}, []string{}}
// 	buf, err := protobuf.Encode(&emptyAccess)
// 	if err != nil {
// 		return byzcoin.InstanceID{}, xerrors.Errorf("Encoding the access right struct: %w", err)
// 	}
// 	ctx, err := cl.Bcl.CreateTransaction(byzcoin.Instruction{
// 		InstanceID: byzcoin.NewInstanceID(pdarc),
// 		Spawn: &byzcoin.Spawn{
// 			ContractID: ContractAccessRightID,
// 			Args: []byzcoin.Argument{{
// 				Name:  "ar",
// 				Value: buf,
// 			}},
// 		},
// 	})
// 	if err != nil {
// 		return byzcoin.InstanceID{}, xerrors.Errorf("Creating the transaction: %w", err)
// 	}
// 	id, err := cl.addDeferredTransaction(ctx, adid)
// 	if err != nil {
// 		return byzcoin.InstanceID{}, xerrors.Errorf("Adding Deferred transaction to the ledger: %w", err)
// 	}
// 	return id, nil
// }

// Add a name resolver to the accessright contract instance (to AR) from the project darc.
// To work, it is required that the naming contract is spawned (when setting up the Byzcoin chain).
func (cl *Client) AttachAccessRightToProject(arID byzcoin.InstanceID) error {
	ctx, err := cl.Bcl.CreateTransaction(byzcoin.Instruction{
		InstanceID: byzcoin.NamingInstanceID,
		Invoke: &byzcoin.Invoke{
			ContractID: byzcoin.ContractNamingID,
			Command:    "add",
			Args: byzcoin.Arguments{
				{
					Name:  "instanceID",
					Value: arID.Slice(),
				},
				{
					Name:  "name",
					Value: []byte("AR"),
				},
			},
		},
		SignerCounter: []uint64{cl.signerCounter},
	})
	if err != nil {
		return xerrors.Errorf("Creating the transaction: %w", err)
	}
	return cl.spawnTransaction(ctx)
}

// Get the instance id of the contract deployed after the execution of a deferred transaction
func (cl *Client) GetContractInstanceID(id byzcoin.InstanceID, instIdx uint64) (byzcoin.InstanceID, error) {
	deferredData, err := cl.Bcl.GetDeferredData(id)
	if err != nil {
		return byzcoin.InstanceID{}, xerrors.Errorf("Getting the deferred data: %w", err)
	}
	finalId := deferredData.ExecResult[instIdx]
	return byzcoin.NewInstanceID(finalId), nil
}

func (cl *Client) GetLastSignerCounter() uint64 {
	signerCtrs, _ := cl.Bcl.GetSignerCounters(cl.GetKeys().Identity().String())
	return signerCtrs.Counters[0]
}

func (cl *Client) incrementSignerCounter() {
	cl.signerCounter++
}

func (cl *Client) updateSignerCounter(sc uint64) {
	cl.signerCounter = sc
}

func (cl *Client) SyncSignerCounter() {
	cl.signerCounter = cl.GetLastSignerCounter()
	cl.incrementSignerCounter()
}

// Get the access right from the accessright contract instance attached to the project darc given as argument.
// Performs the name resolution for name 'AR' and decode the data stored in the accessright contract instance
// func (cl *Client) GetAccessRightFromProjectDarcID(pdid darc.ID) (*AccessRight, byzcoin.InstanceID, error) {
// 	arid, err := cl.Bcl.ResolveInstanceID(pdid, "AR")
// 	if err != nil {
// 		return &AccessRight{}, byzcoin.InstanceID{}, xerrors.Errorf("Resolving the instance id of access right instance: %w", err)
// 	}
// 	pr, err := cl.Bcl.GetProof(arid.Slice())
// 	if err != nil {
// 		return &AccessRight{}, byzcoin.InstanceID{}, xerrors.Errorf("Resolving the proof of the access right instance: %w", err)
// 	}
// 	buf, _, _, err := pr.Proof.Get(arid.Slice())
// 	if err != nil {
// 		return &AccessRight{}, byzcoin.InstanceID{}, xerrors.Errorf("Getting the proof of access right: %w", err)
// 	}
// 	ar := AccessRight{}
// 	err = protobuf.Decode(buf, &ar)
// 	if err != nil {
// 		return &AccessRight{}, byzcoin.InstanceID{}, xerrors.Errorf("Encoding: %w", err)
// 	}
// 	return &ar, arid, nil
// }

// func (cl *Client) AddQuerierToProject(pdid, adid darc.ID, qid, access string) (byzcoin.InstanceID, error) {
// 	arid, err := cl.Bcl.ResolveInstanceID(pdid, "AR")
// 	if err != nil {
// 		return byzcoin.InstanceID{}, xerrors.Errorf("Reolving access right instance: %w", err)
// 	}
// 	ctx, err := cl.Bcl.CreateTransaction(byzcoin.Instruction{
// 		InstanceID: arid,
// 		Invoke: &byzcoin.Invoke{
// 			ContractID: ContractAccessRightID,
// 			Command:    "add",
// 			Args: []byzcoin.Argument{{
// 				Name:  "id",
// 				Value: []byte(qid),
// 			},
// 				{
// 					Name:  "ar",
// 					Value: []byte(access),
// 				}},
// 		},
// 	})
// 	if err != nil {
// 		return byzcoin.InstanceID{}, xerrors.Errorf("Creating the transaction: %w", err)
// 	}
// 	id, err := cl.addDeferredTransaction(ctx, adid)
// 	if err != nil {
// 		return byzcoin.InstanceID{}, xerrors.Errorf("Adding Deferred transaction to the ledger: %w", err)
// 	}
// 	return id, nil
// }

// func (cl *Client) RemoveQuerierFromProject(pdid, adid darc.ID, qid string) (byzcoin.InstanceID, error) {
// 	arid, err := cl.Bcl.ResolveInstanceID(pdid, "AR")
// 	if err != nil {
// 		return byzcoin.InstanceID{}, xerrors.Errorf("Reolving access right instance: %w", err)
// 	}
// 	ctx, err := cl.Bcl.CreateTransaction(byzcoin.Instruction{
// 		InstanceID: arid,
// 		Invoke: &byzcoin.Invoke{
// 			ContractID: ContractAccessRightID,
// 			Command:    "delete",
// 			Args: []byzcoin.Argument{{
// 				Name:  "id",
// 				Value: []byte(qid),
// 			},
// 			},
// 		},
// 	})
// 	if err != nil {
// 		return byzcoin.InstanceID{}, xerrors.Errorf("Creating the transaction: %w", err)
// 	}
// 	id, err := cl.addDeferredTransaction(ctx, adid)
// 	if err != nil {
// 		return byzcoin.InstanceID{}, xerrors.Errorf("Adding Deferred transaction to the ledger: %w", err)
// 	}
// 	return id, nil
// }

// func (cl *Client) ModifyQuerierAccessRightsForProject(pdid, adid darc.ID, qid, access string) (byzcoin.InstanceID, error) {
// 	arid, err := cl.Bcl.ResolveInstanceID(pdid, "AR")
// 	if err != nil {
// 		return byzcoin.InstanceID{}, xerrors.Errorf("Reolving access right instance: %w", err)
// 	}
// 	ctx, err := cl.Bcl.CreateTransaction(byzcoin.Instruction{
// 		InstanceID: arid,
// 		Invoke: &byzcoin.Invoke{
// 			ContractID: ContractAccessRightID,
// 			Command:    "update",
// 			Args: []byzcoin.Argument{{
// 				Name:  "id",
// 				Value: []byte(qid),
// 			},
// 				{
// 					Name:  "ar",
// 					Value: []byte(access),
// 				}},
// 		},
// 	})
// 	if err != nil {
// 		return byzcoin.InstanceID{}, xerrors.Errorf("Creating the transaction: %w", err)
// 	}
// 	id, err := cl.addDeferredTransaction(ctx, adid)
// 	if err != nil {
// 		return byzcoin.InstanceID{}, xerrors.Errorf("Adding Deferred transaction to the ledger: %w", err)
// 	}
// 	return id, nil
// }

// func (cl *Client) VerifyAccessRights(qid, access string, pdid darc.ID) (bool, error) {
// 	arid, err := cl.Bcl.ResolveInstanceID(pdid, "AR")
// 	if err != nil {
// 		return false, xerrors.Errorf("Reolving access right instance: %w", err)
// 	}
// 	pr, err := cl.Bcl.GetProof(arid.Slice())
// 	if err != nil {
// 		return false, xerrors.Errorf("Getting the proof of the accessright contract instance: %w", err)
// 	}
// 	buf, _, _, err := pr.Proof.Get(arid.Slice())
// 	if err != nil {
// 		return false, xerrors.Errorf("Getting the value: %w", err)
// 	}
// 	received := AccessRight{}
// 	err = protobuf.Decode(buf, &received)
// 	if err != nil {
// 		return false, xerrors.Errorf("Unmarshalling: %w", err)
// 	}
// 	idx, _ := Find(received.Ids, qid)
// 	if idx == -1 {
// 		return false, xerrors.Errorf("There is no such querier registered in the accessright contract")
// 	}
// 	// the different access rights for a user are assumed to be splitted with a ':'
// 	_, present := Find(strings.Split(received.Access[idx], ":"), access)
// 	return present, nil
// }

// func (cl *Client) ShowAccessRights(qid string, pdid darc.ID) (string, error) {
// 	arid, err := cl.Bcl.ResolveInstanceID(pdid, "AR")
// 	if err != nil {
// 		return "", xerrors.Errorf("Reolving access right instance: %w", err)
// 	}
// 	pr, err := cl.Bcl.GetProof(arid.Slice())
// 	if err != nil {
// 		return "", xerrors.Errorf("Getting the proof of the accessright contract instance: %w", err)
// 	}
// 	buf, _, _, err := pr.Proof.Get(arid.Slice())
// 	if err != nil {
// 		return "", xerrors.Errorf("Getting the value: %w", err)
// 	}
// 	received := AccessRight{}
// 	err = protobuf.Decode(buf, &received)
// 	if err != nil {
// 		return "", xerrors.Errorf("Unmarshalling: %w", err)
// 	}
// 	idx, _ := Find(received.Ids, qid)
// 	if idx == -1 {
// 		return "", xerrors.Errorf("There is no such querier registered in the accessright contract")
// 	}

// 	return received.Access[idx], nil
// }
