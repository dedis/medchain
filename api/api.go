package api

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

	"go.dedis.ch/cothority/v3"
	"go.dedis.ch/cothority/v3/byzcoin"
	"go.dedis.ch/cothority/v3/darc"
	"go.dedis.ch/cothority/v3/darc/expression"
	"go.dedis.ch/cothority/v3/skipchain"
	"go.dedis.ch/kyber/v3"
	"go.dedis.ch/onet/v3"
	"go.dedis.ch/onet/v3/log"
	"go.dedis.ch/onet/v3/network"
	"go.dedis.ch/protobuf"
)

// Client is a structure to communicate with Medchain service
type Client struct {
	ByzCoin *byzcoin.Client
	// The DarcID with "invoke:queryContract.update" & "invoke:queryContract.verifystatus "permission on it.
	DarcID darc.ID
	// Signers are the Darc signers that will sign transactions sent with this client.
	Signers    []darc.Signer
	Instance   byzcoin.InstanceID
	c          *onet.Client
	sc         *skipchain.Client
	signerCtrs []uint64
	genDarc    *darc.Darc
	aDarc      *darc.Darc // project A darc
	bDarc      *darc.Darc //project B darc
	gMsg       *byzcoin.CreateGenesisBlock
}

// NewClient creates a new client to talk to the medchain service.
// Fields DarcID, Instance, and Signers must be filled in before use.
func NewClient(ol *byzcoin.Client) *Client {
	return &Client{
		ByzCoin:    ol,
		c:          onet.NewClient(cothority.Suite, ServiceName),
		sc:         skipchain.NewClient(),
		signerCtrs: nil,
	}
}

// Create creates a new medchain. This method is synchronous: it will only
// return once the new query has been committed into the ledger (or after
// a timeout). Upon non-error return, c.Instance will be correctly set.
func (c *Client) Create() error {
	if c.signerCtrs == nil {
		c.RefreshSignerCounters()
	}
	fmt.Println(len(c.Signers))
	fmt.Println(c.Signers[0])
	// define darc contracts for project
	rulesA := darc.InitRules([]darc.Identity{c.Signers[0].Identity()}, []darc.Identity{c.Signers[0].Identity()})
	actionsA := "spawn:queryContract,invoke:queryContract.update,databaseA.patient_list,databaseA.count_per_site,databaseA.count_per_site_obfuscated," +
		"databaseA.count_per_site_shuffled,databaseA.count_per_site_shuffled_obfuscated,databaseA.count_global," +
		"databaseA.count_global_obfuscated"
	exprA := expression.InitOrExpr(c.Signers[0].Identity().String())
	c.aDarc, _ = c.EvolveDarc(c.genDarc, rulesA, "Project A Darc", c.Signers[0])
	c.aDarc = AddRuleToDarc(c.aDarc, actionsA, exprA)

	// Verify the darc is correct
	//require.Nil(t, c.aDarc.Verify(true))

	fmt.Println("**************** Genesis darc ******************")
	fmt.Println(c.genDarc.String())
	// Print darcs for project A and B
	fmt.Println("**************** Darc of Project A ******************")
	fmt.Println(c.aDarc.String())

	//sets the options so that only the given node will be contacted
	c.ByzCoin.UseNode(0)
	defer c.Close()

	// Signer 1 sends a query to project A

	args := byzcoin.Arguments{
		{
			Name:  "q1",                //+ c.Signers[0].Identity().String(),
			Value: []byte("Submitted"), //string representation of the darc ID,
		},
	}

	instr := byzcoin.Instruction{
		InstanceID: byzcoin.NewInstanceID(c.DarcID),
		Spawn: &byzcoin.Spawn{
			ContractID: MedchainContractID,
			Args:       args},
		SignerCounter: c.nextCtrs(),
	}

	tx, err := c.ByzCoin.CreateTransaction(instr)
	if err != nil {
		return err
	}
	//require.NoError(t, err)
	err = tx.FillSignersAndSignWith(c.Signers...)
	// if err := tx.FillSignersAndSignWith(c.Signers...); err != nil {
	// 	return err
	// }
	//require.NoError(t, err)
	// if _, err := c.ByzCoin.AddTransactionAndWait(tx, 10); err != nil {
	// 	return err
	// }
	_, err = c.ByzCoin.AddTransactionAndWait(tx, 10)
	//require.NoError(t, err)
	c.incrementCtrs()
	c.Instance = tx.Instructions[0].DeriveID("")
	return nil
}

