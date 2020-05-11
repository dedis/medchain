// Package medchain is the client side API for communicating with medchain
// service.
package medchain

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"strings"

	"go.dedis.ch/cothority/v3"
	"go.dedis.ch/cothority/v3/byzcoin"
	"go.dedis.ch/cothority/v3/darc"
	"go.dedis.ch/cothority/v3/darc/expression"
	"go.dedis.ch/cothority/v3/skipchain"
	"go.dedis.ch/kyber/v3"
	"go.dedis.ch/kyber/v3/util/key"
	"go.dedis.ch/onet/v3"
	"go.dedis.ch/onet/v3/log"
	"go.dedis.ch/onet/v3/network"
	"go.dedis.ch/protobuf"
	"golang.org/x/xerrors"
)

// QueryKey is an opaque unique identifier useful to find a given query later
// via GetQuery.
var QueryKey []byzcoin.InstanceID

// Client is a structure to communicate with MedChain service
type Client struct {
	onetcl     *onet.Client
	sccl       *skipchain.Client
	Bcl        *byzcoin.Client
	ClientID   string
	entryPoint *network.ServerIdentity
	public     kyber.Point
	private    kyber.Scalar
	// The DarcID with "invoke:medchain.update" & "invoke:medchain.verifystatus "permission on it.
	DarcID darc.ID
	// Signers are the Darc signers that will sign transactions sent with this client.
	Signers []darc.Signer
	// Instance ID of naming contract
	NamingInstance byzcoin.InstanceID
	GenDarc        *darc.Darc
	// Map projects to their darcs
	AllDarcs   map[string]*darc.Darc
	AllDarcIDs map[string]darc.ID
	GMsg       *byzcoin.CreateGenesisBlock
	signerCtrs []uint64
}

// NewClient creates a new client to talk to the medchain service.
// Fields DarcID, Instance, and Signers must be filled in before use.
func NewClient(bcl *byzcoin.Client, entryPoint *network.ServerIdentity, clientID string) (*Client, error) {
	keys := key.NewKeyPair(TSuite)

	if bcl == nil {
		return nil, errors.New("Byzcoin client is required")
	}
	return &Client{
		Bcl:        bcl,
		onetcl:     onet.NewClient(cothority.Suite, ServiceName),
		sccl:       skipchain.NewClient(),
		ClientID:   clientID,
		entryPoint: entryPoint,
		public:     keys.Public,
		private:    keys.Private,
		signerCtrs: nil,
	}, nil
}

// Create creates a new medchain by spawning an instance of Naming contract. After
// this method is executed, c.NamingInstance will be correctly set.
func (c *Client) Create() error {
	if c.signerCtrs == nil {
		c.RefreshSignerCounters()
	}
	c.AllDarcs = make(map[string]*darc.Darc)
	c.AllDarcIDs = make(map[string]darc.ID)
	// Spawn an instance of naming contract
	namingTx, err := c.Bcl.CreateTransaction(
		byzcoin.Instruction{
			InstanceID: byzcoin.NewInstanceID(c.GenDarc.GetBaseID()),
			Spawn: &byzcoin.Spawn{
				ContractID: byzcoin.ContractNamingID,
			},
			SignerCounter: c.IncrementCtrs(),
		},
	)
	if err != nil {
		return err
	}

	err = c.spawnTx(namingTx)
	if err != nil {
		xerrors.Errorf("Could not add naming contract instace to the ledger: %w", err)
	}

	log.Lvl1("[INFO] (Create) Contract_name instance was added to the ledger")

	return nil
}

// AuthorizeQuery asks the service to write queries to the ledger.
func (c *Client) AuthorizeQuery(qu Query, instID byzcoin.InstanceID) (byzcoin.InstanceID, error) {
	log.Lvl1("[INFO] Authorization of query ")
	return c.createQueryAndWait(qu, instID)
}

// createQueryAndWait sends a request to create a query and waits for N block intervals
// that the queries are added to the ledger
func (c *Client) createQueryAndWait(qu Query, instID byzcoin.InstanceID) (byzcoin.InstanceID, error) {
	if c.signerCtrs == nil {
		c.RefreshSignerCounters()
	}

	ctx, err := c.prepareTx(qu, instID)
	if err != nil {
		return *new(byzcoin.InstanceID), xerrors.Errorf("Could not create transaction: %w", err)
	}
	err = c.spawnTx(ctx)
	if err != nil {
		return *new(byzcoin.InstanceID), xerrors.Errorf("Could not add transaction to ledger: %w", err)
	}

	// newInstID := ctx.Instructions[0].DeriveID("")
	log.Lvl1("[INFO] (Invoke) Query was added to the ledger")

	return instID, nil
}

