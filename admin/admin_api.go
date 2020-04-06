package admin

import (
	"encoding/binary"
	"errors"
	"strings"

	"go.dedis.ch/cothority/v3/byzcoin"
	"go.dedis.ch/cothority/v3/byzcoin/bcadmin/lib"
	"go.dedis.ch/cothority/v3/darc"
	"go.dedis.ch/cothority/v3/darc/expression"
	"go.dedis.ch/protobuf"
	"golang.org/x/xerrors"
)

type Client struct {
	bcl           *byzcoin.Client
	adminkeys     darc.Signer
	genDarc       darc.Darc
	signerCounter uint64
}

func NewClient(bcl *byzcoin.Client) (*Client, error) {
	if bcl == nil {
		return nil, errors.New("A Byzcoin Client is required")
	}
	client := &Client{
		bcl:           bcl,
		adminkeys:     darc.NewSignerEd25519(nil, nil),
		signerCounter: 1,
	}
	if genDarc, err := bcl.GetGenDarc(); err == nil {
		client.genDarc = *genDarc
		return client, nil
	} else {
		return nil, xerrors.Errorf("getting genesis darc from chain: %w", err)
	}
}

func NewClientWithAuth(bcl *byzcoin.Client, keys *darc.Signer) (*Client, error) {
	if keys == nil {
		return nil, errors.New("Keys are required")
	}
	client, err := NewClient(bcl)
	client.adminkeys = *keys
	return client, err
}

func (cl *Client) AuthKey() darc.Signer {
	return cl.adminkeys
}

func (cl *Client) SpawnNewAdminDarc() (*darc.Darc, error) {
	adminDarc, err := cl.createAdminDarc()
	if err != nil {
		return nil, err
	}
	adminDarcProto, err := adminDarc.ToProto()
	if err != nil {
		return nil, err
	}
	ctx, err := cl.bcl.CreateTransaction(byzcoin.Instruction{
		InstanceID: byzcoin.NewInstanceID(cl.genDarc.GetBaseID()),
		Spawn: &byzcoin.Spawn{
			ContractID: byzcoin.ContractDarcID,
			Args: byzcoin.Arguments{
				{
					Name:  "darc",
					Value: adminDarcProto,
				},
			},
		},
		// SignerIdentities: []darc.Identity{superAdmin.Identity()},
		SignerCounter: []uint64{cl.signerCounter},
	})
	err = ctx.FillSignersAndSignWith(cl.adminkeys)
	if err != nil {
		return nil, xerrors.Errorf("Signing: %w", err)
	}
	_, err = cl.bcl.AddTransaction(ctx)
	if err != nil {
		return nil, xerrors.Errorf("Adding transaction to the ledger: %w", err)
	}
	cl.incrementSignerCounter()
	return adminDarc, err

}

func (cl *Client) createAdminDarc() (*darc.Darc, error) {
	adminDarcDescription := "Admin darc guards medchain project darcs"
	rules := darc.InitRules([]darc.Identity{cl.adminkeys.Identity()}, []darc.Identity{cl.adminkeys.Identity()})
	adminDarc := darc.NewDarc(rules, []byte(adminDarcDescription))
	adminDarcActions := "invoke:darc.evolve,spawn:value,spawn:deferred,invoke:deferred.addProof,invoke:deferred.execProposedTx"
	adminDarcExpr := expression.InitAndExpr(cl.adminkeys.Identity().String())
	err := AddRuleToDarc(adminDarc, adminDarcActions, adminDarcExpr)
	return adminDarc, err
}

