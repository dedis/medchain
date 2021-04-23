package contracts

import (
	"go.dedis.ch/cothority/v3/byzcoin"
	"go.dedis.ch/onet/v3/log"
	"go.dedis.ch/protobuf"
	"golang.org/x/xerrors"
)

// QueryContractID is the name of the query Contract.
//
// The query contract represents a request from a user to perform an action on a
// project (dataset). This contract is spawned by the Project contract. The
// project contract will set the "status" field to "pending" or "rejected"
// when it spawns the contract, based on the project attributes.
const QueryContractID = "query"

const (
	QueryDescriptionKey     = "description"
	QueryUserIDKey          = "userID"
	QueryProjectIDKey       = "projectID"
	QueryQueryIDKey         = "queryID"
	QueryQueryDefinitionKey = "queryDefinition"
	QueryStatusKey          = "status"

	QueryUpdateAction = "update"

	QueryRejectedStatus = "rejected"
	QueryPendingStatus  = "pending"
	QuerySuccessStatus  = "successful"
	QueryFailedStatus   = "failed"
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

	UserID          string
	ProjectID       string
	QueryID         string
	QueryDefinition string

	Status string
}

// VerifyInstruction implements byzcoin.Contract
func (c QueryContract) VerifyInstruction(rst byzcoin.ReadOnlyStateTrie,
	inst byzcoin.Instruction, ctxHash []byte) error {

	// TODO: who is allowed to invoke:update a query ???
	return nil
}

// Spawn implements byzcoin.Contract
func (c QueryContract) Spawn(rst byzcoin.ReadOnlyStateTrie, inst byzcoin.Instruction,
	coins []byzcoin.Coin) ([]byzcoin.StateChange, []byzcoin.Coin, error) {

	return nil, nil, xerrors.Errorf("spawn must be done via the project contract")
}

// Invoke implements byzcoin.Contract
func (c QueryContract) Invoke(rst byzcoin.ReadOnlyStateTrie, inst byzcoin.Instruction,
	coins []byzcoin.Coin) ([]byzcoin.StateChange, []byzcoin.Coin, error) {

	if inst.Invoke.Command != QueryUpdateAction {
		return nil, nil, xerrors.Errorf("only the update action is allowed")
	}

	status := string(inst.Arguments().Search(QueryStatusKey))
	if status != QuerySuccessStatus && status != QueryFailedStatus {
		return nil, nil, xerrors.Errorf("invalid status: %s", status)
	}

	c.Status = status

	buf, err := protobuf.Encode(&c)
	if err != nil {
		return nil, nil, xerrors.Errorf("failed to encode query: %v", err)
	}

	_, _, _, darcID, err := rst.GetValues(inst.InstanceID.Slice())
	if err != nil {
		return nil, nil, xerrors.Errorf("failed to get DARC: %v", err)
	}

	sc := byzcoin.NewStateChange(byzcoin.Update, inst.InstanceID,
		ProjectContractID, buf, darcID)

	return []byzcoin.StateChange{sc}, coins, nil
}

// Delete implements byzcoin.Contract
func (c QueryContract) Delete(_ byzcoin.ReadOnlyStateTrie, _ byzcoin.Instruction,
	_ []byzcoin.Coin) ([]byzcoin.StateChange, []byzcoin.Coin, error) {

	return nil, nil, xerrors.Errorf("delete not allowed in query contract")
}
