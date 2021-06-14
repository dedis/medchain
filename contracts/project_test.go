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

// We try to spawn a project without setting the spawn:project DARC rule.
func TestProject_Spawn_No_Rule(t *testing.T) {
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

	_, err = addProject(t, "n", "d", gDarc, signer, cl)
	require.Error(t, err)
}

func TestProject_Spawn_Ok(t *testing.T) {
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

	description := "desc"
	name := "name"

	ctx, err := addProject(t, name, description, gDarc, signer, cl)
	require.NoError(t, err)

	instID := ctx.Instructions[0].DeriveID("")

	resp, err := cl.GetProofFromLatest(instID.Slice())
	require.NoError(t, err)

	_, val, _, _, _ := resp.Proof.KeyValue()
	project := ProjectContract{}

	err = protobuf.Decode(val, &project)
	require.NoError(t, err)

	require.Equal(t, description, project.Description)
	require.Equal(t, name, project.Name)

	local.WaitDone(genesisMsg.BlockInterval)
}

// Using a wrong command should fail
func TestProject_Invoke_Wrong_Command(t *testing.T) {
	local := onet.NewTCPTest(cothority.Suite)
	defer local.CloseAll()

	signer := darc.NewSignerEd25519(nil, nil)
	_, roster, _ := local.GenTree(3, true)

	genesisMsg, err := byzcoin.DefaultGenesisMsg(byzcoin.CurrentVersion, roster,
		[]string{"spawn:project", "invoke:project.wrong"}, signer.Identity())
	require.NoError(t, err)
	gDarc := &genesisMsg.GenesisDarc

	genesisMsg.BlockInterval = time.Second

	cl, _, err := byzcoin.NewLedger(genesisMsg, false)
	require.NoError(t, err)

	description := "desc"
	name := "name"

	ctx, err := addProject(t, name, description, gDarc, signer, cl)
	require.NoError(t, err)

	instID := ctx.Instructions[0].DeriveID("")

	ctx, err = cl.CreateTransaction(byzcoin.Instruction{
		InstanceID: instID,
		Invoke: &byzcoin.Invoke{
			ContractID: ProjectContractID,
			Command:    "wrong",
			Args:       byzcoin.Arguments{},
		},
		SignerCounter: []uint64{2},
	})
	require.NoError(t, err)

	err = ctx.FillSignersAndSignWith(signer)
	require.NoError(t, err)

	_, err = cl.AddTransactionAndWait(ctx, 10)
	require.Error(t, err)

	local.WaitDone(genesisMsg.BlockInterval)
}

