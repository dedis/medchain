package medchain

import (
	"errors"

	"go.dedis.ch/cothority/v3/byzcoin"
	"go.dedis.ch/cothority/v3/darc"
	"go.dedis.ch/protobuf"
	"golang.org/x/xerrors"
)

// The query contract can simply store a query in an instance.

// MedchainContractID denotes a contract that can store and update
// key/value pairs corresponding to queries. Key is the query ID
// and Status is the query Status (i.e., it is the concatenation
// of query/status/user)
var MedchainContractID = "medchain"

type medchainContract struct {
	byzcoin.BasicContract
	QueryData
}

// ContractMedchainFromBytes defines the contract
func ContractMedchainFromBytes(in []byte) (byzcoin.Contract, error) {
	cv := &medchainContract{}
	err := protobuf.Decode(in, &cv.QueryData)
	if err != nil {
		return nil, err
	}
	return cv, nil
}

// medchainContract implments the main logic of medchian node
// It is a key/value store type contract that manipulates queries
// received from the client (e.g., medco-connector) and writes to
// Byzcoin "instances".
// This contract implements 2 main functionalities:
// (1) Spawn new key-value instances of queries and store all the arguments in the data field.
// (2) Update existing key-value instances.
func (c *medchainContract) Spawn(rst byzcoin.ReadOnlyStateTrie, inst byzcoin.Instruction, coins []byzcoin.Coin) (sc []byzcoin.StateChange, cout []byzcoin.Coin, err error) {
	cout = coins

	// Find the darcID for this instance
	var darcID darc.ID
	_, _, _, darcID, err = rst.GetValues(inst.InstanceID.Slice())
	if err != nil {
		return nil, nil, xerrors.Errorf("failed to get the darc ID: %v", err)
	}

	// Put the data received in the inst.Spawn.Args into our QueryData structure.
	cs := &c.QueryData
	for _, kv := range inst.Spawn.Args {
		cs.Storage = append(cs.Storage, Query{kv.Name, kv.Value})
	}

	// Encode the data into our QueryDataStorage structure that holds all the key-value pairs
	csBuf, err := protobuf.Encode(&c.QueryData)
	if err != nil {
		return nil, nil, xerrors.Errorf("failed to encode QueryDataStorage: %v", err)
	}

	// Then create a StateChange request with the data of the instance. The
	// InstanceID is given by the DeriveID method of the instruction that allows
	// to create multiple instanceIDs out of a given instruction in a pseudo-
	// random way that will be the same for all nodes.
	sc = []byzcoin.StateChange{
		byzcoin.NewStateChange(byzcoin.Create, inst.DeriveID(""), MedchainContractID, csBuf, darcID),
	}
	return sc, cout, nil
}

