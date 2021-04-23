package contracts

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.dedis.ch/cothority/v3"
	"go.dedis.ch/cothority/v3/byzcoin"
	"go.dedis.ch/cothority/v3/darc"
	"go.dedis.ch/onet/v3"
	"go.dedis.ch/protobuf"
)

// if there isn't the "spawn:query" DARC rule it shouldn't work
func TestQuery_Spawn_No_Rule(t *testing.T) {
	local := onet.NewTCPTest(cothority.Suite)
	defer local.CloseAll()

	signer := darc.NewSignerEd25519(nil, nil)
	_, roster, _ := local.GenTree(3, true)

	genesisMsg, err := byzcoin.DefaultGenesisMsg(byzcoin.CurrentVersion, roster,
		[]string{}, signer.Identity())
	require.NoError(t, err)
	gDarc := &genesisMsg.GenesisDarc

	genesisMsg.BlockInterval = time.Second

	cl, _, err := byzcoin.NewLedger(genesisMsg, false)
	require.NoError(t, err)

	instruction := byzcoin.Instruction{
		InstanceID: byzcoin.NewInstanceID(gDarc.GetBaseID()),
		Spawn: &byzcoin.Spawn{
			ContractID: QueryContractID,
			Args: []byzcoin.Argument{{
				Name:  QueryDescriptionKey,
				Value: []byte("dec"),
			}, {
				Name:  QueryUserIDKey,
				Value: []byte("userID"),
			}, {
				Name:  QueryProjectIDKey,
				Value: []byte("projectID"),
			}, {
				Name:  QueryQueryIDKey,
				Value: []byte("queryID"),
			}, {
				Name:  QueryQueryDefinitionKey,
				Value: []byte("queryDef"),
			}, {
				Name:  QueryStatusKey,
				Value: []byte("status"),
			}},
		},
		SignerCounter: []uint64{1},
	}

	ctx, err := cl.CreateTransaction(instruction)
	require.NoError(t, err)
	require.NoError(t, ctx.FillSignersAndSignWith(signer))

	_, err = cl.AddTransactionAndWait(ctx, 10)
	require.Error(t, err)
}

// Using a project instance to spawn a query instance. Must be rejected since
// the user is not authorized.
func TestQuery_Spawn_With_Project_Rejected(t *testing.T) {
	local := onet.NewTCPTest(cothority.Suite)
	defer local.CloseAll()

	signer := darc.NewSignerEd25519(nil, nil)
	_, roster, _ := local.GenTree(3, true)

	genesisMsg, err := byzcoin.DefaultGenesisMsg(byzcoin.CurrentVersion, roster,
		[]string{"spawn:project"}, signer.Identity())
	require.NoError(t, err)
	gDarc := &genesisMsg.GenesisDarc

	genesisMsg.BlockInterval = time.Second

	cl, _, err := byzcoin.NewLedger(genesisMsg, false)
	require.NoError(t, err)

	projectName := "name"

	ctx, err := addProject(t, projectName, "d", gDarc, signer, cl)
	require.NoError(t, err)

	projectInstID := ctx.Instructions[0].DeriveID("")

	instruction := byzcoin.Instruction{
		// that's the key part, where we provide the instanceID of the project
		// instance we just spawned. This instance will spawn the query.
		InstanceID: projectInstID,
		Spawn: &byzcoin.Spawn{
			ContractID: QueryContractID,
			Args: []byzcoin.Argument{{
				Name:  QueryDescriptionKey,
				Value: []byte("desc"),
			}, {
				Name:  QueryUserIDKey,
				Value: []byte("userID"),
			}, {
				Name:  QueryQueryIDKey,
				Value: []byte("queryID"),
			}, {
				Name:  QueryQueryDefinitionKey,
				Value: []byte("queryDef"),
			}},
		},
		SignerCounter: []uint64{2},
	}

	ctx, err = cl.CreateTransaction(instruction)
	require.NoError(t, err)
	require.NoError(t, ctx.FillSignersAndSignWith(signer))

	_, err = cl.AddTransactionAndWait(ctx, 10)
	require.NoError(t, err)

	queryInstID := ctx.Instructions[0].DeriveID("")

	resp, err := cl.GetProofFromLatest(queryInstID.Slice())
	require.NoError(t, err)

	_, val, _, _, _ := resp.Proof.KeyValue()
	query := QueryContract{}

	err = protobuf.Decode(val, &query)
	require.NoError(t, err)

	require.Equal(t, "desc", query.Description)
	require.Equal(t, "userID", query.UserID)
	require.Equal(t, projectName, query.ProjectID)
	require.Equal(t, "queryID", query.QueryID)
	require.Equal(t, "queryDef", query.QueryDefinition)
	require.Equal(t, QueryRejectedStatus, query.Status)

	local.WaitDone(genesisMsg.BlockInterval)
}

