// Package medchain is the client side API for communicating with medchain
// service.
package medchain

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"go.dedis.ch/cothority/v3"
	"go.dedis.ch/cothority/v3/byzcoin"
	"go.dedis.ch/cothority/v3/byzcoin/bcadmin/lib"
	"go.dedis.ch/cothority/v3/byzcoin/contracts"
	"go.dedis.ch/cothority/v3/darc"
	"go.dedis.ch/cothority/v3/darc/expression"
	"go.dedis.ch/cothority/v3/skipchain"
	"go.dedis.ch/kyber/v3"
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
	EntryPoint *network.ServerIdentity
	public     kyber.Point
	private    kyber.Scalar
	// Signers are the Darc signers that will sign transactions sent with this client.
	Signers []darc.Signer
	// Instance ID of naming contract
	NamingInstance byzcoin.InstanceID
	GenDarcID      darc.ID
	GenDarc        *darc.Darc
	// // Map projects to their darcs
	AllDarcs   map[string]*darc.Darc
	AllDarcIDs map[string]darc.ID
	// GMsg       *byzcoin.CreateGenesisBlock
	signerCtrs []uint64
}

// NewClient creates a new client to talk to the medchain service.
// Fields DarcID, Instance, and Signers must be filled in before use.
func NewClient(bcl *byzcoin.Client, entryPoint *network.ServerIdentity, clientID string, keys darc.Signer) (*Client, error) {
	//keys := key.NewKeyPair(TSuite)
	if bcl == nil {
		return nil, errors.New("Byzcoin client is required")
	}

	if keys.Type() == -1 {
		return nil, xerrors.New("empty key")

	}
	pub := keys.Ed25519.Point
	priv, err := keys.GetPrivate()
	if err != nil {
		return nil, xerrors.Errorf("error in getting private key: %v", err)
	}

	cl := &Client{
		Bcl:        bcl,
		onetcl:     onet.NewClient(cothority.Suite, ServiceName),
		sccl:       skipchain.NewClient(),
		ClientID:   clientID,
		EntryPoint: entryPoint,
		public:     pub,
		private:    priv,
		signerCtrs: nil,
	}

	cl.Signers = []darc.Signer{keys}
	gDarc, err := bcl.GetGenDarc()
	if err != nil {
		return nil, xerrors.Errorf("error in getting genesis darc from ByzCoin: %v", err)
	}

	for _, signer := range cl.Signers {
		resp, err := cl.Bcl.GetSignerCounters(signer.Identity().String())
		if err != nil {
			return nil, xerrors.Errorf("could not get signer counter: %v", err)
		}
		cl.signerCtrs = append(cl.signerCtrs, resp.Counters[0])
	}
	log.Info("[INFO] (NewClient) latest coutners are", (cl.signerCtrs))

	cl.GenDarc = gDarc
	cl.GenDarcID = gDarc.GetBaseID()
	log.Info("[INFO] (NewClient) Genesis darc:", gDarc.String())

	return cl, nil
}

// Create creates a new medchain by spawning an instance of Naming contract. After
// this method is executed, c.NamingInstance will be correctly set.
func (c *Client) Create() error {

	log.Info("[INFO] (API) Creating the MedChain client:")
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
	log.Info("[INFO] (API) Spawning the instance of naming contract")
	err = c.spawnTx(namingTx)
	if err != nil {
		xerrors.Errorf("Could not add naming contract instace to the ledger: %v", err)
	}
	log.Info("[INFO] (Create) contract_name instance was added to the ledger")
	log.Info("[INFO] (Create) Genesis Darc:", c.GenDarc)

	return nil
}

//SpawnDeferredQuery spawns a query as well as a deferred contract with medchain contract
func (c *Client) SpawnDeferredQuery(req *AddDeferredQueryRequest) (*AddDeferredQueryReply, error) {
	log.Info("[INFO] Spawning the deferred query ")

	if len(req.QueryID) == 0 {
		return nil, xerrors.New("query ID required")
	}

	if len(req.QueryInstID) == 0 {
		return nil, xerrors.New("query instance ID required")
	}

	if len(req.ClientID) == 0 {
		return nil, xerrors.New("ClientID required")
	}

	if len(req.DarcID) == 0 {
		return nil, xerrors.New("Darc ID required")
	}

	req.QueryStatus = []byte("Submitted")
	log.Info("[INFO] Spawning the deferred query ")
	return c.createDeferredInstance(req)
}

