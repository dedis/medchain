package contract

import (
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.dedis.ch/cothority/v3"
	"go.dedis.ch/cothority/v3/byzcoin"
	"go.dedis.ch/cothority/v3/darc"
	"go.dedis.ch/cothority/v3/skipchain"
	"go.dedis.ch/kyber/v3/suites"
	"go.dedis.ch/onet/v3"
	"go.dedis.ch/protobuf"
)

var tSuite = suites.MustFind("Ed25519")

// Use this block interval for logic tests. Stress test often use a different
// block interval.
var testBlockInterval = 500 * time.Millisecond

// func TestMain(m *testing.M) {
// 	log.MainTest(m)
// }

func TestClient_Medchain(t *testing.T) {
	s, c := newSer(t)
	leader := s.services[0]
	defer s.close()

	err := c.Create()
	require.Nil(t, err)
	require.NotNil(t, c.Instance)
	// waitForKey(t, leader.omni, c.ByzCoin.ID, c.Instance.Slice(), testBlockInterval)

	ids, err := c.WriteQueries(NewQuery("query1", "submitted")) //, NewQuery("query2", "submitted"), NewQuery("query3", "submitted"))

	require.Nil(t, err)
	require.Equal(t, 1, len(ids))
	require.Equal(t, 32, len(ids[0]))

	// Loop while we wait for the next block to be created.
	// waitForKey(t, leader.omni, c.ByzCoin.ID, ids[2], testBlockInterval)

	// Check consistency and # of queries.
	for i := 0; i < 10; i++ {
		leader.waitForBlock(c.ByzCoin.ID)
	}

	// // Fetch the index, and check its length.
	_ = checkProof(t, c, leader.omni, c.Instance.Slice(), c.ByzCoin.ID)
	// expected := len(c.Instance)
	// require.Equal(t, expected, len(idx), fmt.Sprintf("index key content is %v, expected %v", len(idx), expected))

	// Use the client API to get the query back
	for _, key := range ids {
		_, err := c.GetQuery(key)
		//fmt.Println(k)
		require.Nil(t, err)
	}

	// Test naming, this is just a sanity check for medchain, the main
	// naming test is in the byzcoin package.
	// use contract_name to create a mapping from a darc ID and name tuple
	// to another instance ID
	spawnTx := byzcoin.ClientTransaction{
		Instructions: byzcoin.Instructions{
			{
				InstanceID: byzcoin.NewInstanceID(s.genDarc.GetBaseID()),
				Spawn: &byzcoin.Spawn{
					ContractID: byzcoin.ContractNamingID,
				},
				SignerCounter: c.incrementCtrs(),
			},
		},
	}
	require.NoError(t, spawnTx.FillSignersAndSignWith(c.Signers...))
	_, err = c.ByzCoin.AddTransactionAndWait(spawnTx, 10)
	require.NoError(t, err)

	namingTx, err := c.ByzCoin.CreateTransaction(byzcoin.Instruction{
		InstanceID: byzcoin.NamingInstanceID,
		Invoke: &byzcoin.Invoke{
			ContractID: byzcoin.ContractNamingID,
			Command:    "add",
			Args: byzcoin.Arguments{
				{
					Name:  "instanceID",
					Value: c.Instance.Slice(),
				},
				{
					Name:  "name",
					Value: []byte("queryContract"),
				},
			},
		},
		SignerCounter: c.incrementCtrs(),
	})
	require.NoError(t, err)
	require.NoError(t, namingTx.FillSignersAndSignWith(c.Signers...))

	_, err = c.ByzCoin.AddTransactionAndWait(namingTx, 10)
	require.NoError(t, err)

	replyID, err := c.ByzCoin.ResolveInstanceID(s.genDarc.GetID(), "queryContract")
	require.NoError(t, err)
	require.Equal(t, replyID, c.Instance)
}

func TestClient_Query100(t *testing.T) {
	if testing.Short() {
		return
	}
	s, c := newSer(t)
	leader := s.services[0]
	defer s.close()

	err := c.Create()
	require.Nil(t, err)
	waitForKey(t, leader.omni, c.ByzCoin.ID, c.Instance.Slice(), time.Second)

	qCount := 100
	// Write the queries in chunks to make sure that the verification
	// can be done in time.
	for i := 0; i < 5; i++ {
		current := s.getCurrentBlock(t)

		start := i * qCount / 5
		for ct := start; ct < start+qCount/5; ct++ {
			_, err := c.WriteQueries(NewQuery("query"+string(ct), "submitted"))
			require.Nil(t, err)
		}

		s.waitNextBlock(t, current)
	}

	// Also, one call to write a query with multiple queries in it.
	for i := 0; i < 5; i++ {
		current := s.getCurrentBlock(t)

		qs := make([]Query, qCount/5)
		for j := range qs {
			qs[j] = NewQuery("query"+string(j), "submitted")
		}
		_, err = c.WriteQueries(qs...)
		require.Nil(t, err)

		s.waitNextBlock(t, current)
	}

	for i := 0; i < 10; i++ {
		// leader.waitForBlock isn't enough, so wait a bit longer.
		time.Sleep(s.req.BlockInterval)
		leader.waitForBlock(c.ByzCoin.ID)
	}
	require.Nil(t, err)

	// Fetch index, and check its length.
	idx := checkProof(t, c, leader.omni, c.Instance.Slice(), c.ByzCoin.ID)
	expected := len(c.Instance)
	require.Equal(t, expected, len(idx), fmt.Sprintf("index key content is %v, expected %v", len(idx), expected))

	// for _, eventID := range queryIDs {
	// 	eventBuf := checkProof(t, leader.omni, eventID, c.ByzCoin.ID)
	// 	var q Query
	// 	require.Nil(t, protobuf.Decode(eventBuf, &e))
	// }
	// require.Nil(t, s.local.WaitDone(10*time.Second))
}