// prepareTx prepares a transaction that will be committed to the ledger.
func (c *Client) prepareTx(query Query, instID byzcoin.InstanceID) (byzcoin.ClientTransaction, error) {

	ok := true
	//var action string
	var args byzcoin.Argument

	// Get the project darc
	project := c.getProjectFromOneQuery(query)
	darcID := c.AllDarcIDs[project]
	action := c.getActionFromOneQuery(query)

	// Check if the query is authorized/rejected
	authorizations, err := c.checkAuth(query, darcID, action)
	if err != nil {
		return *new(byzcoin.ClientTransaction), err
	}
	for _, res := range authorizations {
		if res == false {
			ok = false //reject the query as at least one of the signers can't sign
			args = byzcoin.Argument{
				Name:  query.ID,
				Value: []byte("Rejected"),
			}
			// This action will not be rejected by Darc and thus query rejection will be recorded
			// in the ledger
			action = "update"
			log.Lvl1("[INFO] (Invoke) Query was REJECTED")
		}
	}
	if ok {
		args = byzcoin.Argument{
			Name:  query.ID,
			Value: []byte("Authorized"),
		}
		action = c.getActionFromOneQuery(query)
		log.Lvl1("[INFO] (Invoke) Query was AUTHORIZED")

	}

	// // Get the instance ID of the query instance using its name
	// // For the sake of performance, one should try to avoidusing
	// // ResolveInstance() to get the instance ID using its name.
	// replyID, err := c.ByzCoin.ResolveInstanceID(darcID, query.ID)
	// if err != nil {
	// 	fmt.Println("debug6")
	// 	return nil, nil, err
	// }

	instr := byzcoin.Instruction{
		InstanceID: instID,
		Invoke: &byzcoin.Invoke{
			ContractID: ContractName,
			Command:    action,
			Args:       []byzcoin.Argument{args},
		},
		SignerCounter: c.IncrementCtrs(),
	}

	ctx, err := c.Bcl.CreateTransaction(instr)
	if err != nil {
		return *new(byzcoin.ClientTransaction), err
	}

	return ctx, nil
}

// AuthorizeDeferredQuery asks the service to write queries using deferred transactions to the ledger.
func (c *Client) AuthorizeDeferredQuery(spawnedKeys []byzcoin.InstanceID, qu ...Query) ([]Query, []byzcoin.InstanceID, error) {
	return c.CreateQueryAndWaitDeferred(10, spawnedKeys, qu...)
}

// CreateQueryAndWaitDeferred sends a request to create a query as a deferred transaction and waits for N block intervals
// that the queries are added to the ledger
func (c *Client) CreateQueryAndWaitDeferred(numInterval int, spawnedKeys []byzcoin.InstanceID, qu ...Query) ([]Query, []byzcoin.InstanceID, error) {
	if c.signerCtrs == nil {
		c.RefreshSignerCounters()
	}

	tx, keys, err := c.prepareDeferredTx(qu, spawnedKeys)
	if err != nil {
		fmt.Println("debug1")
		return qu, nil, err
	}
	if _, err := c.Bcl.AddTransactionAndWait(*tx, numInterval); err != nil {
		fmt.Println("debug2")
		return qu, nil, err
	}
	log.Lvl1("[INFO] (Invoke) Query was added to the ledger")

	return qu, keys, nil
}

// checkAuth checks authorizations for the query
func (c *Client) checkAuth(query Query, darcID darc.ID, action string) ([]bool, error) {
	// We need the identity part of the signatures before
	// calling ToDarcRequest() below, because the identities
	// go into the message digest.
	sigs := make([]darc.Signature, len(c.Signers))
	authorizations := make([]bool, len(c.Signers))
	for i, x := range c.Signers {
		sigs[i].Signer = x.Identity()
	}

	// Check signers' authorizations for a specific action
	for i, signer := range c.Signers {
		a, err := c.Bcl.CheckAuthorization(darcID, signer.Identity())
		if err != nil {
			return authorizations, err
		}

		for _, authAction := range a {
			if darc.Action("invoke:medchain."+action) == authAction {
				authorizations[i] = true
			} else {
				continue
			}
		}
	}
	return authorizations, nil
}