//createDeferredInstance spawns a query that
func (c *Client) createDeferredInstance(req *AddDeferredQueryRequest) (*AddDeferredQueryReply, error) {
	log.Info("[INFO] Spawning the deferred query with ID: ", req.QueryID)
	log.Info("[INFO] Spawning the deferred query with Status: ", req.QueryStatus)
	log.Info("[INFO] Spawning the deferred query with Status: ", string(req.QueryStatus))
	log.Info("[INFO] Spawning the deferred query with Darc ID: ", req.DarcID)
	query := Query{}
	query.ID = req.QueryID
	query.Status = req.QueryStatus
	// Perfomr darc sanity check
	_, err := lib.GetDarcByID(c.Bcl, req.DarcID)
	if err != nil {
		return nil, xerrors.Errorf(" error in retrieving darc : %v", err)
	}
	log.Info("[INFO] Spawning the deferred query with decoded query: ", query)
	proposedInstr := byzcoin.Instruction{
		InstanceID: byzcoin.NewInstanceID(req.DarcID),
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

	req.QueryInstID, err = c.spawnDeferredInstance(query, proposedTransactionBuf, req.DarcID, req.QueryInstID)
	if err != nil {
		return nil, xerrors.Errorf("could not spawn instance: %v", err)
	}
	req.BlockID = c.Bcl.ID
	reply := &AddDeferredQueryReply{}
	err = c.onetcl.SendProtobuf(c.EntryPoint, req, reply)
	if err != nil {
		return nil, xerrors.Errorf("could not get AddDeferredQueryReply from service: %v", err)
	}
	log.Info("[INFO] Successfully spawned the deferred query and now propating the instance ID: ", req.QueryInstID.String())

	sharingReq := &PropagateIDRequest{req.QueryInstID, &c.Bcl.Roster}
	sharingReply := &PropagateIDReply{}
	err = c.onetcl.SendProtobuf(c.EntryPoint, sharingReq, sharingReply)
	if err != nil {
		return nil, xerrors.Errorf("could not get PropagateIDReply from service: %v", err)
	}

	return reply, nil
}

// SpawnDeferredInstance spwans a deferred instance
func (c *Client) spawnDeferredInstance(query Query, proposedTransactionBuf []byte, darcID darc.ID, oldInstID byzcoin.InstanceID) (byzcoin.InstanceID, error) {

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
		return *new(byzcoin.InstanceID), xerrors.Errorf("could not create deferred transaction: %v", err)
	}
	err = c.spawnTx(ctx)
	if err != nil {
		return *new(byzcoin.InstanceID), xerrors.Errorf("could not add deferred transaction to ledger: %v", err)
	}
	instID := ctx.Instructions[0].DeriveID("")
	return instID, err
}

// AuthorizeQuery checks authorizations of the query
func (c *Client) AuthorizeQuery(req *AuthorizeQueryRequest) (*AuthorizeQueryReply, error) {
	log.Info("[INFO] Authorization of query ")

	if len(req.QueryID) == 0 {
		return nil, xerrors.New("query ID required")
	}

	if len(req.QueryInstID) == 0 {
		return nil, xerrors.New("query instance ID required")
	}

	req.QueryStatus = []byte("Submitted")
	log.Info("[INFO] Checking the authorizations for the query with instance ID: ", req.QueryInstID)
	log.Info("[INFO] Checking the authorizations for the query with query ID: ", req.QueryID)
	return c.createQueryAndWait(req)
}

// createQueryAndWait sends a request to create a query and waits for N block intervals
// that the queries are added to the ledger
func (c *Client) createQueryAndWait(req *AuthorizeQueryRequest) (*AuthorizeQueryReply, error) {
	if c.signerCtrs == nil {
		c.RefreshSignerCounters()
	}

	query := Query{}
	query.ID = req.QueryID
	query.Status = req.QueryStatus

	ctx, status, err := c.prepareTx(query, req.DarcID, req.QueryInstID)
	if err != nil {
		return nil, xerrors.Errorf("could not create transaction: %v", err)
	}
	err = c.spawnTx(ctx)
	if err != nil {
		return nil, xerrors.Errorf("could not add transaction to ledger: %v", err)
	}
	// update
	req.BlockID = c.Bcl.ID
	req.QueryStatus = status
	log.Info("[INFO] (AuthorizeQuery) Query was added to the ledger")
	log.Info("[INFO] (AuthorizeQuery) instance ID is:", req.QueryInstID)
	log.Info("[INFO] (AuthorizeQuery) Query status is:", string(req.QueryStatus))

	reply := &AuthorizeQueryReply{}
	err = c.onetcl.SendProtobuf(c.EntryPoint, req, reply)
	if err != nil {
		return nil, xerrors.Errorf("could not get AuthorizeQueryReply from service: %v", err)
	}
	reply.QueryInstID = req.QueryInstID
	log.Info("[INFO] (AuthorizeQuery) InstanceID received from service is:", reply.QueryInstID.String())
	log.Info("[INFO] (AuthorizeQuery) reply.OK received from service is:", reply.OK)

	return reply, nil
}