func checkProof(t *testing.T, c *Client, omni *byzcoin.Service, key []byte, scID skipchain.SkipBlockID) []byte {

	// req := &byzcoin.GetProof{
	// 	Version: byzcoin.CurrentVersion,
	// 	Key:     key,
	// 	ID:      scID,
	// }
	// resp, err := omni.GetProof(req)
	// require.Nil(t, err)

	// p := resp.Proof
	// fmt.Println(p.InclusionProof.Match(key))
	// require.True(t, p.InclusionProof.Match(key), "proof of exclusion of index")

	// v0, _, _, err := p.Get(key)
	// require.NoError(t, err)
	// fmt.Println(string(v0))
	// return v0

	// Get the proof from byzcoin
	reply, err := c.ByzCoin.GetProof(key)
	require.Nil(t, err)
	// Make sure the proof is a matching proof and not a proof of absence.
	pr := reply.Proof
	require.True(t, pr.InclusionProof.Match(key))

	// Get the raw values of the proof.
	_, val, contractID, dID, err := pr.KeyValue()
	require.Nil(t, err)
	fmt.Println("contractID")
	fmt.Println(contractID)
	fmt.Println("dID")
	fmt.Println(dID)
	// And decode the buffer to a QueryData
	cs := QueryData{}
	err = protobuf.Decode(val, &cs)
	require.Nil(t, err)

	return val
}

func waitForKey(t *testing.T, s *byzcoin.Service, scID skipchain.SkipBlockID, key []byte, interval time.Duration) {
	if len(key) == 0 {
		t.Fatal("key len", len(key))
	}
	var found bool
	var resp *byzcoin.GetProofResponse
	for ct := 0; ct < 10; ct++ {
		req := &byzcoin.GetProof{
			Version: byzcoin.CurrentVersion,
			Key:     key,
			ID:      scID,
		}
		var err error
		resp, err = s.GetProof(req)
		if err == nil {
			p := resp.Proof.InclusionProof
			if p.Match(key) {
				found = true
				break
			}
		} else {
			t.Log("err", err)
		}
		time.Sleep(interval)
	}
	if !found {
		require.Fail(t, "timeout")
	}
	_, _, _, err := resp.Proof.Get(key)
	require.NoError(t, err)
}

type ser struct {
	local    *onet.LocalTest
	hosts    []*onet.Server
	roster   *onet.Roster
	services []*Service
	id       skipchain.SkipBlockID
	owner    darc.Signer
	req      *byzcoin.CreateGenesisBlock
	genDarc  *darc.Darc // the genesis darc
}

func (s *ser) close() {
	s.local.CloseAll()
}

func newSer(t *testing.T) (*ser, *Client) {
	s := &ser{
		local: onet.NewTCPTest(tSuite),
		owner: darc.NewSignerEd25519(nil, nil),
	}
	s.hosts, s.roster, _ = s.local.GenTree(3, true)

	for sv := range s.local.GetServices(s.hosts, sid) {

		service := s.local.GetServices(s.hosts, sid)[sv].(*Service)
		s.services = append(s.services, service)
	}

	fmt.Println("length of sevices:" + strconv.Itoa(len(s.services)))
	var err error
	s.req, err = byzcoin.DefaultGenesisMsg(byzcoin.CurrentVersion, s.roster,
		[]string{"spawn:" + contractName, "invoke:" + contractName + "." + "update", "invoke:" + contractName + "." + "verifystatus", "_name:" + contractName}, s.owner.Identity())
	if err != nil {
		t.Fatal(err)
	}
	s.genDarc = &s.req.GenesisDarc
	fmt.Println(s.genDarc.String())
	s.req.BlockInterval = testBlockInterval
	cl := onet.NewClient(cothority.Suite, byzcoin.ServiceName)

	var resp byzcoin.CreateGenesisBlockResponse
	err = cl.SendProtobuf(s.roster.List[0], s.req, &resp)
	if err != nil {
		t.Fatal(err)
	}
	s.id = resp.Skipblock.Hash

	ol := byzcoin.NewClient(s.id, *s.roster)

	c := NewClient(ol)
	c.DarcID = s.genDarc.GetBaseID()
	c.Signers = []darc.Signer{s.owner}
	c.genDarc = s.genDarc

	// The user must include at least one contract that can be parsed as a
	// DARC and it must exist.
	fmt.Println("Num of darc contracts")
	fmt.Println(len(s.req.DarcContractIDs))
	return s, c
}

func (s *ser) getCurrentBlock(t *testing.T) skipchain.SkipBlockID {
	reply, err := skipchain.NewClient().GetUpdateChain(s.roster, s.id)
	require.Nil(t, err)
	return reply.Update[len(reply.Update)-1].Hash
}

func (s *ser) waitNextBlock(t *testing.T, current skipchain.SkipBlockID) {
	for i := 0; i < 10; i++ {
		reply, err := skipchain.NewClient().GetUpdateChain(s.roster, s.id)
		require.Nil(t, err)
		if !current.Equal(reply.Update[len(reply.Update)-1].Hash) {
			return
		}
		time.Sleep(s.req.BlockInterval)
	}
	require.Fail(t, "waited too long for new block to appear")
}

// waitForBlock is for use in tests; it will sleep long enough to be sure that
// a block has been created.
func (s *Service) waitForBlock(scID skipchain.SkipBlockID) {
	dur, _, err := s.omni.LoadBlockInfo(scID)
	if err != nil {
		panic(err.Error())
	}
	time.Sleep(5 * dur)
}