// prepareDeferredTx prepares a transaction and checks for authorazations of the transaction. The transaction wil then be committed to the ledger.
func (c *Client) prepareDeferredTx(queries []Query, spawnedKeys []byzcoin.InstanceID) (*byzcoin.ClientTransaction, []byzcoin.InstanceID, error) {

	ok := true
	//var action string
	var args byzcoin.Argument
	keys := make([]byzcoin.InstanceID, len(queries))
	instrs := make([]byzcoin.Instruction, len(queries))

	for i, query := range queries {
		// Get the project darc
		project := c.getProjectFromOneQuery(query)
		darcID := c.AllDarcIDs[project]
		action := c.getActionFromOneQuery(query)

		// Check if the query is authorized/rejected
		authorizations, err := c.checkAuth(query, darcID, action)
		if err != nil {
			return nil, nil, err
		}
		for _, res := range authorizations {
			if res == false {
				ok = false //reject the query as at least one of the signers can't sign
				args = byzcoin.Argument{
					Name:  query.ID,
					Value: []byte("Rejected"),
				}
				// This action will not be rejected by Darc and thus query rejection will be recorded
				// in the ledger
				action = "update"
				log.Lvl1("[INFO] (Invoke) Query was REJECTED")
			}
		}
		if ok {
			args = byzcoin.Argument{
				Name:  query.ID,
				Value: []byte("Authorized"),
			}
			action = c.getActionFromOneQuery(query)
			log.Lvl1("[INFO] (Invoke) Query was AUTHORIZED")

		}

		instrs[i] = byzcoin.Instruction{
			InstanceID: spawnedKeys[i],
			Invoke: &byzcoin.Invoke{
				ContractID: ContractName,
				Command:    action,
				Args:       []byzcoin.Argument{args},
			},
			SignerCounter: c.IncrementCtrs(),
		}

	}
	ctx, err := c.Bcl.CreateTransaction(instrs...)
	if err != nil {
		return new(byzcoin.ClientTransaction), nil, xerrors.Errorf("Could not create transaction: %w", err)
	}
	err = c.spawnTx(ctx)
	if err != nil {
		return new(byzcoin.ClientTransaction), nil, xerrors.Errorf("Could not add transaction to ledger: %w", err)
	}
	for i := range ctx.Instructions {
		keys[i] = ctx.Instructions[i].DeriveID("")
	}
	return &ctx, keys, nil
}

//SpawnQuery spawns a query instance
func (c *Client) SpawnQuery(qu Query) (byzcoin.InstanceID, error) {
	return c.createInstance(qu)
}

//CreateInstance spawns a query
func (c *Client) createInstance(query Query) (byzcoin.InstanceID, error) {

	//keys := make([]byzcoin.InstanceID, len(queries))
	//instrs := make([]byzcoin.Instruction, len(queries))
	//instIDs := make([]byzcoin.InstanceID, len(queries))

	// Get the project darc
	project := c.getProjectFromOneQuery(query)
	darcID := c.AllDarcIDs[project]

	// If the query has just been submitted, spawn a query instance;
	//otherwise, inoke an update to change its status
	// TODO: check proof instead of status to make this more stable and
	// reliable (the latter may not be very efficient)
	instr := byzcoin.Instruction{
		InstanceID: byzcoin.NewInstanceID(darcID),
		Spawn: &byzcoin.Spawn{
			ContractID: ContractName,
			Args: byzcoin.Arguments{
				{
					Name:  query.ID,
					Value: []byte(query.Status),
				},
			},
		},
		SignerCounter: c.IncrementCtrs(),
	}
	tx, err := c.Bcl.CreateTransaction(instr)
	if err != nil {
		return *new(byzcoin.InstanceID), err
	}
	if err != nil {
		return *new(byzcoin.InstanceID), xerrors.Errorf("could not create transaction: %w", err)
	}
	err = c.spawnTx(tx)
	if err != nil {
		return *new(byzcoin.InstanceID), xerrors.Errorf("could not add transaction to ledger: %w", err)
	}
	instID := tx.Instructions[0].DeriveID("")

	// Name the instance of the query with as its key using contract_name to
	// make retrievals easier
	err = c.nameInstance(instID, darcID, query.ID)
	if err != nil {
		return *new(byzcoin.InstanceID), xerrors.Errorf("could not name the instance: %w", err)
	}

	return instID, nil
}

