package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/DPPH/MedChain/medChainServer/messages"
	"github.com/DPPH/MedChain/medChainServer/metadata"
	"github.com/DPPH/MedChain/medChainUtils"
	"github.com/dedis/cothority"
	"github.com/dedis/cothority/omniledger/darc"
	"github.com/dedis/cothority/omniledger/service"
	"github.com/dedis/onet/network"
)

/**
This file takes care of committing or cancelling
	actions that add new generic users (admin, manager, user) to the system
**/

/**
This function is called by the CancelAction entry point when the given action
	is to add a new generic user.
It takes care of cleaning the metadata to erase the effects of the action.
**/
func cancelNewGenericUser(w http.ResponseWriter, r *http.Request, transaction_string, user_type string) {

	transaction, err := extractTransactionFromString(transaction_string)
	if medChainUtils.CheckError(err, w, r) {
		return
	}
	new_darc, evolved_darc, err := checkTransactionForNewGenericUser(transaction, user_type)
	if medChainUtils.CheckError(err, w, r) {
		return
	}
	err = CancelAndRemoveUserFromMetadata(new_darc, evolved_darc, user_type)
	if medChainUtils.CheckError(err, w, r) {
		return
	}
	reply := messages.CommitRequest{Transaction: transaction_string}
	json_val, err := json.Marshal(&reply)
	if medChainUtils.CheckError(err, w, r) {
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(json_val)
	if medChainUtils.CheckError(err, w, r) {
		return
	}
}

func CancelAndRemoveUserFromMetadata(new_darc, evolved_darc *darc.Darc, user_type string) error {
	new_darc_base_id := medChainUtils.IDToB64String(new_darc.GetBaseID())
	user_metadata, ok := metaData.WaitingForCreation[new_darc_base_id]
	if !ok {
		return errors.New("Could not find the metadata of the new user")
	}
	if user_metadata.IsCreated {
		return errors.New("This user was already created")
	}
	hospital_metadata := user_metadata.Hospital
	switch user_type {
	case "Admin":
		hospital_metadata.Admins = removeUserMetadataFromList(hospital_metadata.Admins, user_metadata.Id.String())
	case "Manager":
		hospital_metadata.Managers = removeUserMetadataFromList(hospital_metadata.Managers, user_metadata.Id.String())
	case "User":
		hospital_metadata.Users = removeUserMetadataFromList(hospital_metadata.Users, user_metadata.Id.String())
	default:
		return errors.New("Could not find the metadata of the new user")
	}
	delete(metaData.GenericUsers, user_metadata.Id.String())
	delete(metaData.WaitingForCreation, new_darc_base_id)
	return nil
}

func removeUserMetadataFromList(user_list []*metadata.GenericUser, user_id string) []*metadata.GenericUser {
	new_list := []*metadata.GenericUser{}
	for _, user := range user_list {
		if user.Id.String() != user_id {
			new_list = append(new_list, user)
		}
	}
	return new_list
}

/**
This function is called by the CommitAction entry point when the given action
	is to add a new generic user.
It takes care of submitting the transaction, checking that it has been accepted, and adapt the metadata.
**/
func commitNewGenericUserToChain(w http.ResponseWriter, r *http.Request, transaction_string, user_type string) {
	start := time.Now()
	reply, err := subFunctionCommitGenericUser(transaction_string, user_type)
	if medChainUtils.CheckError(err, w, r) {
		return
	}
	json_val, err := json.Marshal(reply)
	if medChainUtils.CheckError(err, w, r) {
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(json_val)
	if medChainUtils.CheckError(err, w, r) {
		return
	}
	elapsed := time.Since(start)
	fmt.Printf("Time to commit new %s : %s\n", user_type, elapsed.String())
}

func subFunctionCommitGenericUser(transaction_string, user_type string) (*messages.GenericUserInfoReply, error) {
	transaction, err := extractTransactionFromString(transaction_string)
	if err != nil {
		return nil, err
	}
	new_darc, evolved_darc, err := checkTransactionForNewGenericUser(transaction, user_type)
	if err != nil {
		return nil, err
	}
	user_metadata, err := adaptMetadata(new_darc, evolved_darc, user_type)
	if err != nil {
		return nil, err
	}
	err = submitTransactionForNewGenericUser(transaction, new_darc, evolved_darc)
	if err != nil {
		return nil, err
	}

	reply := messages.GenericUserInfoReply{Id: user_metadata.Id.String(), Name: user_metadata.Name, DarcBaseId: user_metadata.DarcBaseId, SuperAdminId: user_metadata.Hospital.SuperAdmin.Id.String(), IsCreated: user_metadata.IsCreated}
	return &reply, nil
}

func adaptMetadata(new_darc, evolved_darc *darc.Darc, user_type string) (*metadata.GenericUser, error) {
	new_darc_base_id := medChainUtils.IDToB64String(new_darc.GetBaseID())
	user_metadata, ok := metaData.WaitingForCreation[new_darc_base_id]
	if !ok {
		return nil, errors.New("Could not find the metadata of the new user")
	}
	if user_metadata.IsCreated {
		return nil, errors.New("This user was already created")
	}
	hospital_metadata := user_metadata.Hospital
	evolved_darc_base_id := medChainUtils.IDToB64String(evolved_darc.GetBaseID())
	var test_base_id string
	switch user_type {
	case "Admin":
		test_base_id = hospital_metadata.AdminListDarcBaseId
	case "Manager":
		test_base_id = hospital_metadata.ManagerListDarcBaseId
	case "User":
		test_base_id = hospital_metadata.UserListDarcBaseId
	default:
		return nil, errors.New("Could not find the metadata of the new user")
	}
	if test_base_id != evolved_darc_base_id {
		return nil, errors.New("The evolved darc doesn't correspond to the hospital list")
	}
	user_metadata.DarcBaseId = addDarcToMaps(new_darc, metaData)
	user_metadata.IsCreated = true
	if user_type == "Admin" {
		for name, project := range user_metadata.Hospital.SuperAdmin.Projects {
			user_metadata.Projects[name] = project
		}
	}
	addDarcToMaps(evolved_darc, metaData)
	delete(metaData.WaitingForCreation, new_darc_base_id)
	return user_metadata, nil
}

func checkTransactionForNewGenericUser(transaction *service.ClientTransaction, user_type string) (*darc.Darc, *darc.Darc, error) {
	if len(transaction.Instructions) < 2 {
		return nil, nil, errors.New("Not enough instructions")
	}
	new_darc, err := checkSpawnDarcInstructionForGenericUser(transaction.Instructions[0])
	if err != nil {
		return nil, nil, err
	}
	evolved_darc, err := checkEvolveDarcInstructionForGenericUser(transaction.Instructions[1])
	if err != nil {
		return nil, nil, err
	}
	new_darc_base_id := medChainUtils.IDToB64String(new_darc.GetBaseID())
	user_metadata, ok := metaData.WaitingForCreation[new_darc_base_id]
	if !ok {
		return nil, nil, errors.New("Could not find the metadata of the new user")
	}
	if user_metadata.IsCreated {
		return nil, nil, errors.New("This user was already created")
	}
	hospital_metadata := user_metadata.Hospital
	evolved_darc_base_id := medChainUtils.IDToB64String(evolved_darc.GetBaseID())
	var test_base_id string
	switch user_type {
	case "Admin":
		test_base_id = hospital_metadata.AdminListDarcBaseId
	case "Manager":
		test_base_id = hospital_metadata.ManagerListDarcBaseId
	case "User":
		test_base_id = hospital_metadata.UserListDarcBaseId
	default:
		return nil, nil, errors.New("Could not find the metadata of the new user")
	}
	if test_base_id != evolved_darc_base_id {
		return nil, nil, errors.New("The evolved darc doesn't correspond to the hospital list")
	}
	return new_darc, evolved_darc, nil
}

func submitTransactionForNewGenericUser(transaction *service.ClientTransaction, new_darc, evolved_darc *darc.Darc) error {
	// Commit transaction

	if _, err := cl.AddTransaction(*transaction); err != nil {
		return err
	}

	// // Verify DARC creation
	instID := service.NewInstanceID(new_darc.GetBaseID())
	pr, err := cl.WaitProof(instID, time.Duration(20)*metaData.GenesisMsg.BlockInterval, nil)
	if err != nil || pr.InclusionProof.Match() == false {
		return errors.New("Could not get proof of the darc creation")
	}

	// Verify DARC evolution
	instID = service.NewInstanceID(evolved_darc.GetBaseID())
	// darcBuf, err := evolved_darc.ToProto()
	// if err != nil {
	// 	return err
	// }
	pr, err = cl.WaitProof(instID, metaData.GenesisMsg.BlockInterval, nil)
	if err != nil || pr.InclusionProof.Match() == false {
		return errors.New("Could not get proof of the darc evolution")
	}
	return nil
}

func checkSpawnDarcInstructionForGenericUser(instruction service.Instruction) (*darc.Darc, error) {
	spawn := instruction.Spawn
	if spawn == nil {
		return nil, errors.New("First instruction wasn't a spawn")
	}
	if spawn.ContractID != service.ContractDarcID {
		return nil, errors.New("Spawn instruction wasn't spawn:darc")
	}
	args := spawn.Args
	if len(args) < 1 {
		return nil, errors.New("Not enough arguments in the spawn:darc instruction")
	}
	arg := args[0]
	if arg.Name != "darc" {
		return nil, errors.New("The first argument wasn't the darc value")
	}
	darcBuf := arg.Value
	newDarc, err := darc.NewFromProtobuf(darcBuf)
	if err != nil {
		return nil, errors.New("Could not retrieve the darc from the buffer")
	}
	return newDarc, nil
}

func checkEvolveDarcInstructionForGenericUser(instruction service.Instruction) (*darc.Darc, error) {
	invoke := instruction.Invoke
	if invoke == nil {
		return nil, errors.New("First instruction wasn't an invoke")
	}
	if invoke.Command != "evolve" {
		return nil, errors.New("Invoke instruction wasn't invoke:evolve")
	}
	args := invoke.Args
	if len(args) < 1 {
		return nil, errors.New("Not enough arguments in the invoke:evolve instruction")
	}
	arg := args[0]
	if arg.Name != "darc" {
		return nil, errors.New("The first argument wasn't the darc value")
	}
	darcBuf := arg.Value
	evolved_darc, err := darc.NewFromProtobuf(darcBuf)
	if err != nil {
		return nil, errors.New("Could not retrieve the darc from the buffer")
	}
	return evolved_darc, nil
}

func extractTransactionFromString(transaction_string string) (*service.ClientTransaction, error) {
	transaction_bytes, err := base64.StdEncoding.DecodeString(transaction_string)
	if err != nil {
		return nil, err
	}
	// Load the transaction
	var transaction *service.ClientTransaction
	_, tmp, err := network.Unmarshal(transaction_bytes, cothority.Suite)
	if err != nil {
		return nil, err
	}
	transaction, ok := tmp.(*service.ClientTransaction)
	if !ok {
		return nil, errors.New("Could not retrieve the transaction")
	}
	return transaction, nil
}
