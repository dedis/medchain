package admin

import (
	"encoding/binary"
	"errors"
	"strings"

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
	scl               *onet.Client // Oner Client to the ShareID service
	adminkeys         darc.Signer
	genDarc           darc.Darc
	signerCounter     uint64
	slc               []string                    // List of known admin ids (used to create expressions)
	pendingDefferedTx map[byzcoin.InstanceID]bool // Set of deferred transactions IDs
}

var adminActions = map[darc.Action]uint{
	"invoke:darc.evolve":             0,
	"spawn:deferred":                 1,
	"invoke:deferred.addProof":       1,
	"invoke:deferred.execProposedTx": 1,
	"spawn:darc":                     0,
}

// func (cl *Client) DummyTest() error {
// 	// sh := ShareID{"id12134655"}
// 	err := cl.Bcl.SendProtobuf(cl.Bcl.Roster.RandomServerIdentity(), &sh, nil)
// 	return err
// }

func NewClient(bcl *byzcoin.Client) (*Client, error) {
	if bcl == nil {
		return nil, errors.New("A Byzcoin Client is required")
	}
	cl := &Client{
		Bcl:               bcl,
		adminkeys:         darc.NewSignerEd25519(nil, nil), // TODO add as optional arguments
		signerCounter:     1,
		scl:               onet.NewClient(cothority.Suite, "ShareID"),
		pendingDefferedTx: make(map[byzcoin.InstanceID]bool),
	}
	if genDarc, err := bcl.GetGenDarc(); err == nil {
		cl.genDarc = *genDarc
		return cl, nil
	} else {
		return nil, xerrors.Errorf("getting genesis darc from chain: %w", err)
	}
}

func NewClientWithAuth(bcl *byzcoin.Client, keys *darc.Signer) (*Client, error) {
	if keys == nil {
		return nil, errors.New("Keys are required")
	}
	cl, err := NewClient(bcl)
	cl.adminkeys = *keys
	return cl, err
}

// TODO for now admin different ids are saved locally
func (cl *Client) SyncDarc(slc []string) {
	cl.slc = slc
}
func (cl *Client) AuthKey() darc.Signer {
	return cl.adminkeys
}

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
		// SignerIdentities: []darc.Identity{superAdmin.Identity()},
		SignerCounter: []uint64{cl.signerCounter},
	})
	if err != nil {
		return nil, xerrors.Errorf("Creating the deffered transaction: %w", err)
	}
	err = cl.spawnTransaction(ctx)
	if err != nil {
		return nil, xerrors.Errorf("Adding transaction to the ledger: %w", err)
	}
	return adminDarc, err

}

// TODO will need to use the method to create threshold multisig rules when implemented
func createMultisigRuleExpression(al []string) expression.Expr {
	return expression.InitAndExpr(al...) // For now everyone needs to sign
}

func (cl *Client) createAdminDarc() (*darc.Darc, error) {
	description := "Admin darc guards medchain project darcs"
	rules := darc.InitRules([]darc.Identity{cl.adminkeys.Identity()}, []darc.Identity{cl.adminkeys.Identity()})
	adminDarc := darc.NewDarc(rules, []byte(description))
	adminDarcActions := "invoke:darc.evolve,spawn:deferred,invoke:deferred.addProof,invoke:deferred.execProposedTx,spawn:darc"
	adminDarcExpr := createMultisigRuleExpression([]string{cl.adminkeys.Identity().String()})
	err := AddRuleToDarc(adminDarc, adminDarcActions, adminDarcExpr)
	return adminDarc, err
}

func (cl *Client) addDeferredTransaction(tx byzcoin.ClientTransaction, adid darc.ID) (byzcoin.InstanceID, error) {
	txBuf, err := protobuf.Encode(&tx)
	if err != nil {
		return *new(byzcoin.InstanceID), xerrors.Errorf("Marshalling the transaction: %w", err)
	}
	ctxID, err := cl.spawnDeferredInstance(txBuf, adid)
	if err != nil {
		return *new(byzcoin.InstanceID), xerrors.Errorf("Creating the deffered transaction: %w", err)
	}
	res := DefferedIDReply{}
	err = cl.scl.SendProtobuf(cl.Bcl.Roster.RandomServerIdentity(), &DefferedID{ctxID, &cl.Bcl.Roster}, &res)
	if err != nil {
		return *new(byzcoin.InstanceID), xerrors.Errorf("Sharing the id of the deferred transaction instance: %w", err)
	}
	return ctxID, nil
}