//SpawnDeferredQuery spawns a query as well as a deferred contract with medchain contract
func (c *Client) SpawnDeferredQuery(req *AddDeferredQueryRequest) (*AddDeferredQueryReply, error) {
	log.Lvl1("[INFO] Spawning the deferred query ")

	if len(req.QueryID) == 0 {
		return nil, errors.New("query ID required")
	}

	if len(req.ClientID) == 0 {
		return nil, errors.New("ClientID required")
	}

	req.QueryStatus = "Submitted"
	log.Lvl1("[INFO] Spawning the deferred query ")
	return c.createDeferredInstance(req)
}

//createDeferredInstance spawns a query that
func (c *Client) createDeferredInstance(req *AddDeferredQueryRequest) (*AddDeferredQueryReply, error) {

	// var proposedTransaction byzcoin.ClientTransaction
	// Get the project darc
	query := Query{}
	query.ID = req.QueryID
	query.Status = req.QueryStatus
	project := c.getProjectFromOneQuery(query)
	darcID := c.AllDarcIDs[project]
	req.DarcID = darcID
	// If the query has just been submitted, spawn a query instance;
	// otherwise, invoke an update to change its status
	// TODO: check proof instead of status to make this more stable and
	// reliable (the latter may not be very efficient)
	proposedInstr := byzcoin.Instruction{
		InstanceID: byzcoin.NewInstanceID(darcID),
		Spawn: &byzcoin.Spawn{
			ContractID: ContractName,
			Args: byzcoin.Arguments{
				{
					Name:  req.QueryID,
					Value: []byte(req.QueryStatus),
				},
			},
		},
		SignerCounter: c.IncrementCtrs(),
	}

	// create deferred instance
	proposedTransaction, err := c.Bcl.CreateTransaction(proposedInstr)
	proposedTransactionBuf, err := protobuf.Encode(&proposedTransaction)
	if err != nil {
		return nil, err
	}

	req.QueryInstID, err = c.spawnDeferredInstance(query, proposedTransactionBuf, darcID)
	if err != nil {
		return nil, xerrors.Errorf("could not spawn instance: %w", err)
	}
	req.BlockID = c.Bcl.ID
	reply := &AddDeferredQueryReply{}
	err = c.onetcl.SendProtobuf(c.entryPoint, req, reply)
	if err != nil {
		return nil, xerrors.Errorf("could not get reply from service: %w", err)
	}
	return reply, nil
}

// SpawnDeferredInstance spwans a deferred instance
func (c *Client) spawnDeferredInstance(query Query, proposedTransactionBuf []byte, darcID darc.ID) (byzcoin.InstanceID, error) {

	// TODO: make this an env var
	expireBlockIndexInt := uint64(6000)
	expireBlockIndexBuf := make([]byte, 8)
	binary.LittleEndian.PutUint64(expireBlockIndexBuf, expireBlockIndexInt)

	numExecutionInt := uint64(3)
	numExecutionBuf := make([]byte, 8)
	binary.LittleEndian.PutUint64(numExecutionBuf, numExecutionInt)

	ctx, err := c.Bcl.CreateTransaction(byzcoin.Instruction{
		InstanceID: byzcoin.NewInstanceID(darcID),
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
				{
					Name:  "numExecution",
					Value: numExecutionBuf,
				},
			},
		},
		SignerCounter: c.signerCtrs,
	})
	if err != nil {
		return *new(byzcoin.InstanceID), xerrors.Errorf("Could not create deferred transaction: %w", err)
	}
	err = c.spawnTx(ctx)
	if err != nil {
		return *new(byzcoin.InstanceID), xerrors.Errorf("Could not add deferred transaction to ledger: %w", err)
	}
	instID := ctx.Instructions[0].DeriveID("")

	// Name the instance of the query with as its key using contract_name to
	// make retrievals easier
	err = c.nameInstance(instID, darcID, query.ID)
	if err != nil {
		fmt.Println("debug4")
		return *new(byzcoin.InstanceID), err
	}
	log.Lvl1("[INFO] Deferred Query instance was named ")

	return instID, err
}