func (cl *Client) AddAdminToAdminDarc(adid darc.ID, newAdmin darc.Identity) (byzcoin.InstanceID, error) {
	adminDarc, err := lib.GetDarcByID(cl.bcl, adid)
	if err != nil {
		return *new(byzcoin.InstanceID), xerrors.Errorf("Getting the admin darc from chain: %w", err)
	}
	exp := adminDarc.Rules.GetEvolutionExpr()
	slc := strings.Split(string(exp), "&")
	for i := range slc {
		slc[i] = strings.TrimSpace(slc[i])
	}
	slc = append(slc, newAdmin.String())
	evolvedAdminDarc := adminDarc.Copy()
	newAdminAndExpr := expression.InitAndExpr(slc...)
	newAdminOrExpr := expression.InitOrExpr(slc...)
	err = cl.UpdateAdminRules(evolvedAdminDarc, newAdminAndExpr, newAdminOrExpr)
	if err != nil {
		return *new(byzcoin.InstanceID), xerrors.Errorf("Updating admin rules: %w", err)
	}
	err = evolvedAdminDarc.EvolveFrom(adminDarc)
	if err != nil {
		return *new(byzcoin.InstanceID), xerrors.Errorf("Evolving the admin darc: %w", err)
	}
	_, evolvedAdminDarc2Buf, err := evolvedAdminDarc.MakeEvolveRequest(cl.AuthKey())
	if err != nil {
		return *new(byzcoin.InstanceID), xerrors.Errorf("Creating the evolution request: %w", err)
	}

	proposedTransaction, err := cl.bcl.CreateTransaction(byzcoin.Instruction{
		InstanceID: byzcoin.NewInstanceID(adminDarc.GetBaseID()),
		Invoke: &byzcoin.Invoke{
			ContractID: byzcoin.ContractDarcID,
			Command:    "evolve",
			Args: []byzcoin.Argument{{
				Name:  "darc",
				Value: evolvedAdminDarc2Buf,
			}},
		},
	})
	if err != nil {
		return *new(byzcoin.InstanceID), xerrors.Errorf("Creating the transaction: %w", err)
	}
	proposedTransactionBuf, err := protobuf.Encode(&proposedTransaction)
	if err != nil {
		return *new(byzcoin.InstanceID), xerrors.Errorf("Marshalling the transaction: %w", err)
	}
	ctxID, err := cl.spawnDeferredInstance(proposedTransactionBuf, adid)
	if err != nil {
		return *new(byzcoin.InstanceID), xerrors.Errorf("Creating the deffered transaction: %w", err)
	}
	return ctxID, nil
}
func (cl *Client) UpdateAdminRules(evolvedAdminDarc *darc.Darc, newAdminAndExpr, newAdminOrExpr expression.Expr) error {
	err := evolvedAdminDarc.Rules.UpdateEvolution(newAdminAndExpr)
	if err != nil {
		return xerrors.Errorf("Updating the _evolve expression in admin darc: %w", err)
	}
	err = evolvedAdminDarc.Rules.UpdateSign(newAdminOrExpr)
	if err != nil {
		return xerrors.Errorf("Updating the _sign expression in admin darc: %w", err)
	}
	err = evolvedAdminDarc.Rules.UpdateRule("invoke:deferred.addProof", newAdminOrExpr)
	if err != nil {
		return xerrors.Errorf("Updating the invoke:deferred.addProof expression in admin darc: %w", err)
	}
	err = evolvedAdminDarc.Rules.UpdateRule("invoke:deferred.execProposedTx", newAdminOrExpr)
	if err != nil {
		return xerrors.Errorf("Updating the invoke:deferred.execProposedTx expression in admin darc: %w", err)
	}
	err = evolvedAdminDarc.Rules.UpdateRule("invoke:darc.evolve", newAdminAndExpr)
	if err != nil {
		return xerrors.Errorf("Updating the invoke:darc.evolve expression in admin darc: %w", err)
	}
	return nil
}

