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
//
// The project contract express the authorization of users on 1 or more
// datasets. This contract can spawn query instances and will set the query
// status based on the authorizations of the project.
const ProjectContractID = "project"

const (
	ProjectDescriptionKey = "description"
	ProjectNameKey        = "name"
	ProjectUserIDKey      = "userID"
	ProjectQueryTermKey   = "queryTerm"
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

	Name           string
	Description    string
	Authorizations Authorizations
}

// VerifyInstruction implements byzcoin.Contract.
func (p *ProjectContract) VerifyInstruction(rst byzcoin.ReadOnlyStateTrie,
	inst byzcoin.Instruction, ctxHash []byte) error {

	if inst.ContractID() == QueryContractID {
		// We don't do any check there because the project's authorization is
		// managed by the admin, and we perform the authorization check in
		// spawnQuery().
		return nil
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
	queryTerm := string(inst.Arguments().Search(ProjectQueryTermKey))

	switch inst.Invoke.Command {
	case "add":
		// queryTerm can be a coma separated list of terms: term1, term1, ...
		for _, a := range strings.Split(queryTerm, ",") {
			a = strings.TrimSpace(a)
			p.updateAuth(userID, a)
		}
	case "remove":
		p.removeAuth(userID, queryTerm)
	default:
		return nil, nil, xerrors.Errorf("wrong command: %s", inst.Invoke.Command)
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

// spawnQuery spawns a query contract and sets its "status" and "projectID"
// arguments. The status is given based on the authorization of the userID
// stored on the authorization of this contract. Status is set to "pending" if
// at least one of the element in the query definition is allowed, otherwise it
// sets the status to "rejected".
func (p *ProjectContract) spawnQuery(rst byzcoin.ReadOnlyStateTrie,
	inst byzcoin.Instruction, coins []byzcoin.Coin) ([]byzcoin.StateChange, []byzcoin.Coin, error) {

	args := inst.Spawn.Args

	queryDefinition := args.Search(QueryQueryDefinitionKey)
	status := QueryRejectedStatus

	auth := p.Authorizations.Find(string(args.Search(QueryUserIDKey)))
	if auth != nil && auth.IsAllowed(string(queryDefinition)) {
		status = QueryPendingStatus
	}

	_, _, _, darcID, err := rst.GetValues(inst.InstanceID.Slice())
	if err != nil {
		return nil, nil, xerrors.Errorf("failed to get DARC: %v", err)
	}

	state := QueryContract{
		Description:     string(args.Search(QueryDescriptionKey)),
		UserID:          string(args.Search(QueryUserIDKey)),
		ProjectID:       p.Name,
		QueryID:         string(args.Search(QueryQueryIDKey)),
		QueryDefinition: string(args.Search(QueryQueryDefinitionKey)),
		Status:          status,
	}

	buf, err := protobuf.Encode(&state)
	if err != nil {
		return nil, nil, xerrors.Errorf("failed to encode state: %v", err)
	}

	sc := byzcoin.NewStateChange(byzcoin.Create, inst.DeriveID(""),
		QueryContractID, buf, darcID)

	return []byzcoin.StateChange{sc}, coins, nil
}

func (p ProjectContract) String() string {
	out := new(strings.Builder)
	fmt.Fprintln(out, "- Project")
	fmt.Fprintf(out, "-- Name: %s\n", p.Name)
	fmt.Fprintf(out, "-- Description: %s\n", p.Description)
	fmt.Fprintf(out, "-- Authorization:\n%s", p.Authorizations)

	return out.String()
}

func (p *ProjectContract) updateAuth(userID, action string) {
	entry := p.Authorizations.Find(userID)
	if entry == nil {
		entry = &Authorization{UserID: userID, QueryTerms: []string{}}
		p.Authorizations = append(p.Authorizations, entry)
	}

	if entry.IsAllowed(action) {
		return
	}

	entry.QueryTerms = append(entry.QueryTerms, action)
}

func (p *ProjectContract) removeAuth(userID, action string) {
	entry := p.Authorizations.Find(userID)
	if entry == nil {
		return
	}

	var i int
	for i = 0; i < len(entry.QueryTerms); i++ {
		if entry.QueryTerms[i] == action {
			break
		}
	}

	if i == len(entry.QueryTerms) {
		// action not present, nothing to remove
		return
	}

	entry.QueryTerms = append(entry.QueryTerms[:i], entry.QueryTerms[i+1:]...)
}

// Authorizations defines the list of authorizations.
type Authorizations []*Authorization

// Find search for an entry and return nil if not found.
func (e Authorizations) Find(userID string) *Authorization {
	for _, entry := range e {
		if entry.UserID == userID {
			return entry
		}
	}

	return nil
}

// String produces a text representation of Authorizations
func (e Authorizations) String() string {
	out := new(strings.Builder)

	for i, entry := range e {
		fmt.Fprintf(out, "- authorization %d:\n%s", i, entry)
	}

	return out.String()
}

// Authorization defines the query terms that a user is allowed to execute.
type Authorization struct {
	UserID     string
	QueryTerms []string
}

// IsAllowed checks if the action is present in the entry.
// TODO: here we should check that at least on element of the query definition,
// Like (Q1 AND Q2) OR Q3 is allowed.
// I'm not sure yet of the format, nor how to parse it.
func (e Authorization) IsAllowed(queryDefinition string) bool {
	for _, action := range e.QueryTerms {
		if action == queryDefinition {
			return true
		}
	}

	return false
}

// String produces a text representation of an Authorization.
func (e Authorization) String() string {
	out := new(strings.Builder)

	fmt.Fprintf(out, "- UserID: %s\n", e.UserID)
	fmt.Fprintf(out, "- QueryTerms: %v\n", e.QueryTerms)

	return out.String()
}