// prepareTx prepares a transaction that will be committed to the ledger.
func (c *Client) prepareTx(query Query, darcID darc.ID, instID byzcoin.InstanceID) (byzcoin.ClientTransaction, []byte, error) {

	ok := true
	var status []byte
	var args byzcoin.Argument

	// Get the project darc
	action := c.getActionFromOneQuery(query)

	// Check if the query is authorized/rejected
	authorization, err := c.checkAuth(query, c.Signers[0], darcID, action)
	if err != nil {
		return *new(byzcoin.ClientTransaction), nil, err
	}
	res := authorization
	if res == false {
		ok = false //reject the query as at least one of the signers can't sign
		args = byzcoin.Argument{
			Name:  query.ID,
			Value: []byte("Rejected"),
		}
		status = []byte("Rejected")
		log.Info("[INFO] (Invoke) Query was REJECTED")
	}

	if ok {
		args = byzcoin.Argument{
			Name:  query.ID,
			Value: []byte("Authorized"),
		}
		status = []byte("Authorized")
		action = c.getActionFromOneQuery(query)
		log.Info("[INFO] (Invoke) Query was AUTHORIZED")
	}

	instr := byzcoin.Instruction{
		InstanceID: instID,
		Invoke: &byzcoin.Invoke{
			ContractID: contracts.ContractValueID,
			Command:    "update",
			Args:       []byzcoin.Argument{args},
		},
		SignerCounter: c.IncrementCtrs(),
	}

	ctx, err := c.Bcl.CreateTransaction(instr)
	if err != nil {
		return *new(byzcoin.ClientTransaction), nil, err
	}

	return ctx, status, nil
}

// checkAuth checks authorizations for the query
func (c *Client) checkAuth(query Query, signer darc.Signer, darcID darc.ID, action string) (bool, error) {
	var auth bool
	auth = false
	// We need the identity part of the signatures before
	// calling ToDarcRequest() below, because the identities
	// go into the message digest.
	ddarc, err := lib.GetDarcByID(c.Bcl, darcID)
	if err != nil {
		return false, xerrors.Errorf(" error in retrieving darc : %v", err)
	}
	log.Info("[INFO] (checkAuth) darc:", ddarc.String())

	dAction := darc.Action("invoke:medchain." + action)
	exists := ddarc.Rules.Contains(dAction)
	if !exists {
		log.Info("[INFO] (checkAuth) Darc action does not exist")
		return false, nil
	}
	log.Info("[INFO] (checkAuth) checking actions")
	for _, r := range ddarc.Rules.List {
		if r.Action == dAction {
			ruleStr := r.String()
			idExists := strings.Contains(ruleStr, signer.Identity().String())
			if idExists {
				log.Info("[INFO] (checkAuth) ID existance in darc rule:", idExists)
				auth = true
			}

		}
	}
	return auth, nil
}

//SpawnQuery spawns a query instance
func (c *Client) SpawnQuery(req *AddQueryRequest) (*AddQueryReply, error) {
	log.Info("[INFO] (SpawnQuery) Spawning the query ")
	ddarc, err := lib.GetDarcByID(c.Bcl, req.DarcID)
	if err != nil {
		return nil, xerrors.Errorf("error in getting darc from ByzCoin: %v", err)
	}
	log.Info("[INFO] (SpawnQuery) darcID,", ddarc.GetIdentityString())

	if len(req.QueryID) == 0 {
		return nil, xerrors.New("query ID required")
	}

	if len(req.ClientID) == 0 {
		return nil, xerrors.New("ClientID required")
	}
	if len(req.DarcID) == 0 {
		return nil, xerrors.New("Darc ID required")
	}

	req.QueryStatus = []byte("Submitted")
	log.Info("[INFO] Spawning the deferred query ")

	return c.createInstance(req)
}