// We use a project to spawn a query instance. We add the user ID in the project
// authorization so the query should be accepted.
func TestQuery_Spawn_With_Project_Pending(t *testing.T) {
	local := onet.NewTCPTest(cothority.Suite)
	defer local.CloseAll()

	signer := darc.NewSignerEd25519(nil, nil)
	_, roster, _ := local.GenTree(3, true)

	genesisMsg, err := byzcoin.DefaultGenesisMsg(byzcoin.CurrentVersion, roster,
		[]string{"spawn:project", "invoke:project.add"}, signer.Identity())
	require.NoError(t, err)
	gDarc := &genesisMsg.GenesisDarc

	genesisMsg.BlockInterval = time.Second

	cl, _, err := byzcoin.NewLedger(genesisMsg, false)
	require.NoError(t, err)

	projectName := "name"
	userID := "userdID"
	queryTerm := "queryTerm"

	ctx, err := addProject(t, projectName, "d", gDarc, signer, cl)
	require.NoError(t, err)

	projectInstID := ctx.Instructions[0].DeriveID("")

	ctx, err = cl.CreateTransaction(byzcoin.Instruction{
		InstanceID: projectInstID,
		Invoke: &byzcoin.Invoke{
			ContractID: ProjectContractID,
			Command:    "add",
			Args: byzcoin.Arguments{{
				Name:  ProjectUserIDKey,
				Value: []byte(userID),
			}, {
				Name:  ProjectQueryTermKey,
				Value: []byte(queryTerm),
			}},
		},
		SignerCounter: []uint64{2},
	})
	require.NoError(t, err)

	err = ctx.FillSignersAndSignWith(signer)
	require.NoError(t, err)

	_, err = cl.AddTransactionAndWait(ctx, 10)
	require.NoError(t, err)

	instruction := byzcoin.Instruction{
		// that's the key part, where we provide the instanceID of the project
		// instance we just spawned. This instance will spawn the query.
		InstanceID: projectInstID,
		Spawn: &byzcoin.Spawn{
			ContractID: QueryContractID,
			Args: []byzcoin.Argument{{
				Name:  QueryDescriptionKey,
				Value: []byte("desc"),
			}, {
				Name:  QueryUserIDKey,
				Value: []byte(userID),
			}, {
				Name:  QueryQueryIDKey,
				Value: []byte("queryID"),
			}, {
				Name:  QueryQueryDefinitionKey,
				Value: []byte(queryTerm),
			}},
		},
		SignerCounter: []uint64{3},
	}

	ctx, err = cl.CreateTransaction(instruction)
	require.NoError(t, err)
	require.NoError(t, ctx.FillSignersAndSignWith(signer))

	_, err = cl.AddTransactionAndWait(ctx, 10)
	require.NoError(t, err)

	queryInstID := ctx.Instructions[0].DeriveID("")

	resp, err := cl.GetProofFromLatest(queryInstID.Slice())
	require.NoError(t, err)

	_, val, _, _, _ := resp.Proof.KeyValue()
	query := QueryContract{}

	err = protobuf.Decode(val, &query)
	require.NoError(t, err)

	require.Equal(t, "desc", query.Description)
	require.Equal(t, userID, query.UserID)
	require.Equal(t, projectName, query.ProjectID)
	require.Equal(t, "queryID", query.QueryID)
	require.Equal(t, queryTerm, query.QueryDefinition)
	require.Equal(t, QueryPendingStatus, query.Status)

	local.WaitDone(genesisMsg.BlockInterval)
}