func (cl *Client) AddSignatureToDefferedTx(myID byzcoin.InstanceID) error {
	result, err := cl.bcl.GetDeferredData(myID)
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
	index := uint32(0) // The index of the instruction to sign in the transaction
	indexBuf := make([]byte, 4)
	binary.LittleEndian.PutUint32(indexBuf, uint32(index))

	ctx, err := cl.bcl.CreateTransaction(byzcoin.Instruction{
		InstanceID: myID,
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
	err = ctx.FillSignersAndSignWith(cl.AuthKey())
	if err != nil {
		return xerrors.Errorf("Signing: %w", err)
	}
	_, err = cl.bcl.AddTransactionAndWait(ctx, 10)
	if err != nil {
		return xerrors.Errorf("Adding transaction to the ledger: %w", err)
	}
	cl.incrementSignerCounter()
	return nil
}

func (cl *Client) ExecDefferedTx(myID byzcoin.InstanceID) error {
	ctx, err := cl.bcl.CreateTransaction(byzcoin.Instruction{
		InstanceID: myID,
		Invoke: &byzcoin.Invoke{
			ContractID: byzcoin.ContractDeferredID,
			Command:    "execProposedTx",
		},
		SignerCounter: []uint64{cl.signerCounter},
	})
	if err != nil {
		return xerrors.Errorf("Creating the transaction: %w", err)
	}
	err = ctx.FillSignersAndSignWith(cl.AuthKey())
	if err != nil {
		return xerrors.Errorf("Signing: %w", err)
	}
	_, err = cl.bcl.AddTransactionAndWait(ctx, 10)
	if err != nil {
		return xerrors.Errorf("Adding transaction to the ledger: %w", err)
	}
	cl.incrementSignerCounter()
	return nil
}

func (cl *Client) getLastSignerCounter() uint64 {
	signerCtrs, _ := cl.bcl.GetSignerCounters(cl.AuthKey().Identity().String())
	return signerCtrs.Counters[0]
}
func (cl *Client) incrementSignerCounter() {
	cl.signerCounter++
}
func (cl *Client) updateSignerCounter(sc uint64) {
	cl.signerCounter = sc
}

func (cl *Client) spawnDeferredInstance(proposedTransactionBuf []byte, adid darc.ID) (byzcoin.InstanceID, error) {
	expireBlockIndexInt := uint64(6000)
	expireBlockIndexBuf := make([]byte, 8)
	binary.LittleEndian.PutUint64(expireBlockIndexBuf, expireBlockIndexInt)

	ctx, err := cl.bcl.CreateTransaction(byzcoin.Instruction{
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
	err = ctx.FillSignersAndSignWith(cl.adminkeys)
	if err != nil {
		return *new(byzcoin.InstanceID), xerrors.Errorf("Signing: %w", err)
	}
	_, err = cl.bcl.AddTransactionAndWait(ctx, 10)
	if err != nil {
		return *new(byzcoin.InstanceID), xerrors.Errorf("Spawning the deffered transaction: %w", err)
	}
	cl.incrementSignerCounter() // Need to check for errors before incrementing the signer counter
	return ctx.Instructions[0].DeriveID(""), err
}

func IndexOf(rule string, rules []string) int {
	for k, v := range rules {
		if rule == v {
			return k
		}
	}
	return -1
}

func (cl *Client) RemoveAdminFromAdminDarc(adid darc.ID, adminId darc.Identity) (byzcoin.InstanceID, error) {
	adminDarc, err := lib.GetDarcByID(cl.bcl, adid)
	if err != nil {
		return *new(byzcoin.InstanceID), xerrors.Errorf("Getting the admin darc from chain: %w", err)
	}
	exp := adminDarc.Rules.GetEvolutionExpr()
	slc := strings.Split(string(exp), "&")
	for i := range slc {
		slc[i] = strings.TrimSpace(slc[i])
	}
	idx := IndexOf(adminId.String(), slc)
	if idx == -1 {
		return *new(byzcoin.InstanceID), xerrors.Errorf("The adminID doesn't exist in the admin darc")
	}
	slc = append(slc[:idx], slc[idx+1:]...)
	evolvedAdminDarc := adminDarc.Copy()
	newAdminAndExpr := expression.InitAndExpr(slc...)
	newAdminOrExpr := expression.InitOrExpr(slc...)
	err = cl.UpdateAdminRules(evolvedAdminDarc, newAdminAndExpr, newAdminOrExpr)
	if err != nil {
		return *new(byzcoin.InstanceID), xerrors.Errorf("Updating admin rules: %w", err)
	}
	err = evolvedAdminDarc.EvolveFrom(adminDarc)
	if err != nil {
		return *new(byzcoin.InstanceID), xerrors.Errorf("Evolving the admin darc: %w", err)
	}
	_, evolvedAdminDarc2Buf, err := evolvedAdminDarc.MakeEvolveRequest(cl.AuthKey())
	if err != nil {
		return *new(byzcoin.InstanceID), xerrors.Errorf("Creating the evolution request: %w", err)
	}

	proposedTransaction, err := cl.bcl.CreateTransaction(byzcoin.Instruction{
		InstanceID: byzcoin.NewInstanceID(adminDarc.GetBaseID()),
		Invoke: &byzcoin.Invoke{
			ContractID: byzcoin.ContractDarcID,
			Command:    "evolve",
			Args: []byzcoin.Argument{{
				Name:  "darc",
				Value: evolvedAdminDarc2Buf,
			}},
		},
	})
	if err != nil {
		return *new(byzcoin.InstanceID), xerrors.Errorf("Creating the transaction: %w", err)
	}
	proposedTransactionBuf, err := protobuf.Encode(&proposedTransaction)
	if err != nil {
		return *new(byzcoin.InstanceID), xerrors.Errorf("Marshalling the transaction: %w", err)
	}
	ctxID, err := cl.spawnDeferredInstance(proposedTransactionBuf, adid)
	if err != nil {
		return *new(byzcoin.InstanceID), xerrors.Errorf("Creating the deffered transaction: %w", err)
	}
	return ctxID, nil
}

func (cl *Client) ModifyAdminKeysFromAdminDarc() {

}

func (cl *Client) CreateNewProject() {
}

func (cl *Client) AddQuerierToProject() {

}

func (cl *Client) RemoveQuerierFromProject() {

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