//CreateInstance spawns a query
func (c *Client) createInstance(req *AddQueryRequest) (*AddQueryReply, error) {

	log.Info("[INFO] (SpawnQuery) Spawning the query with ID: ", req.QueryID)
	log.Info("[INFO] (SpawnQuery)Spawning the query with Status: ", string(req.QueryStatus))

	query := Query{}
	query.ID = req.QueryID
	query.Status = req.QueryStatus
	// Perfomr darc sanity check
	_, err := lib.GetDarcByID(c.Bcl, req.DarcID)
	if err != nil {
		return nil, xerrors.Errorf(" error in retrieving darc : %v", err)
	}
	log.Infof("[INFO] (SpawnQuery) Spawning the query %v with value contract", query)
	instr := byzcoin.Instruction{
		InstanceID: byzcoin.NewInstanceID(req.DarcID),
		Spawn: &byzcoin.Spawn{
			ContractID: contracts.ContractValueID,
			Args: byzcoin.Arguments{
				{
					Name:  req.QueryID,
					Value: []byte(req.QueryStatus),
				},
			},
		},
		SignerCounter: c.IncrementCtrs(),
	}
	ctx, err := c.Bcl.CreateTransaction(instr)
	if err != nil {
		return nil, xerrors.Errorf("could not create value transaction: %v", err)
	}
	err = c.spawnTx(ctx)
	if err != nil {
		return nil, xerrors.Errorf("could not add value transaction to ledger: %v", err)
	}
	instID := ctx.Instructions[0].DeriveID("")

	req.QueryInstID = instID
	req.BlockID = c.Bcl.ID
	req.QueryInstID = instID
	reply := &AddQueryReply{}
	err = c.onetcl.SendProtobuf(c.EntryPoint, req, reply)
	if err != nil {
		return nil, xerrors.Errorf("could not get AddQueryReply from service: %v", err)
	}
	if !reply.QueryInstID.Equal(req.QueryInstID) {
		return nil, xerrors.Errorf("invalid AddQueryReply")
	}
	// Name the instance of the query with as its key using contract_name to
	// make retrievals easier
	err = c.nameInstance(req.QueryInstID, req.DarcID, query.ID)
	if err != nil {
		return nil, xerrors.Errorf("could not name the instance: %v", err)
	}

	if reply.OK == true {
		log.Infof("[INFO] (SpawnQuery) Successfully spawned query with instance ID %v and now will check its authorizations:", reply.QueryInstID.String())
		req2 := &AuthorizeQueryRequest{}
		req2.QueryID = req.QueryID
		req2.QueryStatus = req.QueryStatus
		req2.QueryInstID = req.QueryInstID
		req2.DarcID = req.DarcID //TODO: double-check
		req.BlockID = c.Bcl.ID
		reply2, err := c.AuthorizeQuery(req2)
		if err != nil {
			return reply, xerrors.Errorf("query authorization failed: %v", err)
		}
		if reply2.OK != true {
			return reply, xerrors.Errorf("reply not ok: query authorization failedquery authorization failed")
		}
		if req.QueryInstID != req2.QueryInstID {
			return reply, xerrors.Errorf("instance id not ok: query authorization failed")
		}
		reply2.QueryInstID = reply.QueryInstID

		if reply2.OK {
			log.Infof("[INFO] (SpawnQuery) Successfully authorized query with instance ID %v :", reply.QueryInstID.String())

			log.Info("[INFO] (SpawnQuery) Query status now is: ", string(reply2.QueryStatus))
			reply.QueryStatus = reply2.QueryStatus
			if string(reply2.QueryStatus) == "Authorized" {
				log.Info("[INFO] (SpawnQuery) spawning deferred query")
				req3 := &AddDeferredQueryRequest{}
				query := NewQuery(req2.QueryID, string(reply2.QueryStatus))
				req3.QueryInstID = req2.QueryInstID
				req3.QueryID = query.ID
				req3.QueryStatus = query.Status
				req3.ClientID = c.ClientID
				req3.DarcID = req.DarcID
				req3.BlockID = c.Bcl.ID
				reply3, err := c.SpawnDeferredQuery(req3)
				if err != nil {
					return nil, xerrors.Errorf("could not spawn deferred query: %v", err)
				}
				if reply3.OK != true {
					return nil, xerrors.Errorf("could not spawn deferred query: %v", err)
				}
				if reply3.QueryInstID.Equal(req2.QueryInstID) {
					return nil, xerrors.Errorf("invalid instance ID returned by AddeferredQueryReply")
				}
				reply.QueryInstID = reply3.QueryInstID
			}

		}

	}

	return reply, nil
}

// AddSignatureToDeferredQuery allows MedChain user to sign a deferred query
// by invoking an addProof action from the deferred contract on the deferred
// query instance
func (c *Client) AddSignatureToDeferredQuery(req *SignDeferredTxRequest) (*SignDeferredTxReply, error) {
	log.Info("[INFO] (AddSignatureToDeferredQuery) Add signature to the query transaction")
	log.Info("[INFO] (AddSignatureToDeferredQuery) length of signers", len(c.Signers))
	log.Info("[INFO] (AddSignatureToDeferredQuery) coutners", (c.signerCtrs))
	if req.Keys.Type() == -1 {
		return nil, errors.New("client keys are required")
	}
	if len(req.ClientID) == 0 {
		return nil, errors.New("ClientID required")
	}

	if len(req.QueryInstID) == 0 {
		return nil, errors.New("query instance ID required")
	}
	result, err := c.Bcl.GetDeferredData(req.QueryInstID)
	log.Info("[INFO] (AddSignatureToDeferredQuery) retrieved deferred data:", (result))
	if err != nil {
		return nil, xerrors.Errorf("failed to get deffered instance from skipchain: %v", err)
	}
	rootHash := result.InstructionHashes
	// signer := c.Signers[0]
	identity := req.Keys.Identity() // TODO: Sign with private key of client
	identityBuf, err := protobuf.Encode(&identity)
	if err != nil {
		return nil, xerrors.Errorf("could not get the user identity: %v", err)
	}
	signature, err := req.Keys.Sign(rootHash[0]) // == index
	if err != nil {
		return nil, xerrors.Errorf("could not sign the deffered query: %v", err)
	}
	index := uint32(0) // The index of the instruction to sign in the transaction
	indexBuf := make([]byte, 4)
	binary.LittleEndian.PutUint32(indexBuf, uint32(index))

	log.Info("[INFO] (AddSignatureToDeferredQuery) creating addProof tx")
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
		return nil, xerrors.Errorf("failed to create the transaction: %v", err)
	}
	log.Info("[INFO] (AddSignatureToDeferredQuery) length of signers", len(c.Signers))
	log.Info("[INFO] (AddSignatureToDeferredQuery) coutners", (c.signerCtrs))
	err = c.spawnTx(ctx)
	if err != nil {
		return nil, xerrors.Errorf("failed to sign the deferred transaction: %v", err)
	}

	reply := &SignDeferredTxReply{}
	err = c.onetcl.SendProtobuf(c.EntryPoint, req, reply)
	if err != nil {
		return nil, xerrors.Errorf("could not get reply from service: %v", err)
	}
	return reply, nil
}