// Invoke implements update and the database funtions (i.e., the user queries)
// It is also possible to have the queries not implmented as functions here.
// However, they have to be added to the project darcs.
func (c *medchainContract) Invoke(rst byzcoin.ReadOnlyStateTrie, inst byzcoin.Instruction, coins []byzcoin.Coin) (sc []byzcoin.StateChange, cout []byzcoin.Coin, err error) {
	cout = coins

	//TODO: add the darcs and  check for approval
	var darcID darc.ID
	_, _, _, darcID, err = rst.GetValues(inst.InstanceID.Slice())
	if err != nil {
		return nil, nil, xerrors.Errorf("failed to authorize the query with error:", err)
	}

	if inst.Invoke.Command != "update" && inst.Invoke.Command != "verifystatus" && inst.Invoke.Command != "patient_list" && inst.Invoke.Command != "count_per_site" && inst.Invoke.Command != "count_per_site_obfuscated" && inst.Invoke.Command != "count_per_site_shuffled" && inst.Invoke.Command != "count_per_site_shuffled_obfuscated" && inst.Invoke.Command != "count_global" && inst.Invoke.Command != "count_global_obfuscated" {
		return nil, nil, errors.New("This action is not supported by contract" + ContractName)
	}

	switch inst.Invoke.Command {
	// One of the commands we can invoke is 'update' which will store the new values
	// given in the arguments in the data.
	//  1. decode the existing data
	//  2. update the data
	//  3. encode the data into protobuf again
	case "update":
		kvd := &c.QueryData
		kvd.Update(inst.Invoke.Args)
		var buf []byte
		buf, err = protobuf.Encode(kvd)
		if err != nil {
			return nil, nil, xerrors.Errorf("failed to encode data with error : %v", err)
		}
		sc = []byzcoin.StateChange{
			byzcoin.NewStateChange(byzcoin.Update, inst.InstanceID,
				MedchainContractID, buf, darcID),
		}
	case "verifystatus":
		kvd := &c.QueryData
		err := kvd.VerifyStatus(inst.Invoke.Args)
		if err != nil {
			return nil, nil, xerrors.Errorf("failed to verify the query status with error: %v", err)
		}
		var buf []byte
		buf, err = protobuf.Encode(kvd)
		if err != nil {
			return nil, nil, xerrors.Errorf("failed to encode data with error : %v", err)
		}
		sc = []byzcoin.StateChange{
			byzcoin.NewStateChange(byzcoin.Create, inst.InstanceID,
				MedchainContractID, buf, darcID),
		}

	case "patient_list":
		kvd := &c.QueryData
		kvd.Update(inst.Invoke.Args)
		var buf []byte
		buf, err = protobuf.Encode(kvd)
		if err != nil {
			return nil, nil, xerrors.Errorf("failed to encode data with error : %v", err)
		}
		sc = []byzcoin.StateChange{
			byzcoin.NewStateChange(byzcoin.Update, inst.InstanceID,
				MedchainContractID, buf, darcID),
		}

	case "count_per_site":
		kvd := &c.QueryData
		kvd.Update(inst.Invoke.Args)
		var buf []byte
		buf, err = protobuf.Encode(kvd)
		if err != nil {
			return nil, nil, xerrors.Errorf("failed to encode data with error : %v", err)
		}
		sc = []byzcoin.StateChange{
			byzcoin.NewStateChange(byzcoin.Update, inst.InstanceID,
				MedchainContractID, buf, darcID),
		}

	case "count_per_site_obfuscated":
		kvd := &c.QueryData
		kvd.Update(inst.Invoke.Args)
		var buf []byte
		buf, err = protobuf.Encode(kvd)
		if err != nil {
			return nil, nil, xerrors.Errorf("failed to encode data with error : %v", err)
		}
		sc = []byzcoin.StateChange{
			byzcoin.NewStateChange(byzcoin.Update, inst.InstanceID,
				MedchainContractID, buf, darcID),
		}

	case "count_per_site_shuffled":
		kvd := &c.QueryData
		kvd.Update(inst.Invoke.Args)
		var buf []byte
		buf, err = protobuf.Encode(kvd)
		if err != nil {
			return nil, nil, xerrors.Errorf("failed to encode data with error : %v", err)
		}
		sc = []byzcoin.StateChange{
			byzcoin.NewStateChange(byzcoin.Update, inst.InstanceID,
				MedchainContractID, buf, darcID),
		}

	case "count_per_site_shuffled_obfuscated":
		kvd := &c.QueryData
		kvd.Update(inst.Invoke.Args)
		var buf []byte
		buf, err = protobuf.Encode(kvd)
		if err != nil {
			return nil, nil, xerrors.Errorf("failed to encode data with error : %v", err)
		}
		sc = []byzcoin.StateChange{
			byzcoin.NewStateChange(byzcoin.Update, inst.InstanceID,
				MedchainContractID, buf, darcID),
		}

	case "count_global":
		kvd := &c.QueryData
		kvd.Update(inst.Invoke.Args)
		var buf []byte
		buf, err = protobuf.Encode(kvd)
		if err != nil {
			return nil, nil, xerrors.Errorf("failed to encode data with error : %v", err)
		}
		sc = []byzcoin.StateChange{
			byzcoin.NewStateChange(byzcoin.Update, inst.InstanceID,
				MedchainContractID, buf, darcID),
		}

	case "count_global_obfuscated":
		kvd := &c.QueryData
		kvd.Update(inst.Invoke.Args)
		var buf []byte
		buf, err = protobuf.Encode(kvd)
		if err != nil {
			return nil, nil, xerrors.Errorf("failed to encode data with error : %v", err)
		}
		sc = []byzcoin.StateChange{
			byzcoin.NewStateChange(byzcoin.Update, inst.InstanceID,
				MedchainContractID, buf, darcID),
		}

		return sc, cout, nil

	}

	return
}

// Update goes through all the arguments and:
//  - updates the value if the key already exists
//  - deletes the key-value pair if the value is empty (??)
//  - adds a new key-value pair if the key does not exist yet
func (cs *QueryData) Update(args byzcoin.Arguments) {
	for _, kv := range args {
		var updated bool
		for i, stored := range cs.Storage {
			if stored.ID == kv.Name {
				updated = true
				if kv.Value == nil || len(kv.Value) == 0 {
					cs.Storage = append(cs.Storage[0:i], cs.Storage[i+1:]...)
					break
				}
				cs.Storage[i].Status = kv.Value
			}

		}
		if !updated {
			cs.Storage = append(cs.Storage, Query{kv.Name, kv.Value})
		}
	}
}

//VerifyStatus goes through all the arguments and:
//- if found: returns the status of the query off the ledger
//- if not found: returns nil
func (cs *QueryData) VerifyStatus(args byzcoin.Arguments) (err error) {
	for _, kv := range args {
		var found bool
		for _, stored := range cs.Storage {
			if stored.ID == kv.Name {
				found = true
				if string(stored.Status) == "Authorized" {
					return nil
				}
				return xerrors.Errorf("query %s has status %s and has not been authorized", stored.ID, stored.Status)

			}

		}
		if !found {
			return xerrors.Errorf("could not find the query with ID %s", kv.Name)
		}

	}
	return
}

// VerifyDeferredInstruction implements the byzcoin.Contract interface
// This should be implemented to be able to use the contract with deferred contract
func (c medchainContract) VerifyDeferredInstruction(rst byzcoin.ReadOnlyStateTrie, inst byzcoin.Instruction, ctxHash []byte) error {
	return inst.VerifyWithOption(rst, ctxHash, &byzcoin.VerificationOptions{IgnoreCounters: true})
}