func TestProject_Invoke_Add(t *testing.T) {
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

	ctx, err := addProject(t, "n", "d", gDarc, signer, cl)
	require.NoError(t, err)

	instID := ctx.Instructions[0].DeriveID("")

	userID1 := "userID1"
	userID2 := "userID2"
	queryTerm1 := "q1"
	queryTerm2 := "q2,q3, q4" // can be a coma separated list of query term

	ctx, err = cl.CreateTransaction(byzcoin.Instruction{
		InstanceID: instID,
		Invoke: &byzcoin.Invoke{
			ContractID: ProjectContractID,
			Command:    "add",
			Args: byzcoin.Arguments{{
				Name:  ProjectUserIDKey,
				Value: []byte(userID1),
			}, {
				Name:  ProjectQueryTermKey,
				Value: []byte(queryTerm1),
			}},
		},
		SignerCounter: []uint64{2},
	}, byzcoin.Instruction{
		InstanceID: instID,
		Invoke: &byzcoin.Invoke{
			ContractID: ProjectContractID,
			Command:    "add",
			Args: byzcoin.Arguments{{
				Name:  ProjectUserIDKey,
				Value: []byte(userID1),
			}, {
				Name:  ProjectQueryTermKey,
				Value: []byte(queryTerm2),
			}},
		},
		SignerCounter: []uint64{3},
		// adding two times the same userID/queryTerm should add it only once
	}, byzcoin.Instruction{
		InstanceID: instID,
		Invoke: &byzcoin.Invoke{
			ContractID: ProjectContractID,
			Command:    "add",
			Args: byzcoin.Arguments{{
				Name:  ProjectUserIDKey,
				Value: []byte(userID1),
			}, {
				Name:  ProjectQueryTermKey,
				Value: []byte(queryTerm2),
			}},
		},
		SignerCounter: []uint64{4},
	}, byzcoin.Instruction{
		InstanceID: instID,
		Invoke: &byzcoin.Invoke{
			ContractID: ProjectContractID,
			Command:    "add",
			Args: byzcoin.Arguments{{
				Name:  ProjectUserIDKey,
				Value: []byte(userID2),
			}, {
				Name:  ProjectQueryTermKey,
				Value: []byte(queryTerm1),
			}},
		},
		SignerCounter: []uint64{5},
	})
	require.NoError(t, err)

	err = ctx.FillSignersAndSignWith(signer)
	require.NoError(t, err)

	_, err = cl.AddTransactionAndWait(ctx, 10)
	require.NoError(t, err)

	resp, err := cl.GetProofFromLatest(instID.Slice())
	require.NoError(t, err)

	_, val, _, _, _ := resp.Proof.KeyValue()
	project := ProjectContract{}

	err = protobuf.Decode(val, &project)
	require.NoError(t, err)

	expected := Authorizations{
		&Authorization{
			UserID:     userID1,
			QueryTerms: []string{queryTerm1, "q2", "q3", "q4"},
		},
		&Authorization{
			UserID:     userID2,
			QueryTerms: []string{queryTerm1},
		},
	}
	require.Equal(t, expected, project.Authorizations)

	local.WaitDone(genesisMsg.BlockInterval)
}

func TestProject_Invoke_Remove(t *testing.T) {
	local := onet.NewTCPTest(cothority.Suite)
	defer local.CloseAll()

	signer := darc.NewSignerEd25519(nil, nil)
	_, roster, _ := local.GenTree(3, true)

	genesisMsg, err := byzcoin.DefaultGenesisMsg(byzcoin.CurrentVersion, roster,
		[]string{"spawn:project", "invoke:project.add", "invoke:project.remove"},
		signer.Identity())
	require.NoError(t, err)
	gDarc := &genesisMsg.GenesisDarc

	genesisMsg.BlockInterval = time.Second

	cl, _, err := byzcoin.NewLedger(genesisMsg, false)
	require.NoError(t, err)

	ctx, err := addProject(t, "n", "d", gDarc, signer, cl)
	require.NoError(t, err)

	instID := ctx.Instructions[0].DeriveID("")

	userID1 := "userID1"
	userID2 := "userID2"
	queryTerm1 := "q1"
	queryTerm2 := "q2"

	ctx, err = cl.CreateTransaction(byzcoin.Instruction{
		InstanceID: instID,
		Invoke: &byzcoin.Invoke{
			ContractID: ProjectContractID,
			Command:    "add",
			Args: byzcoin.Arguments{{
				Name:  ProjectUserIDKey,
				Value: []byte(userID1),
			}, {
				Name:  ProjectQueryTermKey,
				Value: []byte(queryTerm1),
			}},
		},
		SignerCounter: []uint64{2},
	}, byzcoin.Instruction{
		InstanceID: instID,
		Invoke: &byzcoin.Invoke{
			ContractID: ProjectContractID,
			Command:    "add",
			Args: byzcoin.Arguments{{
				Name:  ProjectUserIDKey,
				Value: []byte(userID2),
			}, {
				Name:  ProjectQueryTermKey,
				Value: []byte(queryTerm2),
			}},
		},
		SignerCounter: []uint64{3},
	})
	require.NoError(t, err)

	err = ctx.FillSignersAndSignWith(signer)
	require.NoError(t, err)

	_, err = cl.AddTransactionAndWait(ctx, 10)
	require.NoError(t, err)

	ctx, err = cl.CreateTransaction(byzcoin.Instruction{
		InstanceID: instID,
		Invoke: &byzcoin.Invoke{
			ContractID: ProjectContractID,
			Command:    "remove",
			Args: byzcoin.Arguments{{
				Name:  ProjectUserIDKey,
				Value: []byte(userID1),
			}, {
				Name:  ProjectQueryTermKey,
				Value: []byte(queryTerm1),
			}},
		},
		SignerCounter: []uint64{4},
		// removing an inexistent userID/queryTerm is fine
	}, byzcoin.Instruction{
		InstanceID: instID,
		Invoke: &byzcoin.Invoke{
			ContractID: ProjectContractID,
			Command:    "remove",
			Args: byzcoin.Arguments{{
				Name:  ProjectUserIDKey,
				Value: []byte(userID1),
			}, {
				Name:  ProjectQueryTermKey,
				Value: []byte(queryTerm1),
			}},
		},
		SignerCounter: []uint64{5},
	})
	require.NoError(t, err)

	err = ctx.FillSignersAndSignWith(signer)
	require.NoError(t, err)

	_, err = cl.AddTransactionAndWait(ctx, 10)
	require.NoError(t, err)

	resp, err := cl.GetProofFromLatest(instID.Slice())
	require.NoError(t, err)

	_, val, _, _, _ := resp.Proof.KeyValue()
	project := ProjectContract{}

	err = protobuf.Decode(val, &project)
	require.NoError(t, err)

	expected := Authorizations{
		&Authorization{
			UserID:     userID1,
			QueryTerms: []string(nil),
		},
		&Authorization{
			UserID:     userID2,
			QueryTerms: []string{queryTerm2},
		},
	}
	require.Equal(t, expected, project.Authorizations)

	local.WaitDone(genesisMsg.BlockInterval)
}