// AddSignatureToDeferredQuery allows MedChain user to sign a deferred query
// by invoking an addProof action from the deferred contract on the deferred
// query instance
func (c *Client) AddSignatureToDeferredQuery(req *SignDeferredTxRequest) (*SignDeferredTxReply, error) {
	log.Lvl1("[INFO] Add signature to the query transaction")
	if len(req.QueryID) == 0 {
		return nil, errors.New("query ID required")
	}

	if len(req.ClientID) == 0 {
		return nil, errors.New("ClientID required")
	}

	if len(req.QueryInstID) == 0 {
		return nil, errors.New("query instance ID required")
	}

	result, err := c.Bcl.GetDeferredData(req.QueryInstID)
	if err != nil {
		return nil, xerrors.Errorf("failed to get deffered instance from skipchain: %w", err)
	}
	rootHash := result.InstructionHashes
	signer := c.Signers[0]
	identity := signer.Identity() // TODO: Sign with private key of client
	identityBuf, err := protobuf.Encode(&identity)
	if err != nil {
		return nil, xerrors.Errorf("could not get the user identity: %w", err)
	}
	signature, err := signer.Sign(rootHash[0]) // == index
	if err != nil {
		return nil, xerrors.Errorf("could not sign the deffered query: %w", err)
	}

	index := uint32(0) // The index of the instruction to sign in the transaction
	indexBuf := make([]byte, 4)
	binary.LittleEndian.PutUint32(indexBuf, uint32(index))

	ctx, err := c.Bcl.CreateTransaction(byzcoin.Instruction{
		InstanceID: req.QueryInstID,
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
		SignerCounter: c.IncrementCtrs(),
	})
	if err != nil {
		return nil, xerrors.Errorf("failed to create the transaction: %w", err)
	}

	err = c.spawnTx(ctx)
	if err != nil {
		return nil, xerrors.Errorf("failed to sign the deferred transaction: %w", err)
	}

	reply := &SignDeferredTxReply{}
	err = c.onetcl.SendProtobuf(c.entryPoint, req, reply)
	if err != nil {
		return nil, xerrors.Errorf("could not get reply from service: %w", err)
	}
	return reply, nil
}

// ExecDefferedQuery executes the query that has received enough signatures
func (c *Client) ExecDefferedQuery(instID byzcoin.InstanceID) error {
	log.Lvl1("[INFO] Execute the query transaction")
	ctx, err := c.Bcl.CreateTransaction(byzcoin.Instruction{
		InstanceID: instID,
		Invoke: &byzcoin.Invoke{
			ContractID: byzcoin.ContractDeferredID,
			Command:    "execProposedTx",
		},
		SignerCounter: c.IncrementCtrs(),
	})
	if err != nil {
		return xerrors.Errorf("transaction execution failed: %w", err)
	}
	return c.spawnTx(ctx)
}

// VerifStatus retrieves the status of the query from skipchain
func (c *Client) VerifStatus(req *VerifyStatusRequest) (*VerifyStatusReply, error) {
	reply := &VerifyStatusReply{}
	if err := c.onetcl.SendProtobuf(c.entryPoint, req, reply); err != nil {
		return nil, err
	}
	return reply, nil
}

// CreateNewSigner creates private and public key pairs
func (c *Client) CreateNewSigner(public kyber.Point, private kyber.Scalar) darc.Signer {
	identity := darc.NewSignerEd25519(public, private)
	return identity
}

// AddRuleToDarc adds action rules to the given darc
func (c *Client) AddRuleToDarc(userDarc *darc.Darc, action string, expr expression.Expr) *darc.Darc {
	actions := strings.Split(action, ",")

	for i := 0; i < len(actions); i++ {
		dAction := darc.Action(actions[i])
		userDarc.Rules.AddRule(dAction, expr)
	}
	return userDarc
}

// UpdateDarcRule update action rules of the given darc
func (c *Client) UpdateDarcRule(userDarc *darc.Darc, action string, expr expression.Expr) *darc.Darc {
	actions := strings.Split(action, ",")

	for i := 0; i < len(actions); i++ {
		dAction := darc.Action(actions[i])
		userDarc.Rules.UpdateRule(dAction, expr)
	}
	return userDarc
}

// CreateDarc is used to create a new darc
func (c *Client) CreateDarc(name string, rules darc.Rules, actions string, exprs expression.Expr) (*darc.Darc, error) {
	projectDarc := darc.NewDarc(rules, []byte(name))
	projectDarc = c.AddRuleToDarc(projectDarc, actions, exprs)
	return projectDarc, nil
}