// ExecDefferedQuery executes the query that has received enough signatures
func (c *Client) ExecDefferedQuery(req *ExecuteDeferredTxRequest) (*ExecuteDeferredTxReply, error) {
	log.Info("[INFO] (ExecDefferedQuery) Execute the query transaction")
	if len(req.ClientID) == 0 {
		return nil, xerrors.New("ClientID required")
	}

	if len(req.QueryInstID) == 0 {
		return nil, xerrors.New("query instance ID required")
	}

	ctx, err := c.Bcl.CreateTransaction(byzcoin.Instruction{
		InstanceID: req.QueryInstID,
		Invoke: &byzcoin.Invoke{
			ContractID: byzcoin.ContractDeferredID,
			Command:    "execProposedTx",
		},
		SignerCounter: c.IncrementCtrs(),
	})
	if err != nil {
		return nil, xerrors.Errorf("failed to execute transaction: %v", err)
	}

	err = c.spawnTx(ctx)
	if err != nil {
		return nil, xerrors.Errorf("failed to execute transaction: %v", err)
	}

	reply := &ExecuteDeferredTxReply{}
	err = c.onetcl.SendProtobuf(c.EntryPoint, req, reply)
	if err != nil {
		return nil, xerrors.Errorf("could not get ExecuteDeferredTxReply from service: %v", err)
	}

	if reply.OK != true {
		return reply, xerrors.Errorf("reply not ok: query execution failed")
	}

	if !reply.QueryInstID.Equal(req.QueryInstID) {
		return reply, xerrors.Errorf("invalid instance ID returned by ExecuteDeferredTxReply ")
	}

	return reply, nil
}

// GetDarcRules returns the rules for signers of a query transaction in the project darc
func (c *Client) GetDarcRules(instID byzcoin.InstanceID) error {
	log.Info("[INFO] Checking the signer rules for instance ID", instID)
	instIDBuf, err := hex.DecodeString(instID.String())
	if err != nil {
		return xerrors.Errorf("failed to decode the instid string: %v", err)
	}
	pr, err := c.Bcl.GetProofFromLatest(instIDBuf)
	proof := pr.Proof
	_, _, _, darcID, err := proof.KeyValue()
	darc, err := lib.GetDarcByID(c.Bcl, darcID)
	if err != nil {
		return xerrors.Errorf(" error in retrieving darc : %v", err)
	}
	rules := darc.Rules.List
	for i := range rules {
		fmt.Println(rules[i].String())
	}

	return nil
}

// VerifStatus retrieves the status of the query from skipchain
func (c *Client) VerifStatus(req *VerifyStatusRequest) (*VerifyStatusReply, error) {
	reply := &VerifyStatusReply{}
	if err := c.onetcl.SendProtobuf(c.EntryPoint, req, reply); err != nil {
		return nil, err
	}
	return reply, nil
}

// AddSignerToDarc adds new signer to project darc
// TODO: make this a defferred tx (not important for this part of project;
// it is important for th eadmin part)
func (c *Client) AddSignerToDarc(name string, darcID darc.ID, darcActions []darc.Action, newSigner string, typeStr string) error {

	var typeOfExpr int
	var exp expression.Expr
	if typeStr != "AND" && typeStr != "OR" {
		return xerrors.Errorf(" invalid rule entered")
	}
	if typeStr == "AND" {
		typeOfExpr = 0
	} else {
		typeOfExpr = 1
	}

	projectDarc, err := lib.GetDarcByID(c.Bcl, darcID)
	if err != nil {
		return xerrors.Errorf(" error in retrieving darc : %v", err)
	}
	dAction := darc.Action("spawn:deferred")
	for i, r := range projectDarc.Rules.List {
		if r.Action == dAction {
			exp = projectDarc.Rules.List[i].Expr
		}
	}
	log.Info("[INFO] Expression used to add signers", string(exp))
	signerIDs := strings.Split(string(exp), "|")
	for i := range signerIDs {
		signerIDs[i] = strings.TrimSpace(signerIDs[i])
	}
	signerIDs = append(signerIDs, newSigner)
	log.Info("[INFO] (AddSignerToDarc) signerIDs:", signerIDs)

	ctx, newDarcID, err := c.EvolveProjectDarc(signerIDs, projectDarc, darcActions, typeOfExpr)
	if err != nil {
		return xerrors.Errorf("error in evolving the project darc: %v", err)
	}
	err = c.spawnTx(ctx)
	if err != nil {
		return xerrors.Errorf("error in evolving the project darc: %v", err)
	}
	// update the darc and darc ID of cilent
	newDarc, err := lib.GetDarcByID(c.Bcl, newDarcID)
	if err != nil {
		return xerrors.Errorf("error in retrieving evolved darc : %v", err)
	}
	c.AllDarcIDs[name] = newDarcID
	c.AllDarcs[name] = newDarc
	log.Info("[INFO] Evolved darc is:", newDarc.String())
	return nil
}

