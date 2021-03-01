package contracts

import (
	"go.dedis.ch/cothority/v3/byzcoin"
	"go.dedis.ch/onet/v3/log"
	"go.dedis.ch/protobuf"
	"golang.org/x/xerrors"
)

// QueryContractID is the name of the query Contract.
const QueryContractID = "query"

const (
	QueryDescriptionKey = "description"
	queryProjectKey     = "project"
	QueryActionKey      = "action"
)

func init() {
	err := byzcoin.RegisterGlobalContract(QueryContractID, queryContractFromBytes)
	if err != nil {
		log.ErrFatal(err)
	}
}

// queryContractFromBytes unmarshals a contract
func queryContractFromBytes(in []byte) (byzcoin.Contract, error) {
	var c QueryContract

	err := protobuf.Decode(in, &c)
	if err != nil {
		return nil, xerrors.Errorf("failed to decode: %v", err)
	}

	return c, nil
}

// QueryContract is a contract that represents a user query.
//
// - implements byzcoin.Contract
type QueryContract struct {
	byzcoin.BasicContract

	Description string
	Project     string
	Action      string
}

// VerifyInstruction implements byzcoin.Contract
func (c QueryContract) VerifyInstruction(rst byzcoin.ReadOnlyStateTrie,
	inst byzcoin.Instruction, ctxHash []byte) error {

	return xerrors.Errorf("only a spawn through a project contract is allowed")
}

// Spawn implements byzcoin.Contract
func (c QueryContract) Spawn(rst byzcoin.ReadOnlyStateTrie, inst byzcoin.Instruction,
	coins []byzcoin.Coin) ([]byzcoin.StateChange, []byzcoin.Coin, error) {

	_, _, _, darcID, err := rst.GetValues(inst.InstanceID.Slice())
	if err != nil {
		return nil, nil, xerrors.Errorf("failed to get DARC: %v", err)
	}

	description := string(inst.Spawn.Args.Search(QueryDescriptionKey))
	project := string(inst.Spawn.Args.Search(queryProjectKey))
	action := string(inst.Spawn.Args.Search(QueryActionKey))

	state := QueryContract{
		Description: description,
		Project:     project,
		Action:      action,
	}

	buf, err := protobuf.Encode(&state)
	if err != nil {
		return nil, nil, xerrors.Errorf("failed to encode state: %v", err)
	}

	sc := byzcoin.NewStateChange(byzcoin.Create, inst.DeriveID(""), QueryContractID,
		buf, darcID)

	return []byzcoin.StateChange{sc}, coins, nil
}

// Invoke implements byzcoin.Contract
func (c QueryContract) Invoke(_ byzcoin.ReadOnlyStateTrie, _ byzcoin.Instruction,
	_ []byzcoin.Coin) ([]byzcoin.StateChange, []byzcoin.Coin, error) {

	return nil, nil, xerrors.Errorf("invoke not allowed in query contract")
}

// Delete implements byzcoin.Contract
func (c QueryContract) Delete(_ byzcoin.ReadOnlyStateTrie, _ byzcoin.Instruction,
	_ []byzcoin.Coin) ([]byzcoin.StateChange, []byzcoin.Coin, error) {

	return nil, nil, xerrors.Errorf("delete not allowed in query contract")
}