// AddProjectDarc is used to create project darcs with default rules
func (c *Client) AddProjectDarc(name string) error {

	rules := darc.InitRules([]darc.Identity{c.Signers[0].Identity()}, []darc.Identity{c.Signers[0].Identity()})
	c.AllDarcs[name] = darc.NewDarc(rules, []byte(name))
	// Add _name to Darc rule so that we can name the instances using contract_name
	expr := expression.InitOrExpr(c.Signers[0].Identity().String())
	c.AllDarcs[name].Rules.AddRule("_name:"+ContractName, expr)
	c.AllDarcs[name].Rules.AddRule("spawn:naming", expr)
	darcBuf, err := c.AllDarcs[name].ToProto()
	if err != nil {
		return err
	}
	ctx, err := c.Bcl.CreateTransaction(byzcoin.Instruction{
		InstanceID: byzcoin.NewInstanceID(c.GenDarc.GetBaseID()),
		Spawn: &byzcoin.Spawn{
			ContractID: byzcoin.ContractDarcID,
			Args: byzcoin.Arguments{
				{
					Name:  "darc",
					Value: darcBuf,
				},
			},
		},
		SignerIdentities: []darc.Identity{c.Signers[0].Identity()},
		SignerCounter:    c.IncrementCtrs(),
	})
	if err != nil {
		return err
	}

	err = ctx.FillSignersAndSignWith(c.Signers...)
	if err != nil {
		return err
	}

	_, err = c.Bcl.AddTransactionAndWait(ctx, 10)
	if err != nil {
		return err
	}
	c.AllDarcIDs[name] = c.AllDarcs[name].GetBaseID()

	return nil
}

// AddAdminDarc is used to create admin darcs by the super admin (darc) only
func (c *Client) AddAdminDarc(name string) error {

	rules := darc.InitRules([]darc.Identity{c.Signers[0].Identity()}, []darc.Identity{c.Signers[0].Identity()})
	c.AllDarcs[name] = darc.NewDarc(rules, []byte(name))
	// Add _name to Darc rule so that we can name the instances using contract_name
	expr := expression.InitOrExpr(c.Signers[0].Identity().String())
	c.AllDarcs[name].Rules.AddRule("_name:"+ContractName, expr)
	c.AllDarcs[name].Rules.AddRule("spawn:naming", expr)
	darcBuf, err := c.AllDarcs[name].ToProto()
	if err != nil {
		return err
	}
	ctx, err := c.Bcl.CreateTransaction(byzcoin.Instruction{
		InstanceID: byzcoin.NewInstanceID(c.GenDarc.GetBaseID()),
		Spawn: &byzcoin.Spawn{
			ContractID: byzcoin.ContractDarcID,
			Args: byzcoin.Arguments{
				{
					Name:  "darc",
					Value: darcBuf,
				},
			},
		},
		SignerIdentities: []darc.Identity{c.Signers[0].Identity()},
		SignerCounter:    c.IncrementCtrs(),
	})
	if err != nil {
		return xerrors.Errorf("Could not create transaction: %w", err)
	}
	err = c.spawnTx(ctx)
	if err != nil {
		xerrors.Errorf("Could not add transaction to ledger: %w", err)
	}
	c.AllDarcIDs[name] = c.AllDarcs[name].GetBaseID()

	return nil
}

// GetQuery asks the service to retrieve a query from the ledger by its key.
func (c *Client) GetQuery(key []byte) (*Query, error) {
	reply, err := c.Bcl.GetProof(key)
	if err != nil {
		return nil, err
	}
	if !reply.Proof.InclusionProof.Match(key) {
		return nil, errors.New("not an inclusion proof")
	}
	k, v0, _, _, err := reply.Proof.KeyValue()
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(k, key) {
		return nil, errors.New("wrong key")
	}
	q := Query{}
	err = protobuf.Decode(v0, &q)
	if err != nil {
		return nil, err
	}
	return &q, nil
}

// Close closes all the websocket connections.
func (c *Client) Close() error {
	err := c.Bcl.Close()
	if err2 := c.sccl.Close(); err2 != nil {
		err = err2
	}
	if err2 := c.onetcl.Close(); err2 != nil {
		err = err2
	}
	return err
}

// StreamHandler is the signature of the handler used when streaming queries.
type StreamHandler func(query Query, blockID []byte, err error)

// StreamQueries is a blocking call where it calls the handler on every new
// event until the connection is closed or the server stops.
func (c *Client) StreamQueries(handler StreamHandler) error {
	h := func(resp byzcoin.StreamingResponse, err error) {
		if err != nil {
			handler(Query{}, nil, err)
			return
		}
		// don't need to handle error because it's given to the handler
		_ = handleBlocks(handler, resp.Block)
	}
	// the following blocks
	return c.Bcl.StreamTransactions(h)
}