// RefreshSignerCounters talks to the service to get the latest signer
// counters, the client should call this function if the internal counters
// become de-synchronised.
func (c *Client) RefreshSignerCounters() {
	signerIDs := make([]string, len(c.Signers))
	for i := range c.Signers {
		signerIDs[i] = c.Signers[i].Identity().String()
	}
	signerCtrs, err := c.ByzCoin.GetSignerCounters(signerIDs...)
	if err != nil {
		log.Error(err)
		return
	}
	c.signerCtrs = signerCtrs.Counters
}

// incrementCtrs will update the client state
func (c *Client) incrementCtrs() []uint64 {
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

// A QueryKey is an opaque unique identifier useful to find a given query later
// via GetQuery.
type QueryKey []byte

// WriteQueries asks the service to write queries to the ledger.
func (c *Client) WriteQueries(qu ...Query) ([]QueryKey, error) {
	return c.CreateQueryAndWait(10, qu...)
}

// CreateQueryAndWait sends a request to create a query and waits for N block intervals
// that the queries are added to the ledger
func (c *Client) CreateQueryAndWait(numInterval int, qu ...Query) ([]QueryKey, error) {
	if c.signerCtrs == nil {
		c.RefreshSignerCounters()
	}

	tx, keys, err := c.prepareTx(qu)
	if err != nil {
		return nil, err
	}
	if _, err := c.ByzCoin.AddTransactionAndWait(*tx, numInterval); err != nil {
		return nil, err
	}
	return keys, nil
}

// GetQuery asks the service to retrieve a query from the ledger by its key.
func (c *Client) GetQuery(key []byte) (*Query, error) {
	// reply, err := c.ByzCoin.GetProof(key)
	// if err != nil {
	// 	return nil, err
	// }
	// if !reply.Proof.InclusionProof.Match(key) {
	// 	return nil, errors.New("not an inclusion proof")
	// }
	// k, v0, _, _, err := reply.Proof.KeyValue()
	// if err != nil {
	// 	return nil, err
	// }
	// if !bytes.Equal(k, key) {
	// 	return nil, errors.New("wrong key")
	// }
	// q := Query{}
	// err = protobuf.Decode(v0, &q)
	// if err != nil {
	// 	return nil, err
	// }
	// return &q, nil

	// Get the proof from byzcoin
	reply, err := c.ByzCoin.GetProof(key)
	if err != nil {
		return nil, err
	}
	// Make sure the proof is a matching proof and not a proof of absence.
	pr := reply.Proof

	// Get the raw values of the proof.
	_, val, _, _, err := pr.KeyValue()
	if err != nil {
		return nil, err
	}
	// And decode the buffer to a Query struct
	cs := Query{}
	err = protobuf.Decode(val, &cs)
	fmt.Println("*** *** *** *** ***")
	fmt.Println(&cs)
	return &cs, nil
}

// prepareTx prepares a transaction that will be committed to the ledger.
func (c *Client) prepareTx(queries []Query) (*byzcoin.ClientTransaction, []QueryKey, error) {
	// We need the identity part of the signatures before
	// calling ToDarcRequest() below, because the identities
	// go into the message digest.
	sigs := make([]darc.Signature, len(c.Signers))
	for i, x := range c.Signers {
		sigs[i].Signer = x.Identity()
	}

	keys := make([]QueryKey, len(queries))

	instrs := make([]byzcoin.Instruction, len(queries))
	for i, id := range queries {
		queryBuf, err := protobuf.Encode(&id)
		if err != nil {
			return nil, nil, err
		}
		args := byzcoin.Argument{
			Name:  id.ID,    //TODO:add the name of queries
			Value: queryBuf, //put the correct value
		}
		instrs[i] = byzcoin.Instruction{
			InstanceID: c.Instance,
			Invoke: &byzcoin.Invoke{
				ContractID: MedchainContractID,
				Command:    "update",
				Args:       []byzcoin.Argument{args},
			},
			SignerCounter: c.incrementCtrs(),
		}
	}
	tx, err := c.ByzCoin.CreateTransaction(instrs...)
	if err != nil {
		return nil, nil, err
	}
	if err := tx.FillSignersAndSignWith(c.Signers...); err != nil {
		return nil, nil, err
	}
	for i := range tx.Instructions {
		keys[i] = QueryKey(tx.Instructions[i].DeriveID("").Slice())
	}
	return &tx, keys, nil
}

// CreateNewSigner creates private and public key pairs
func CreateNewSigner(public kyber.Point, private kyber.Scalar) darc.Signer {
	identity := darc.NewSignerEd25519(public, private)
	return identity
}

// AddRuleToDarc adds action rules to the given darc
func AddRuleToDarc(userDarc *darc.Darc, action string, expr expression.Expr) *darc.Darc {
	actions := strings.Split(action, ",")

	for i := 0; i < len(actions); i++ {
		dAction := darc.Action(actions[i])
		//fmt.Println(dAction)
		userDarc.Rules.AddRule(dAction, expr)
	}
	return userDarc
}

// UpdateDarcRule update action rules of the given darc
func UpdateDarcRule(userDarc *darc.Darc, action string, expr expression.Expr) *darc.Darc {
	actions := strings.Split(action, ",")

	for i := 0; i < len(actions); i++ {
		dAction := darc.Action(actions[i])
		//fmt.Println(dAction)
		userDarc.Rules.UpdateRule(dAction, expr)
	}
	return userDarc
}

// CreateProjectDarc creates darcs for projects (i.e., databases) given the rules, actions, and
// expressions.
func CreateProjectDarc(desc string, rules darc.Rules, actions string, expr expression.Expr) *darc.Darc {
	userDarc := darc.NewDarc(rules, []byte(desc))
	userDarc = AddRuleToDarc(userDarc, actions, expr)
	return userDarc

}

// StreamHandler is the signature of the handler used when streaming queries.
type StreamHandler func(query Query, blockID []byte, err error)

// Close closes all the websocket connections.
func (c *Client) Close() error {
	err := c.ByzCoin.Close()
	if err2 := c.sc.Close(); err2 != nil {
		err = err2
	}
	if err2 := c.c.Close(); err2 != nil {
		err = err2
	}
	return err
}

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
	return c.ByzCoin.StreamTransactions(h)
}