// delete instruction should return an error
func TestProject_Delete(t *testing.T) {
	local := onet.NewTCPTest(cothority.Suite)
	defer local.CloseAll()

	signer := darc.NewSignerEd25519(nil, nil)
	_, roster, _ := local.GenTree(3, true)

	genesisMsg, err := byzcoin.DefaultGenesisMsg(byzcoin.CurrentVersion, roster,
		[]string{"spawn:project", "delete:project"},
		signer.Identity())
	require.NoError(t, err)
	gDarc := &genesisMsg.GenesisDarc

	genesisMsg.BlockInterval = time.Second

	cl, _, err := byzcoin.NewLedger(genesisMsg, false)
	require.NoError(t, err)

	ctx, err := addProject(t, "n", "d", gDarc, signer, cl)
	require.NoError(t, err)

	instID := ctx.Instructions[0].DeriveID("")

	ctx, err = cl.CreateTransaction(byzcoin.Instruction{
		InstanceID: instID,
		Delete: &byzcoin.Delete{
			ContractID: ProjectContractID,
			Args:       byzcoin.Arguments{{}},
		},
		SignerCounter: []uint64{2},
	})
	require.NoError(t, err)

	err = ctx.FillSignersAndSignWith(signer)
	require.NoError(t, err)

	_, err = cl.AddTransactionAndWait(ctx, 10)
	require.Error(t, err)

	local.WaitDone(genesisMsg.BlockInterval)
}

// -----------------------------------------------------------------------------
// Utility functions

func addProject(t *testing.T, name, description string,
	gDarc *darc.Darc, signer darc.Signer, cl *byzcoin.Client) (byzcoin.ClientTransaction, error) {

	instruction := byzcoin.Instruction{
		InstanceID: byzcoin.NewInstanceID(gDarc.GetBaseID()),
		Spawn: &byzcoin.Spawn{
			ContractID: ProjectContractID,
			Args: []byzcoin.Argument{{
				Name:  ProjectDescriptionKey,
				Value: []byte(description),
			}, {
				Name:  ProjectNameKey,
				Value: []byte(name),
			}},
		},
		SignerCounter: []uint64{1},
	}

	ctx, err := cl.CreateTransaction(instruction)
	require.NoError(t, err)
	require.NoError(t, ctx.FillSignersAndSignWith(signer))

	_, err = cl.AddTransactionAndWait(ctx, 10)
	return ctx, err
}