func (cl *Client) AddAdminToAdminDarc(adid darc.ID, newAdmin string) (byzcoin.InstanceID, error) {
	adminDarc, err := lib.GetDarcByID(cl.Bcl, adid)
	if err != nil {
		return *new(byzcoin.InstanceID), xerrors.Errorf("Getting the admin darc from chain: %w", err)
	}
	exp := adminDarc.Rules.GetEvolutionExpr()
	slc := strings.Split(string(exp), "&")
	for i := range slc {
		slc[i] = strings.TrimSpace(slc[i])
	}
	slc = append(slc, newAdmin)
	cl.slc = slc
	proposedTransaction, err := cl.evolveAdminDarc(slc, adminDarc)
	if err != nil {
		return *new(byzcoin.InstanceID), xerrors.Errorf("Evolving the admin darc: %w", err)
	}

	return cl.addDeferredTransaction(proposedTransaction, adid)
}

func (cl *Client) RemoveAdminFromAdminDarc(adid darc.ID, adminId string) (byzcoin.InstanceID, error) {
	adminDarc, err := lib.GetDarcByID(cl.Bcl, adid)
	if err != nil {
		return *new(byzcoin.InstanceID), xerrors.Errorf("Getting the admin darc from chain: %w", err)
	}
	exp := adminDarc.Rules.GetEvolutionExpr()
	slc := strings.Split(string(exp), "&")
	for i := range slc {
		slc[i] = strings.TrimSpace(slc[i])
	}
	idx := IndexOf(adminId, slc)
	if idx == -1 {
		return *new(byzcoin.InstanceID), xerrors.Errorf("The adminID doesn't exist in the admin darc")
	}
	slc = append(slc[:idx], slc[idx+1:]...)
	cl.slc = slc
	proposedTransaction, err := cl.evolveAdminDarc(slc, adminDarc)
	if err != nil {
		return *new(byzcoin.InstanceID), xerrors.Errorf("Evolving the admin darc: %w", err)
	}
	return cl.addDeferredTransaction(proposedTransaction, adid)
}

func (cl *Client) ModifyAdminKeysFromAdminDarc(adid darc.ID, oldkey, newkey string) (byzcoin.InstanceID, error) {
	adminDarc, err := lib.GetDarcByID(cl.Bcl, adid)
	if err != nil {
		return *new(byzcoin.InstanceID), xerrors.Errorf("Getting the admin darc from chain: %w", err)
	}
	exp := adminDarc.Rules.GetEvolutionExpr()
	slc := strings.Split(string(exp), "&")
	for i := range slc {
		slc[i] = strings.TrimSpace(slc[i])
	}
	idx := IndexOf(oldkey, slc)
	if idx == -1 {
		return *new(byzcoin.InstanceID), xerrors.Errorf("The adminID doesn't exist in the admin darc")
	}
	slc[idx] = newkey
	proposedTransaction, err := cl.evolveAdminDarc(slc, adminDarc)
	if err != nil {
		return *new(byzcoin.InstanceID), xerrors.Errorf("Evolving the admin darc: %w", err)
	}
	return cl.addDeferredTransaction(proposedTransaction, adid)
}

func (cl *Client) updateAdminRules(evolvedAdminDarc *darc.Darc, newAdminExpr []expression.Expr) error {
	err := evolvedAdminDarc.Rules.UpdateEvolution(newAdminExpr[0])
	if err != nil {
		return xerrors.Errorf("Updating the _evolve expression in admin darc: %w", err)
	}
	err = evolvedAdminDarc.Rules.UpdateSign(newAdminExpr[0])
	if err != nil {
		return xerrors.Errorf("Updating the _sign expression in admin darc: %w", err)
	}

	for k, v := range adminActions {
		err = evolvedAdminDarc.Rules.UpdateRule(k, newAdminExpr[v])
		if err != nil {
			return xerrors.Errorf("Updating the %s expression in admin darc: %w", k, err)
		}
	}
	return nil
}