// CreateNewSigner creates private and public key pairs
func (c *Client) CreateNewSigner(public kyber.Point, private kyber.Scalar) darc.Signer {
	identity := darc.NewSignerEd25519(public, private)
	return identity
}

// AddRuleToDarc adds action rules to the given darc
func (c *Client) AddRuleToDarc(userDarc *darc.Darc, action string, expr expression.Expr) (*darc.Darc, error) {
	actions := strings.Split(action, ",")

	for i := 0; i < len(actions); i++ {
		dAction := darc.Action(actions[i])
		userDarc.Rules.AddRule(dAction, expr)
	}
	return userDarc, nil
}

// UpdateDarcActionRule update action rules of the given darc
func (c *Client) UpdateDarcActionRule(userDarc *darc.Darc, action string, expr expression.Expr) *darc.Darc {
	actions := strings.Split(action, ",")

	for i := 0; i < len(actions); i++ {
		dAction := darc.Action(actions[i])
		userDarc.Rules.UpdateRule(dAction, expr)
	}
	return userDarc
}

// CreateDarc is used to create a new darc
func (c *Client) CreateDarc(name string, rules darc.Rules, actionsAnd string, actionsOr string, exprsAnd expression.Expr, exprsOr expression.Expr) (*darc.Darc, error) {
	projectDarc := darc.NewDarc(rules, []byte(name))
	projectDarc, err := c.AddRuleToDarc(projectDarc, actionsAnd, exprsAnd)
	if err != nil {
		return nil, xerrors.Errorf(" error in creating darc : %v", err)
	}
	projectDarc, err = c.AddRuleToDarc(projectDarc, actionsOr, exprsOr)
	if err != nil {
		return nil, xerrors.Errorf(" error in creating darc : %v", err)
	}
	return projectDarc, nil
}

// AddProjectDarc is used to create project darcs with default rules
func (c *Client) AddProjectDarc(name string) (*darc.Darc, error) {
	log.Info("[INFO] Adding project darc")

	rulesA := darc.InitRules([]darc.Identity{c.Signers[0].Identity()}, []darc.Identity{c.Signers[0].Identity()})
	actionsAAnd := "spawn:medchain,invoke:medchain.patient_list,invoke:medchain.count_per_site,invoke:medchain.count_per_site_obfuscated," +
		"invoke:medchain.count_per_site_shuffled,invoke:medchain.count_per_site_shuffled_obfuscated,invoke:medchain.count_global," +
		"invoke:medchain.count_global_obfuscated"

	actionsAOr := "spawn:deferred,invoke:medchain.update,invoke:deferred.addProof,invoke:deferred.execProposedTx,spawn:darc,invoke:darc.evolve,_name:deferred,spawn:naming,_name:medchain,spawn:value,invoke:value.update,_name:value,invoke:darc.evolve_unrestricted"

	// all signers need to sign
	exprAAnd := expression.InitAndExpr(c.Signers[0].Identity().String())

	// at least one signer needs to sign
	var err error
	exprAOr := expression.InitOrExpr(c.Signers[0].Identity().String())
	c.AllDarcs[name], err = c.CreateDarc(name, rulesA, actionsAAnd, actionsAOr, exprAAnd, exprAOr)
	if err != nil {
		return nil, xerrors.Errorf("could not create project darc: %v", err)
	}

	// Verify the darc is correct
	err = c.AllDarcs[name].Verify(true)
	if err != nil {
		return nil, xerrors.Errorf("could not create project darc: %v", err)
	}

	aDarcBuf, err := c.AllDarcs[name].ToProto()
	if err != nil {
		return nil, xerrors.Errorf("could not create project darc: %v", err)
	}
	aDarcCopy, err := darc.NewFromProtobuf(aDarcBuf)
	if err != nil {
		return nil, xerrors.Errorf("could not create project darc: %v", err)
	}
	if c.AllDarcs[name].Equal(aDarcCopy) != true {
		return nil, xerrors.Errorf("could not create project darc")
	}

	// Add darc to byzcoin
	ctx, err := c.Bcl.CreateTransaction(byzcoin.Instruction{
		InstanceID: byzcoin.NewInstanceID(c.GenDarc.GetBaseID()),
		Spawn: &byzcoin.Spawn{
			ContractID: byzcoin.ContractDarcID,
			Args: byzcoin.Arguments{
				{
					Name:  "darc",
					Value: aDarcBuf,
				},
			},
		},
		SignerIdentities: []darc.Identity{c.Signers[0].Identity()},
		SignerCounter:    c.IncrementCtrs(),
	})
	if err != nil {
		return nil, xerrors.Errorf("could not create project darc: %v", err)
	}

	err = c.spawnTx(ctx)
	if err != nil {
		return nil, xerrors.Errorf("could not create project darc: %v", err)
	}
	c.AllDarcIDs[name] = c.AllDarcs[name].GetBaseID()
	log.Info("[INFO] Darc for Project is:", c.AllDarcs[name].String())

	return c.AllDarcs[name], nil
}