// only QuerySuccessStatus and QueryFailedStatus are allowed
func TestQuery_Invoke_Update_Wrong_Status(t *testing.T) {
	local := onet.NewTCPTest(cothority.Suite)
	defer local.CloseAll()

	signer := darc.NewSignerEd25519(nil, nil)
	_, roster, _ := local.GenTree(3, true)

	genesisMsg, err := byzcoin.DefaultGenesisMsg(byzcoin.CurrentVersion, roster,
		[]string{"spawn:project"}, signer.Identity())
	require.NoError(t, err)
	gDarc := &genesisMsg.GenesisDarc

	genesisMsg.BlockInterval = time.Second

	cl, _, err := byzcoin.NewLedger(genesisMsg, false)
	require.NoError(t, err)

	projectName := "name"

	ctx, err := addProject(t, projectName, "d", gDarc, signer, cl)
	require.NoError(t, err)

	projectInstID := ctx.Instructions[0].DeriveID("")

	instruction := byzcoin.Instruction{
		// that's the key part, where we provide the instanceID of the project
		// instance we just spawned. This instance will spawn the query.
		InstanceID: projectInstID,
		Spawn: &byzcoin.Spawn{
			ContractID: QueryContractID,
			Args: []byzcoin.Argument{{
				Name:  QueryDescriptionKey,
				Value: []byte("desc"),
			}, {
				Name:  QueryUserIDKey,
				Value: []byte("userID"),
			}, {
				Name:  QueryQueryIDKey,
				Value: []byte("queryID"),
			}, {
				Name:  QueryQueryDefinitionKey,
				Value: []byte("queryDef"),
			}},
		},
		SignerCounter: []uint64{2},
	}

	ctx, err = cl.CreateTransaction(instruction)
	require.NoError(t, err)
	require.NoError(t, ctx.FillSignersAndSignWith(signer))

	_, err = cl.AddTransactionAndWait(ctx, 10)
	require.NoError(t, err)

	queryInstID := ctx.Instructions[0].DeriveID("")

	// update the status

	instruction = byzcoin.Instruction{
		// that's the key part, where we provide the instanceID of the project
		// instance we just spawned. This instance will spawn the query.
		InstanceID: queryInstID,
		Invoke: &byzcoin.Invoke{
			Command:    QueryUpdateAction,
			ContractID: QueryContractID,
			Args: []byzcoin.Argument{{
				Name:  QueryStatusKey,
				Value: []byte("wrong status"),
			}},
		},
		SignerCounter: []uint64{3},
	}

	ctx, err = cl.CreateTransaction(instruction)
	require.NoError(t, err)
	require.NoError(t, ctx.FillSignersAndSignWith(signer))

	_, err = cl.AddTransactionAndWait(ctx, 10)
	require.Error(t, err)

	local.WaitDone(genesisMsg.BlockInterval)
}

func TestQuery_Invoke_Update_Good_Status(t *testing.T) {
	local := onet.NewTCPTest(cothority.Suite)
	defer local.CloseAll()

	signer := darc.NewSignerEd25519(nil, nil)
	_, roster, _ := local.GenTree(3, true)

	genesisMsg, err := byzcoin.DefaultGenesisMsg(byzcoin.CurrentVersion, roster,
		[]string{"spawn:project"}, signer.Identity())
	require.NoError(t, err)
	gDarc := &genesisMsg.GenesisDarc

	genesisMsg.BlockInterval = time.Second

	cl, _, err := byzcoin.NewLedger(genesisMsg, false)
	require.NoError(t, err)

	projectName := "name"

	ctx, err := addProject(t, projectName, "d", gDarc, signer, cl)
	require.NoError(t, err)

	projectInstID := ctx.Instructions[0].DeriveID("")

	instruction := byzcoin.Instruction{
		// that's the key part, where we provide the instanceID of the project
		// instance we just spawned. This instance will spawn the query.
		InstanceID: projectInstID,
		Spawn: &byzcoin.Spawn{
			ContractID: QueryContractID,
			Args: []byzcoin.Argument{{
				Name:  QueryDescriptionKey,
				Value: []byte("desc"),
			}, {
				Name:  QueryUserIDKey,
				Value: []byte("userID"),
			}, {
				Name:  QueryQueryIDKey,
				Value: []byte("queryID"),
			}, {
				Name:  QueryQueryDefinitionKey,
				Value: []byte("queryDef"),
			}},
		},
		SignerCounter: []uint64{2},
	}

	ctx, err = cl.CreateTransaction(instruction)
	require.NoError(t, err)
	require.NoError(t, ctx.FillSignersAndSignWith(signer))

	_, err = cl.AddTransactionAndWait(ctx, 10)
	require.NoError(t, err)

	queryInstID := ctx.Instructions[0].DeriveID("")

	// update the status

	instruction = byzcoin.Instruction{
		// that's the key part, where we provide the instanceID of the project
		// instance we just spawned. This instance will spawn the query.
		InstanceID: queryInstID,
		Invoke: &byzcoin.Invoke{
			Command:    QueryUpdateAction,
			ContractID: QueryContractID,
			Args: []byzcoin.Argument{{
				Name:  QueryStatusKey,
				Value: []byte(QuerySuccessStatus),
			}},
		},
		SignerCounter: []uint64{3},
	}

	ctx, err = cl.CreateTransaction(instruction)
	require.NoError(t, err)
	require.NoError(t, ctx.FillSignersAndSignWith(signer))

	_, err = cl.AddTransactionAndWait(ctx, 10)
	require.NoError(t, err)

	resp, err := cl.GetProofFromLatest(queryInstID.Slice())
	require.NoError(t, err)

	_, val, _, _, _ := resp.Proof.KeyValue()
	query := QueryContract{}

	err = protobuf.Decode(val, &query)
	require.NoError(t, err)

	require.Equal(t, QuerySuccessStatus, query.Status)

	local.WaitDone(genesisMsg.BlockInterval)
}