// StreamQueriesFrom is a blocking call where it calls the handler on even new
// event from (inclusive) the given block ID until the connection is closed or
// the server stops.
func (c *Client) StreamQueriesFrom(handler StreamHandler, id []byte) error {
	// 1. stream to a buffer (because we don't know which ones will be duplicates yet)
	blockChan := make(chan blockOrErr, 100)
	streamDone := make(chan error)
	go func() {
		err := c.ByzCoin.StreamTransactions(func(resp byzcoin.StreamingResponse, err error) {
			blockChan <- blockOrErr{resp.Block, err}
		})
		streamDone <- err
	}()

	// 2. use GetUpdateChain to find the missing events and call handler
	blocks, err := c.sc.GetUpdateChainLevel(&c.ByzCoin.Roster, id, 0, -1)
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

type blockOrErr struct {
	block *skipchain.SkipBlock
	err   error
}

// handleBlocks calls the handler on the events of the block
func handleBlocks(handler StreamHandler, sb *skipchain.SkipBlock) error {
	var err error
	var header byzcoin.DataHeader
	err = protobuf.DecodeWithConstructors(sb.Data, &header, network.DefaultConstructors(cothority.Suite))
	if err != nil {
		err = errors.New("could not unmarshal header while streaming events " + err.Error())
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

// EvolveDarc is used to evolve a darc
func (c *Client) EvolveDarc(d1 *darc.Darc, rules darc.Rules, name string, prevSigners ...darc.Signer) (*darc.Darc, error) {
	// Now the client wants to evolve the darc (change the owner), so it
	// creates a request and then sends it to the server.
	darcEvol := darc.NewDarc(rules, []byte(name))
	darcEvol.EvolveFrom(d1)
	r, d2Buf, err := darcEvol.MakeEvolveRequest(prevSigners...)
	fmt.Println(err)

	// Client sends request r and serialised darc d2Buf to the server, and
	// the server must verify it. Usually the server will look in its
	// database for the base ID of the darc in the request and find the
	// latest one. But in this case we assume it already knows. If the
	// verification is successful, then the server should add the darc in
	// the request to its database.
	fmt.Println(r.Verify(d1)) // Assume we can find d1 given r.
	d2Server, _ := r.MsgToDarc(d2Buf)
	fmt.Println(bytes.Equal(d2Server.GetID(), darcEvol.GetID()))

	// If the darcs stored on the server are trustworthy, then using
	// `Request.Verify` is enough. To do a complete verification,
	// Darc.Verify should be used. This will traverse the chain of
	// evolution and verify every evolution. However, the Darc.Path
	// attribute must be set.
	fmt.Println(d2Server.VerifyWithCB(func(s string, latest bool) *darc.Darc {
		if s == darc.NewIdentityDarc(d1.GetID()).String() {
			return d1
		}
		return nil
	}, true))
	return darcEvol, nil
}

// CreateDarc is used to create a new darc
func (c *Client) CreateDarc(d1 *darc.Darc, rules darc.Rules, name string, prevSigners ...darc.Signer) (*darc.Darc, error) {
	// Now the client wants to evolve the darc (change the owner), so it
	// creates a request and then sends it to the server.
	darcEvol := darc.NewDarc(rules, []byte(name))
	darcEvol.EvolveFrom(d1)
	r, d2Buf, err := darcEvol.MakeEvolveRequest(prevSigners...)
	fmt.Println(err)

	// Client sends request r and serialised darc d2Buf to the server, and
	// the server must verify it. Usually the server will look in its
	// database for the base ID of the darc in the request and find the
	// latest one. But in this case we assume it already knows. If the
	// verification is successful, then the server should add the darc in
	// the request to its database.
	fmt.Println(r.Verify(d1)) // Assume we can find d1 given r.
	d2Server, _ := r.MsgToDarc(d2Buf)
	fmt.Println(bytes.Equal(d2Server.GetID(), darcEvol.GetID()))

	// If the darcs stored on the server are trustworthy, then using
	// `Request.Verify` is enough. To do a complete verification,
	// Darc.Verify should be used. This will traverse the chain of
	// evolution and verify every evolution. However, the Darc.Path
	// attribute must be set.
	fmt.Println(d2Server.VerifyWithCB(func(s string, latest bool) *darc.Darc {
		if s == darc.NewIdentityDarc(d1.GetID()).String() {
			return d1
		}
		return nil
	}, true))
	return darcEvol, nil
}

// Search executes a search on the filter in req. See the definition of type
// SearchRequest for additional details about how the filter is interpreted.
// The ID and Instance fields of the SearchRequest will be filled in from c.
func (c *Client) Search(req *SearchRequest) (*SearchResponse, error) {
	req.ID = c.ByzCoin.ID
	req.Instance = c.Instance

	reply := &SearchResponse{}
	if err := c.c.SendProtobuf(c.ByzCoin.Roster.List[0], req, reply); err != nil {
		return nil, err
	}
	return reply, nil
}