func (cl *Client) evolveAdminDarc(slc []string, olddarc *darc.Darc) (byzcoin.ClientTransaction, error) {
	newdarc := olddarc.Copy()
	newAdminExpr := []expression.Expr{createMultisigRuleExpression(slc), expression.InitOrExpr(slc...)}
	err := cl.updateAdminRules(newdarc, newAdminExpr)
	if err != nil {
		return byzcoin.ClientTransaction{}, xerrors.Errorf("Updating admin rules: %w", err)
	}
	err = newdarc.EvolveFrom(olddarc)
	if err != nil {
		return byzcoin.ClientTransaction{}, xerrors.Errorf("Evolving the admin darc: %w", err)
	}
	_, darc2Buf, err := newdarc.MakeEvolveRequest(cl.AuthKey())
	if err != nil {
		return byzcoin.ClientTransaction{}, xerrors.Errorf("Creating the evolution request: %w", err)
	}

	proposedTransaction, err := cl.Bcl.CreateTransaction(byzcoin.Instruction{
		InstanceID: byzcoin.NewInstanceID(olddarc.GetBaseID()),
		Invoke: &byzcoin.Invoke{
			ContractID: byzcoin.ContractDarcID,
			Command:    "evolve",
			Args: []byzcoin.Argument{{
				Name:  "darc",
				Value: darc2Buf,
			}},
		},
	})
	if err != nil {
		return byzcoin.ClientTransaction{}, xerrors.Errorf("Creating the transaction: %w", err)
	}
	return proposedTransaction, nil
}

func (cl *Client) spawnDeferredInstance(proposedTransactionBuf []byte, adid darc.ID) (byzcoin.InstanceID, error) {
	// TODO add as arguments
	expireBlockIndexInt := uint64(6000)
	expireBlockIndexBuf := make([]byte, 8)
	binary.LittleEndian.PutUint64(expireBlockIndexBuf, expireBlockIndexInt)

	ctx, err := cl.Bcl.CreateTransaction(byzcoin.Instruction{
		InstanceID: byzcoin.NewInstanceID(adid),
		Spawn: &byzcoin.Spawn{
			ContractID: byzcoin.ContractDeferredID,
			Args: []byzcoin.Argument{
				{
					Name:  "proposedTransaction",
					Value: proposedTransactionBuf,
				},
				{
					Name:  "expireBlockIndex",
					Value: expireBlockIndexBuf,
				},
			},
		},
		SignerCounter: []uint64{cl.signerCounter},
	})
	if err != nil {
		return *new(byzcoin.InstanceID), xerrors.Errorf("Creating the deffered transaction: %w", err)
	}
	err = cl.spawnTransaction(ctx)
	if err != nil {
		return *new(byzcoin.InstanceID), xerrors.Errorf("Adding transaction to the ledger: %w", err)
	}
	return ctx.Instructions[0].DeriveID(""), err
}
func (cl *Client) SynchronizeDefferedInstanceIDs() error {
	defferedIDs, err := cl.FetchNewDefferedInstanceIDs()
	if err != nil {
		return xerrors.Errorf("Fetching Instance IDs: %w", err)
	}
	for _, id := range defferedIDs.Ids {
		cl.pendingDefferedTx[id] = false
	}
	return nil
}

func (cl *Client) FetchNewDefferedInstanceIDs() (GetDeferredIDsReply, error) {
	res := GetDeferredIDsReply{}
	err := cl.scl.SendProtobuf(cl.Bcl.Roster.RandomServerIdentity(), &GetDeferredIDs{}, &res)
	if err != nil {
		return GetDeferredIDsReply{}, xerrors.Errorf("Sending the GetDeferredIDs request to the service : %w", err)
	}
	return res, nil
}