// StreamQueriesFrom is a blocking call where it calls the handler on even new
// event from (inclusive) the given block ID until the connection is closed or
// the server stops.
func (c *Client) StreamQueriesFrom(handler StreamHandler, id []byte) error {
	// 1. stream to a buffer (because we don't know which ones will be duplicates yet)
	blockChan := make(chan blockOrErr, 100)
	streamDone := make(chan error)
	go func() {
		err := c.Bcl.StreamTransactions(func(resp byzcoin.StreamingResponse, err error) {
			blockChan <- blockOrErr{resp.Block, err}
		})
		streamDone <- err
	}()

	// 2. use GetUpdateChain to find the missing events and call handler
	blocks, err := c.sccl.GetUpdateChainLevel(&c.Bcl.Roster, id, 0, -1)
	if err != nil {
		return err
	}
	for _, b := range blocks {
		// to keep the behaviour of the other streaming functions, we
		// don't return an error but let the handler decide what to do
		// with the error
		_ = handleBlocks(handler, b)
	}

	var latest *skipchain.SkipBlock
	if len(blocks) > 0 {
		latest = blocks[len(blocks)-1]
	}

	// 3. read from the buffer, remove duplicates and call the handler
	var foundLink bool
	for {
		select {
		case bOrErr := <-blockChan:
			if bOrErr.err != nil {
				handler(Query{}, nil, bOrErr.err)
				break
			}
			if !foundLink {
				if bOrErr.block.BackLinkIDs[0].Equal(latest.Hash) {
					foundLink = true
				}
			}
			if foundLink {
				_ = handleBlocks(handler, bOrErr.block)
			}
		case err := <-streamDone:
			return err
		}
	}
}

// EvolveDarc is used to evolve a darc
func (c *Client) EvolveDarc(d1 *darc.Darc, rules darc.Rules, name string, prevSigners ...darc.Signer) (*darc.Darc, error) {
	// Now the client wants to evolve the darc (change the owner), so it
	// creates a request and then sends it to the server.
	darcEvol := darc.NewDarc(rules, []byte(name))
	darcEvol.EvolveFrom(d1)
	r, d2Buf, _ := darcEvol.MakeEvolveRequest(prevSigners...)

	// Client sends request r and serialised darc d2Buf to the server, and
	// the server must verify it. Usually the server will look in its
	// database for the base ID of the darc in the request and find the
	// latest one. But in this case we assume it already knows. If the
	// verification is successful, then the server should add the darc in
	// the request to its database.
	d2Server, err := r.MsgToDarc(d2Buf)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(d2Server.GetID(), darcEvol.GetID()) {
		return nil, fmt.Errorf("Darc Evolution failed")
	}

	// If the darcs stored on the server are trustworthy, then using
	// `Request.Verify` is enough. To do a complete verification,
	// Darc.Verify should be used.
	fmt.Println(d2Server.VerifyWithCB(func(s string, latest bool) *darc.Darc {
		if s == darc.NewIdentityDarc(d1.GetID()).String() {
			return d1
		}
		return nil
	}, true))
	return darcEvol, nil
}

// // Search executes a search on the filter in req. See the definition of type
// // SearchRequest for additional details about how the filter is interpreted.
// // The ID and Instance fields of the SearchRequest will be filled in from c.
// func (c *Client) Search(req *SearchRequest) (*SearchResponse, error) {
// 	req.ID = c.ByzCoin.ID
// 	req.Instance = c.Instance

// 	reply := &SearchResponse{}
// 	if err := c.c.SendProtobuf(c.ByzCoin.Roster.List[0], req, reply); err != nil {
// 		return nil, err
// 	}
// 	return reply, nil
// }

// GetProjectFromQuery exports the projects to which queries are directed
func (c *Client) getProjectFromQuery(qu []Query) []string {
	projects := make([]string, len(qu))
	for i, query := range qu {
		projects[i] = strings.Split(query.ID, ":")[1]
	}
	return projects
}

// GetActionFromQuery exports the action from query
func (c *Client) getActionFromQuery(qu []Query) []string {
	//projects := c.GetProjectFromQuery(qu)
	actions := make([]string, len(qu))
	for i, query := range qu {
		actions[i] = strings.Split(query.ID, ":")[2]
		//actions[i] = "database" + projects[i] + "." + actions[i]
	}
	return actions
}

// GetProjectFromOneQuery exports the project to which query is directed
func (c *Client) getProjectFromOneQuery(query Query) string {
	project := strings.Split(query.ID, ":")[1]
	return project

}

