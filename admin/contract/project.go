package contracts

import (
	"fmt"
	"strings"

	"go.dedis.ch/cothority/v3/byzcoin"
	"go.dedis.ch/cothority/v3/darc"
	"go.dedis.ch/onet/v3/log"
	"go.dedis.ch/protobuf"
	"golang.org/x/xerrors"
)

// ProjectContractID is the name of the query Contract.
const ProjectContractID = "project"

const (
	ProjectDescriptionKey = "description"
	ProjectNameKey        = "name"
	ProjectUserIDKey      = "userID"
	ProjectActionKey      = "action"
)

func init() {
	err := byzcoin.RegisterGlobalContract(ProjectContractID, projectContractFromBytes)
	if err != nil {
		log.ErrFatal(err)
	}
}

// projectContractFromBytes unmarshals a contract
func projectContractFromBytes(in []byte) (byzcoin.Contract, error) {
	var c ProjectContract

	fmt.Println("project content:", in)
	fmt.Printf("%s\n", in)

	err := protobuf.Decode(in, &c)
	if err != nil {
		return nil, xerrors.Errorf("failed to decode project: %v", err)
	}

	return &c, nil
}

// ProjectContract is a smart contract that defines the attributes of a project,
// and allow one to spawn query smart contracts.
type ProjectContract struct {
	byzcoin.BasicContract

	AdminInstance  []byte
	Name           string
	Description    string
	Authorizations Authorizations
}

// VerifyInstruction implements byzcoin.Contract.
func (p *ProjectContract) VerifyInstruction(rst byzcoin.ReadOnlyStateTrie,
	inst byzcoin.Instruction, ctxHash []byte) error {

	// In the case the project is used to spawn a query contract, we perform
	// the verification against the project's authorizations.
	if inst.ContractID() == QueryContractID {
		action := string(inst.Spawn.Args.Search(QueryActionKey))

		return p.verifyQuery(inst.SignerIdentities, action)
	}

	return inst.Verify(rst, ctxHash)
}

// VerifyDeferredInstruction implements byzcoin.Contract.
func (p *ProjectContract) VerifyDeferredInstruction(rst byzcoin.ReadOnlyStateTrie,
	inst byzcoin.Instruction, ctxHash []byte) error {

	opts := &byzcoin.VerificationOptions{IgnoreCounters: true}
	return inst.VerifyWithOption(rst, ctxHash, opts)
}

// Spawn implements byzcoin.Contract.
func (p *ProjectContract) Spawn(rst byzcoin.ReadOnlyStateTrie,
	inst byzcoin.Instruction, coins []byzcoin.Coin) ([]byzcoin.StateChange, []byzcoin.Coin, error) {

	// The project contract is also a spawner for the query contract. We use a
	// custom spawner to make the special verification against the
	// authorizations stored on this project.
	if inst.Spawn.ContractID == QueryContractID {
		return p.spawnQuery(rst, inst, coins)
	}

	var darcID darc.ID
	_, _, _, darcID, err := rst.GetValues(inst.InstanceID.Slice())
	if err != nil {
		return nil, nil, xerrors.Errorf("failed to get DARC: %v", err)
	}

	description := string(inst.Spawn.Args.Search(ProjectDescriptionKey))
	name := string(inst.Spawn.Args.Search(ProjectNameKey))

	state := ProjectContract{
		AdminInstance:  inst.InstanceID.Slice(),
		Description:    description,
		Name:           name,
		Authorizations: make(Authorizations, 0),
	}

	buf, err := protobuf.Encode(&state)
	if err != nil {
		return nil, nil, xerrors.Errorf("failed to encode state: %v", err)
	}

	sc := byzcoin.NewStateChange(byzcoin.Create, inst.DeriveID(""), ProjectContractID,
		buf, darcID)

	return []byzcoin.StateChange{sc}, coins, nil
}