func (cl *Client) AddSignatureToDefferedTx(instID byzcoin.InstanceID, instIdx uint64) error {
	if _, ok := cl.pendingDefferedTx[instID]; !ok {
		cl.pendingDefferedTx[instID] = false
	}
	result, err := cl.Bcl.GetDeferredData(instID)
	if err != nil {
		return xerrors.Errorf("Getting the deffered instance from chain: %w", err)
	}
	rootHash := result.InstructionHashes
	identity := cl.AuthKey().Identity()
	identityBuf, err := protobuf.Encode(&identity)
	if err != nil {
		return xerrors.Errorf("Encoding the identity of signer: %w", err)
	}
	signature, err := cl.AuthKey().Sign(rootHash[0]) // == index
	if err != nil {
		return xerrors.Errorf("Signing the deffered transaction: %w", err)
	}
	// TODO: Implement multi instructions transactions.
	index := uint32(instIdx) // The index of the instruction to sign in the transaction
	indexBuf := make([]byte, 4)
	binary.LittleEndian.PutUint32(indexBuf, uint32(index))

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
	cl.pendingDefferedTx[instID] = true
	return cl.spawnTransaction(ctx)
}

func (cl *Client) ExecDefferedTx(instID byzcoin.InstanceID) error {
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

func (cl *Client) CreateNewProject(adid darc.ID, pname string) (byzcoin.InstanceID, darc.ID, string, error) {
	proposedTransaction, pdarcID, pdarcIDString, err := cl.createProjectDarc(pname, adid)
	if err != nil {
		return byzcoin.InstanceID{}, darc.ID{}, "", xerrors.Errorf("Crafting the project darc spawn transaction: %w", err)
	}
	id, err := cl.addDeferredTransaction(proposedTransaction, adid)
	if err != nil {
		return byzcoin.InstanceID{}, darc.ID{}, "", xerrors.Errorf("Spawning the deffered transaction: %w", err)
	}
	return id, pdarcID, pdarcIDString, err
}

func (cl *Client) createProjectDarc(pname string, adid darc.ID) (byzcoin.ClientTransaction, darc.ID, string, error) {
	pdarcDescription := pname
	adminDarc, err := lib.GetDarcByID(cl.Bcl, adid)
	if err != nil {
		return byzcoin.ClientTransaction{}, darc.ID{}, "", xerrors.Errorf("Getting the admin darc from chain: %w", err)
	}
	rules := darc.InitRules([]darc.Identity{cl.adminkeys.Identity()}, []darc.Identity{cl.adminkeys.Identity()})
	pdarc := darc.NewDarc(rules, []byte(pdarcDescription)) //TODO share admin identities
	pdarcActions := "spawn:accessright,invoke:accessright.add,invoke:accessright.update,invoke:accessright.delete"
	pdarcExpr := createMultisigRuleExpression([]string{adminDarc.GetIdentityString()})
	err = AddRuleToDarc(pdarc, pdarcActions, pdarcExpr)
	pdarcActions = "_name:accessright"
	pdarcExpr = expression.InitOrExpr(cl.adminkeys.Identity().String()) //TODO sync the id of all admins
	err = AddRuleToDarc(pdarc, pdarcActions, pdarcExpr)
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

	return ctx, pdarc.GetBaseID(), pdarc.GetIdentityString(), nil
}

func (cl *Client) CreateAccessRight(pdarc, adid darc.ID) (byzcoin.InstanceID, error) {
	emptyAccess := AccessRight{[]string{}, []string{}}
	buf, err := protobuf.Encode(&emptyAccess)
	if err != nil {
		return byzcoin.InstanceID{}, xerrors.Errorf("Encoding the access right struct: %w", err)
	}
	ctx, err := cl.Bcl.CreateTransaction(byzcoin.Instruction{
		InstanceID: byzcoin.NewInstanceID(pdarc),
		Spawn: &byzcoin.Spawn{
			ContractID: ContractAccessRightID,
			Args: []byzcoin.Argument{{
				Name:  "ar",
				Value: buf,
			}},
		},
	})
	if err != nil {
		return byzcoin.InstanceID{}, xerrors.Errorf("Creating the transaction: %w", err)
	}
	id, err := cl.addDeferredTransaction(ctx, adid)
	if err != nil {
		return byzcoin.InstanceID{}, xerrors.Errorf("Adding Deffered transaction to the ledger: %w", err)
	}
	return id, nil
}

//TODO the _name contract doesn't implement VerifyDeferredData -> PR ?
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

func (cl *Client) GetAccessRightInstanceID(id byzcoin.InstanceID) (byzcoin.InstanceID, error) {
	result, err := cl.Bcl.GetDeferredData(id)
	if err != nil {
		return byzcoin.InstanceID{}, xerrors.Errorf("Getting the deferred data: %w", err)
	}
	finalId := result.ExecResult[0] //TODO argument instruction id
	return byzcoin.NewInstanceID(finalId), nil
}

func (cl *Client) GetLastSignerCounter() uint64 {
	signerCtrs, _ := cl.Bcl.GetSignerCounters(cl.AuthKey().Identity().String())
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

func IndexOf(rule string, rules []string) int {
	for k, v := range rules {
		if rule == v {
			return k
		}
	}
	return -1
}

func (cl *Client) GetAccessRightFromProjectDarcID(pdid darc.ID) (*AccessRight, byzcoin.InstanceID, error) {
	arid, err := cl.Bcl.ResolveInstanceID(pdid, "AR")
	if err != nil {
		return &AccessRight{}, byzcoin.InstanceID{}, xerrors.Errorf("Resolving the instance id of access right instance: %w", err)
	}
	pr, err := cl.Bcl.GetProof(arid.Slice())
	if err != nil {
		return &AccessRight{}, byzcoin.InstanceID{}, xerrors.Errorf("Resolving the proof of the access right instance: %w", err)
	}
	v0, _, _, err := pr.Proof.Get(arid.Slice())
	if err != nil {
		return &AccessRight{}, byzcoin.InstanceID{}, xerrors.Errorf("Getting the proof of access right: %w", err)
	}
	ar := AccessRight{}
	err = protobuf.Decode(v0, &ar)
	if err != nil {
		return &AccessRight{}, byzcoin.InstanceID{}, xerrors.Errorf("Encoding: %w", err)
	}
	return &ar, arid, nil
}

func (cl *Client) AddQuerierToProject(pdid, adid darc.ID, qid, access string) (byzcoin.InstanceID, error) {
	arid, err := cl.Bcl.ResolveInstanceID(pdid, "AR")
	if err != nil {
		return byzcoin.InstanceID{}, xerrors.Errorf("Reolving access right instance: %w", err)
	}
	ctx, err := cl.Bcl.CreateTransaction(byzcoin.Instruction{
		InstanceID: arid,
		Invoke: &byzcoin.Invoke{
			ContractID: ContractAccessRightID,
			Command:    "add",
			Args: []byzcoin.Argument{{
				Name:  "id",
				Value: []byte(qid),
			},
				{
					Name:  "ar",
					Value: []byte(access),
				}},
		},
		SignerCounter: []uint64{cl.signerCounter},
	})
	if err != nil {
		return byzcoin.InstanceID{}, xerrors.Errorf("Creating the transaction: %w", err)
	}
	id, err := cl.addDeferredTransaction(ctx, adid)
	if err != nil {
		return byzcoin.InstanceID{}, xerrors.Errorf("Adding Deffered transaction to the ledger: %w", err)
	}
	return id, nil
}

func (cl *Client) RemoveQuerierFromProject(pdid, adid darc.ID, qid string) (byzcoin.InstanceID, error) {
	arid, err := cl.Bcl.ResolveInstanceID(pdid, "AR")
	if err != nil {
		return byzcoin.InstanceID{}, xerrors.Errorf("Reolving access right instance: %w", err)
	}
	ctx, err := cl.Bcl.CreateTransaction(byzcoin.Instruction{
		InstanceID: arid,
		Invoke: &byzcoin.Invoke{
			ContractID: ContractAccessRightID,
			Command:    "delete",
			Args: []byzcoin.Argument{{
				Name:  "id",
				Value: []byte(qid),
			},
			},
		},
		SignerCounter: []uint64{cl.signerCounter},
	})
	if err != nil {
		return byzcoin.InstanceID{}, xerrors.Errorf("Creating the transaction: %w", err)
	}
	id, err := cl.addDeferredTransaction(ctx, adid)
	if err != nil {
		return byzcoin.InstanceID{}, xerrors.Errorf("Adding Deffered transaction to the ledger: %w", err)
	}
	return id, nil
}

func (cl *Client) ModifyQuerierAccessRightsForProject(pdid, adid darc.ID, qid, access string) (byzcoin.InstanceID, error) {
	arid, err := cl.Bcl.ResolveInstanceID(pdid, "AR")
	if err != nil {
		return byzcoin.InstanceID{}, xerrors.Errorf("Reolving access right instance: %w", err)
	}
	ctx, err := cl.Bcl.CreateTransaction(byzcoin.Instruction{
		InstanceID: arid,
		Invoke: &byzcoin.Invoke{
			ContractID: ContractAccessRightID,
			Command:    "update",
			Args: []byzcoin.Argument{{
				Name:  "id",
				Value: []byte(qid),
			},
				{
					Name:  "ar",
					Value: []byte(access),
				}},
		},
		SignerCounter: []uint64{cl.signerCounter},
	})
	if err != nil {
		return byzcoin.InstanceID{}, xerrors.Errorf("Creating the transaction: %w", err)
	}
	id, err := cl.addDeferredTransaction(ctx, adid)
	if err != nil {
		return byzcoin.InstanceID{}, xerrors.Errorf("Adding Deffered transaction to the ledger: %w", err)
	}
	return id, nil
}

func (cl *Client) VerifyAccessRights(qid, access string, pdid darc.ID) (bool, error) {
	arid, err := cl.Bcl.ResolveInstanceID(pdid, "AR")
	if err != nil {
		return false, xerrors.Errorf("Reolving access right instance: %w", err)
	}
	pr, err := cl.Bcl.GetProof(arid.Slice())
	if err != nil {
		return false, xerrors.Errorf("Getting the proof of the accessright contract instance: %w", err)
	}
	vv, _, _, err := pr.Proof.Get(arid.Slice())
	if err != nil {
		return false, xerrors.Errorf("Getting the value: %w", err)
	}
	received := AccessRight{}
	err = protobuf.Decode(vv, &received)
	if err != nil {
		return false, xerrors.Errorf("Unmarshalling: %w", err)
	}
	idx, _ := Find(received.Ids, qid)
	if idx == -1 {
		return false, xerrors.Errorf("There is no such querier registered in the accessright contract")
	}

	return strings.Contains(received.Access[idx], access), nil
}

func (cl *Client) ShowAccessRights(qid string, pdid darc.ID) (string, error) {
	arid, err := cl.Bcl.ResolveInstanceID(pdid, "AR")
	if err != nil {
		return "", xerrors.Errorf("Reolving access right instance: %w", err)
	}
	pr, err := cl.Bcl.GetProof(arid.Slice())
	if err != nil {
		return "", xerrors.Errorf("Getting the proof of the accessright contract instance: %w", err)
	}
	vv, _, _, err := pr.Proof.Get(arid.Slice())
	if err != nil {
		return "", xerrors.Errorf("Getting the value: %w", err)
	}
	received := AccessRight{}
	err = protobuf.Decode(vv, &received)
	if err != nil {
		return "", xerrors.Errorf("Unmarshalling: %w", err)
	}
	idx, _ := Find(received.Ids, qid)
	if idx == -1 {
		return "", xerrors.Errorf("There is no such querier registered in the accessright contract")
	}

	return received.Access[idx], nil
}

func (cl *Client) spawnTransaction(ctx byzcoin.ClientTransaction) error {
	err := ctx.FillSignersAndSignWith(cl.adminkeys)
	if err != nil {
		return xerrors.Errorf("Signing: %w", err)
	}
	_, err = cl.Bcl.AddTransactionAndWait(ctx, 10)
	if err != nil {
		return xerrors.Errorf("Adding transaction to the ledger: %w", err)
	}
	cl.incrementSignerCounter()
	return nil
}

// TODO Create util package to reuse methods
func AddRuleToDarc(userDarc *darc.Darc, action string, expr expression.Expr) error {
	actions := strings.Split(action, ",")

	for i := 0; i < len(actions); i++ {
		dAction := darc.Action(actions[i])
		err := userDarc.Rules.AddRule(dAction, expr)
		if err != nil {
			return err
		}
	}
	return nil
}