// GetActionFromOneQuery exports the action from query
func (c *Client) getActionFromOneQuery(query Query) string {
	action := strings.Split(query.ID, ":")[2]
	return action
}

//NameInstance uses contract_name to name a contract instance
func (c *Client) nameInstance(instID byzcoin.InstanceID, darcID darc.ID, name string) error {

	ctx, err := c.Bcl.CreateTransaction(byzcoin.Instruction{
		InstanceID: byzcoin.NamingInstanceID,
		Invoke: &byzcoin.Invoke{
			ContractID: byzcoin.ContractNamingID,
			Command:    "add",
			Args: byzcoin.Arguments{
				{
					Name:  "instanceID",
					Value: instID.Slice(),
				},
				{
					Name:  "name",
					Value: []byte(name),
				},
			},
		},
		SignerCounter: c.IncrementCtrs(),
	})
	if err != nil {
		return xerrors.Errorf("Could not create transaction: %w", err)
	}
	err = c.spawnTx(ctx)
	if err != nil {
		return xerrors.Errorf("Could not add transaction to ledger: %w", err)
	}
	log.Lvl1("[INFO] (Naming Contract) Query was added to the ledger")

	// // This sanity check heavily reduces the performance
	// replyID, err := c.ByzCoin.ResolveInstanceID(darcID, name)
	// if err != nil {
	// 	return err
	// }
	// if replyID != instID {
	// 	return err
	// }
	return nil
}

func (c *Client) spawnTx(ctx byzcoin.ClientTransaction) error {
	err := ctx.FillSignersAndSignWith(c.Signers...)
	if err != nil {
		return xerrors.Errorf("Signing: %w", err)
	}
	_, err = c.Bcl.AddTransactionAndWait(ctx, 10)
	if err != nil {
		return xerrors.Errorf("Adding transaction to the ledger: %w", err)
	}
	return nil
}

type blockOrErr struct {
	block *skipchain.SkipBlock
	err   error
}

// RefreshSignerCounters talks to the service to get the latest signer
// counters, the client should call this function if the internal counters
// become de-synchronised.
func (c *Client) RefreshSignerCounters() {
	signerIDs := make([]string, len(c.Signers))
	for i := range c.Signers {
		signerIDs[i] = c.Signers[i].Identity().String()
	}
	signerCtrs, err := c.Bcl.GetSignerCounters(signerIDs...)
	if err != nil {
		log.Error(err)
		return
	}
	c.signerCtrs = signerCtrs.Counters
}

// IncrementCtrs is used to increment the signer counters
func (c *Client) IncrementCtrs() []uint64 {
	out := make([]uint64, len(c.signerCtrs))
	for i := range out {
		c.signerCtrs[i]++
		out[i] = c.signerCtrs[i]
	}
	return out
}

// nextCtrs will not update client state
func (c *Client) nextCtrs() []uint64 {
	out := make([]uint64, len(c.signerCtrs))
	for i := range out {
		out[i] = c.signerCtrs[i] + 1
	}
	return out
}

// handleBlocks calls the handler on the events of the block
func handleBlocks(handler StreamHandler, sb *skipchain.SkipBlock) error {
	var err error
	var header byzcoin.DataHeader
	err = protobuf.DecodeWithConstructors(sb.Data, &header, network.DefaultConstructors(cothority.Suite))
	if err != nil {
		err = errors.New("could not unmarshal header while streaming the queries " + err.Error())
		handler(Query{}, nil, err)
		return err
	}

	var body byzcoin.DataBody
	err = protobuf.DecodeWithConstructors(sb.Payload, &body, network.DefaultConstructors(cothority.Suite))
	if err != nil {
		err = errors.New("could not unmarshal body while streaming the queries " + err.Error())
		handler(Query{}, nil, err)
		return err
	}

	for _, tx := range body.TxResults {
		if tx.Accepted {
			for _, instr := range tx.ClientTransaction.Instructions {
				if instr.Invoke == nil {
					continue
				}
				if instr.Invoke.ContractID != MedchainContractID || instr.Invoke.Command != "update" {
					continue
				}
				queryBuf := instr.Invoke.Args.Search("query")
				if queryBuf == nil {
					continue
				}
				query := &Query{}
				if err := protobuf.Decode(queryBuf, query); err != nil {
					handler(Query{}, nil, errors.New("could not decode the query "+err.Error()))
					continue
				}
				handler(*query, sb.Hash, nil)
			}
		}
	}
	return nil
}