// Invoke implements byzcoin.Contract.
func (p ProjectContract) Invoke(rst byzcoin.ReadOnlyStateTrie,
	inst byzcoin.Instruction, coins []byzcoin.Coin) ([]byzcoin.StateChange, []byzcoin.Coin, error) {

	_, _, _, darcID, err := rst.GetValues(inst.InstanceID.Slice())
	if err != nil {
		return nil, nil, xerrors.Errorf("failed to get DARC: %v", err)
	}

	userID := string(inst.Arguments().Search(ProjectUserIDKey))
	action := string(inst.Arguments().Search(ProjectActionKey))

	switch inst.Invoke.Command {
	case "add":
		p.updateAuth(userID, action)
	case "remove":
		p.removeAuth(userID, action)
	}

	buf, err := protobuf.Encode(&p)
	if err != nil {
		return nil, nil, xerrors.Errorf("failed to marshal project: %v", err)
	}

	sc := byzcoin.NewStateChange(byzcoin.Update, inst.InstanceID,
		ProjectContractID, buf, darcID)

	return []byzcoin.StateChange{sc}, coins, nil
}

// Delete implements byzcoin.Contract
func (p ProjectContract) Delete(_ byzcoin.ReadOnlyStateTrie, _ byzcoin.Instruction,
	_ []byzcoin.Coin) ([]byzcoin.StateChange, []byzcoin.Coin, error) {

	return nil, nil, xerrors.Errorf("delete not allowed in project contract")
}

func (p *ProjectContract) spawnQuery(rst byzcoin.ReadOnlyStateTrie,
	inst byzcoin.Instruction, coins []byzcoin.Coin) ([]byzcoin.StateChange, []byzcoin.Coin, error) {

	contract, err := queryContractFromBytes(nil)
	if err != nil {
		return nil, nil, xerrors.Errorf("failed to init query contract")
	}

	args := inst.Spawn.Args
	projectArg := byzcoin.Argument{Name: queryProjectKey, Value: []byte(p.Name)}
	inst.Spawn.Args = append([]byzcoin.Argument{projectArg}, args...)

	return contract.Spawn(rst, inst, coins)
}

func (p ProjectContract) String() string {
	out := new(strings.Builder)
	fmt.Fprintln(out, "- Project")
	fmt.Fprintf(out, "-- Name: %s\n", p.Name)
	fmt.Fprintf(out, "-- Description: %s\n", p.Description)
	fmt.Fprintf(out, "-- Authorization:\n%s", p.Authorizations)

	return out.String()
}

func (p *ProjectContract) verifyQuery(signers []darc.Identity, action string) error {
	// check if any signer is allowed
	for _, signer := range signers {
		entry := p.Authorizations.Find(signer.String())
		if entry != nil && entry.IsAllowed(string(action)) {
			return nil
		}
	}

	return xerrors.New("identity(ies) not allowed")
}

func (p *ProjectContract) updateAuth(userID, action string) {
	entry := p.Authorizations.Find(userID)
	if entry == nil {
		entry = &Authorization{UserID: userID, Actions: []string{}}
		p.Authorizations = append(p.Authorizations, entry)
	}

	if entry.IsAllowed(action) {
		return
	}

	entry.Actions = append(entry.Actions, action)
}

func (p *ProjectContract) removeAuth(userID, action string) {
	entry := p.Authorizations.Find(userID)
	if entry == nil {
		return
	}

	var i int
	for i = 0; i < len(entry.Actions); i++ {
		if entry.Actions[i] == action {
			break
		}
	}

	if i == len(entry.Actions) {
		// action not present, nothing to remove
		return
	}

	entry.Actions = append(entry.Actions[:i], entry.Actions[i+1:]...)
}

// Authorizations ...
type Authorizations []*Authorization

// Find search for an entry an return nil if not found.
func (e Authorizations) Find(userID string) *Authorization {
	for _, entry := range e {
		if entry.UserID == userID {
			return entry
		}
	}

	return nil
}

func (e Authorizations) String() string {
	out := new(strings.Builder)

	for i, entry := range e {
		fmt.Fprintf(out, "- authorization %d:\n%s", i, entry)
	}

	return out.String()
}

// Authorization ...
type Authorization struct {
	UserID  string
	Actions []string
}

// IsAllowed checks if the action is present in the entry.
func (e Authorization) IsAllowed(a string) bool {
	for _, action := range e.Actions {
		if action == a {
			return true
		}
	}

	return false
}

func (e Authorization) String() string {
	out := new(strings.Builder)

	fmt.Fprintf(out, "- UserID: %s\n", e.UserID)
	fmt.Fprintf(out, "- Actions: %v\n", e.Actions)

	return out.String()
}