// UpdateGenesisDarc is used to update client genesis darc so that the user can start inracting with the server
func (c *Client) UpdateGenesisDarc(signers []string) error {

	log.Info("[INFO] Updating Genesis Darc")

	gDarc, err := c.Bcl.GetGenDarc()
	if err != nil {
		return xerrors.Errorf("error in getting genesis darc from ByzCoin: %v", err)
	}
	expr := expression.InitOrExpr(signers...)
	err = gDarc.Rules.AddRule("spawn:"+ContractName, expr)
	if err != nil {
		return xerrors.Errorf("could not add rule to genesis darc: %v", err)
	}
	err = gDarc.Rules.AddRule("invoke:"+ContractName+"."+"update", expr)
	if err != nil {
		return xerrors.Errorf("could not add rule to genesis darc: %v", err)
	}
	err = gDarc.Rules.AddRule("invoke:"+ContractName+"."+"verifystatus", expr)
	if err != nil {
		return xerrors.Errorf("could not add rule to genesis darc: %v", err)
	}
	err = gDarc.Rules.AddRule("_name:"+ContractName, expr)
	if err != nil {
		return xerrors.Errorf("could not add rule to genesis darc: %v", err)
	}
	err = gDarc.Rules.AddRule("spawn:deferred", expr)
	if err != nil {
		return xerrors.Errorf("could not add rule to genesis darc: %v", err)
	}
	err = gDarc.Rules.AddRule("invoke:deferred.addProof", expr)
	if err != nil {
		return xerrors.Errorf("could not add rule to genesis darc: %v", err)
	}
	err = gDarc.Rules.AddRule("invoke:deferred.execProposedTx", expr)
	if err != nil {
		return xerrors.Errorf("could not add rule to genesis darc: %v", err)
	}
	err = gDarc.Rules.AddRule("spawn:value", expr)
	if err != nil {
		return xerrors.Errorf("could not add rule to genesis darc: %v", err)
	}
	err = gDarc.Rules.AddRule("invoke:value.update", expr)
	if err != nil {
		return xerrors.Errorf("could not add rule to genesis darc: %v", err)
	}
	err = gDarc.Rules.AddRule("_name:value", expr)
	if err != nil {
		return xerrors.Errorf("could not add rule to genesis darc: %v", err)
	}
	log.Infof("[INFO] Updated Genesis Darc with ID %v: ", gDarc.GetIdentityString())
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
		InstanceID: byzcoin.NewInstanceID(c.GenDarcID),
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
		return xerrors.Errorf("Could not create transaction: %v", err)
	}
	err = c.spawnTx(ctx)
	if err != nil {
		xerrors.Errorf("Could not add transaction to ledger: %v", err)
	}
	c.AllDarcIDs[name] = c.AllDarcs[name].GetBaseID()

	return nil
}

// GetQuery asks the service to retrieve a query from the ledger by its key.
func (c *Client) GetQuery(key []byte) (*QueryData, error) {
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
	q := QueryData{}
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

// EvolveProjectDarc is used to evolve a darc
func (c *Client) EvolveProjectDarc(signerIDs []string, olddarc *darc.Darc, darcActions []darc.Action, typeOfExpr int) (byzcoin.ClientTransaction, darc.ID, error) {
	log.Info("[INFO] (EvolveProjectDarc) signerIDs:", signerIDs)
	newdarc := olddarc.Copy()
	newExpr := []expression.Expr{expression.InitAndExpr(signerIDs...), expression.InitOrExpr(signerIDs...)}

	err := c.UpdateDarcSignerRule(newdarc, darcActions, newExpr, typeOfExpr)
	if err != nil {
		return byzcoin.ClientTransaction{}, nil, xerrors.Errorf("updating darc signer rules: %v", err)
	}
	err = newdarc.EvolveFrom(olddarc)
	if err != nil {
		return byzcoin.ClientTransaction{}, nil, xerrors.Errorf("evolving the project darc signer rule: %v", err)
	}
	_, darc2Buf, err := newdarc.MakeEvolveRequest(c.Signers...)
	if err != nil {
		return byzcoin.ClientTransaction{}, nil, xerrors.Errorf("evolving the project darc signer rule: %v", err)
	}

	ctx, err := c.Bcl.CreateTransaction(byzcoin.Instruction{
		InstanceID: byzcoin.NewInstanceID(olddarc.GetBaseID()),
		Invoke: &byzcoin.Invoke{
			ContractID: byzcoin.ContractDarcID,
			Command:    "evolve",
			Args: []byzcoin.Argument{{
				Name:  "darc",
				Value: darc2Buf,
			}},
		},
		SignerCounter: c.IncrementCtrs(),
	})
	if err != nil {
		return byzcoin.ClientTransaction{}, nil, xerrors.Errorf("creating the transaction: %v", err)
	}
	return ctx, newdarc.GetBaseID(), nil
}

// UpdateDarcSignerRule updates the rules in project darc
func (c *Client) UpdateDarcSignerRule(evolvedDarc *darc.Darc, darcActions []darc.Action, newSignerExpr []expression.Expr, typeOfExpr int) error {

	for _, action := range darcActions {
		if len(action) == 0 {
			return xerrors.Errorf("error in updating the project darc:action '%v' does not exist", action)
		}

		// typeOfExpr defines the type or rule to use; 0:& and 1:|
		err := evolvedDarc.Rules.UpdateRule(action, newSignerExpr[typeOfExpr])
		if err != nil {
			return xerrors.Errorf("updating the %s expression in darc: %v", action, err)
		}
	}
	return nil
}

// UpdateDarcSignerSignRule updates the _sign in project darc
func (c *Client) UpdateDarcSignerSignRule(evolvedDarc *darc.Darc, darcActions []darc.Action, newSignerExpr []expression.Expr, typeOfExpr int) error {
	err := evolvedDarc.Rules.UpdateSign(newSignerExpr[0])
	if err != nil {
		return xerrors.Errorf("updating the _sign rule in darc: %v", err)
	}

	return nil
}

// UpdateDarcSignerEvolveRule updates the _evolve rule in project darc
func (c *Client) UpdateDarcSignerEvolveRule(evolvedDarc *darc.Darc, darcActions []darc.Action, newSignerExpr []expression.Expr, typeOfExpr int) error {
	err := evolvedDarc.Rules.UpdateEvolution(newSignerExpr[0])
	if err != nil {
		return xerrors.Errorf("updating _evolve rule in darc: %v", err)
	}
	return nil
}

// Search executes a search on the filter in req. See the definition of type
// SearchRequest for additional details about how the filter is interpreted.
// The ID and Instance fields of the SearchRequest will be filled in from c.
func (c *Client) Search(req *SearchRequest) (*SearchReply, error) {
	req.BlockID = c.Bcl.ID

	reply := &SearchReply{}
	if err := c.onetcl.SendProtobuf(c.Bcl.Roster.List[0], req, reply); err != nil {
		return nil, err
	}
	return reply, nil
}

// GetSharedData retreives the new Instance ID saved at nodes
func (c *Client) GetSharedData() (*GetSharedDataReply, error) {
	rep := &GetSharedDataReply{}
	err := c.onetcl.SendProtobuf(c.EntryPoint, &GetSharedDataRequest{}, rep)
	if err != nil {
		return rep, xerrors.Errorf("could not send the GetSharedDataRequest to the service : %v", err)
	}
	return rep, nil
}

func (c *Client) getDarcActions(actions string) ([]darc.Action, error) {
	if len(actions) == 0 {
		return nil, xerrors.Errorf("error in getting actions: invalid actions", actions)
	}
	actoinsList := strings.Split(string(actions), ",")
	darcActions := make([]darc.Action, len(actoinsList))
	for i, action := range actoinsList {
		if len(action) == 0 {
			return nil, xerrors.Errorf("error in getting actions: action %v does not exist", action)
		}
		darcActions[i] = darc.Action(action)
	}
	return darcActions, nil
}

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
func (c *Client) getProjectDarcFromOneQuery(query Query) string {
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
	log.Info("[INFO] (nameInstance) naming instance ID:", instID.String())

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
		return xerrors.Errorf("Could not create transaction: %v", err)
	}
	err = c.spawnTx(ctx)
	if err != nil {
		return xerrors.Errorf("Could not add transaction to ledger: %v", err)
	}
	c.Bcl.WaitPropagation(1)
	log.Info("[INFO] (Naming) Query was added to the ledger")

	return nil
}

func (c *Client) spawnTx(ctx byzcoin.ClientTransaction) error {
	log.Info("[INFO] (spawnTx) length of signers", len(c.Signers))
	log.Info("[INFO] (spawnTx) Counters", (c.signerCtrs))
	err := ctx.FillSignersAndSignWith(c.Signers...)
	if err != nil {
		return xerrors.Errorf("error in signing the tx: %v", err)
	}
	_, err = c.Bcl.AddTransactionAndWait(ctx, 10)
	if err != nil {
		return xerrors.Errorf("error in adding transaction to the ledger: %v", err)
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

// SyncSignerCtrs syncs counters among clients
func (c *Client) SyncSignerCtrs(signers ...darc.Signer) {
	log.Info("[INFO] (SyncSignerCtrs) Sync signer counters")
	c.signerCtrs, _ = c.GetLatestSignerCtrs(signers...)
	//c.IncrementCtrs()
	log.Info("[INFO] (SyncSignerCtrs) latest coutners are", (c.signerCtrs))
}

// GetLatestSignerCtrs updates the coutnters using byzcoin retrieval
func (c *Client) GetLatestSignerCtrs(signers ...darc.Signer) ([]uint64, error) {
	signerCtrs := make([]uint64, len(c.signerCtrs))
	for i, signer := range signers {
		resp, err := c.Bcl.GetSignerCounters(signer.Identity().String())
		if err != nil {
			return nil, xerrors.Errorf("could not get signer counter: %v", err)
		}
		signerCtrs[i] = resp.Counters[0]
	}

	return signerCtrs, nil
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
