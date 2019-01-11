package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/DPPH/MedChain/medChainServer/messages"
	"github.com/DPPH/MedChain/medChainServer/metadata"
	"github.com/DPPH/MedChain/medChainUtils"
	"github.com/dedis/cothority/omniledger/darc"
	"github.com/dedis/cothority/omniledger/service"
)

/**
This function is called by the CancelAction entry point when the given action
	is to add a new project.
It takes care of cleaning the metadata to erase the effects of the action.
**/
func cancelNewProject(w http.ResponseWriter, r *http.Request, transaction_string string) {

	transaction, err := extractTransactionFromString(transaction_string)
	if medChainUtils.CheckError(err, w, r) {
		return
	}

	project_darc, _, _, err := checkTransactionForNewProject(transaction)
	if medChainUtils.CheckError(err, w, r) {
		return
	}
	err = CancelAndRemoveProjectFromMetadata(project_darc)
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

func CancelAndRemoveProjectFromMetadata(project_darc *darc.Darc) error {
	project_darc_base_id := medChainUtils.IDToB64String(project_darc.GetBaseID())
	project_metadata, ok := metaData.ProjectsWaitingForCreation[project_darc_base_id]
	if !ok {
		return errors.New("The commited project needs to be added first")
	}
	if project_metadata.IsCreated {
		errors.New("The project was already created")
	}
	for _, user_metadata := range project_metadata.Users {
		delete(user_metadata.Projects, project_metadata.Name)
	}
	for _, manager_metadata := range project_metadata.Managers {
		delete(manager_metadata.Projects, project_metadata.Name)
		for _, admin_metadata := range manager_metadata.Hospital.Admins {
			delete(admin_metadata.Projects, project_metadata.Name)
		}
		delete(manager_metadata.Hospital.SuperAdmin.Projects, project_metadata.Name)
	}
	delete(metaData.Projects, project_metadata.Name)
	delete(metaData.ProjectsWaitingForCreation, project_darc_base_id)
	return nil
}

/**
This function is called by the CommitAction entry point when the given action
	is to add a new project.
It takes care of submitting the transaction, checking that it has been accepted, and adapt the metadata.
**/
func CommitProject(w http.ResponseWriter, r *http.Request, transaction_string string) {
	start := time.Now()
	reply, err := subFunctionCommitProject(transaction_string)
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
	fmt.Printf("Time to commit new project : %s\n", elapsed.String())
}

func subFunctionCommitProject(transaction_string string) (*messages.ProjectInfoReply, error) {
	transaction, err := extractTransactionFromString(transaction_string)
	if err != nil {
		return nil, err
	}

	project_darc, project_list_bytes, user_bytes, err := checkTransactionForNewProject(transaction)
	if err != nil {
		return nil, err
	}

	err = submitTransactionForNewProject(transaction, project_darc, project_list_bytes, user_bytes)
	if err != nil {
		return nil, err
	}

	project_metadata, err := adaptMetadataForNewProject(project_darc)
	if err != nil {
		return nil, err
	}
	reply := projectMetadataToInfoReply(project_metadata)
	return &reply, nil
}

func checkTransactionForNewProject(transaction *service.ClientTransaction) (*darc.Darc, []byte, []byte, error) {
	if len(transaction.Instructions) < 3 {
		return nil, nil, nil, errors.New("Not enough instructions")
	}
	project_darc, err := checkSpawnDarcInstructionForGenericUser(transaction.Instructions[0])
	if err != nil {
		return nil, nil, nil, err
	}
	project_darc_base_id := medChainUtils.IDToB64String(project_darc.GetBaseID())
	project_metadata, ok := metaData.ProjectsWaitingForCreation[project_darc_base_id]
	if !ok {
		return nil, nil, nil, errors.New("The commited project needs to be added first")
	}
	project_list_bytes, err := checkUpdateProjectListInstruction(transaction.Instructions[1], project_darc)
	if err != nil {
		return nil, nil, nil, err
	}
	user_map_bytes, err := checkUpdateUserProjectMapInstruction(transaction.Instructions[2], project_metadata)
	if err != nil {
		return nil, nil, nil, err
	}
	return project_darc, project_list_bytes, user_map_bytes, nil
}

func checkUpdateProjectListInstruction(instruction service.Instruction, project_darc *darc.Darc) ([]byte, error) {
	if instruction.InstanceID != metaData.AllProjectsListInstanceID {
		return nil, errors.New("Not updating the right project list instance")
	}
	invoke := instruction.Invoke
	if invoke == nil {
		return nil, errors.New("First instruction wasn't an invoke")
	}
	if invoke.Command != "update" {
		return nil, errors.New("Invoke instruction wasn't invoke:update")
	}
	args := invoke.Args
	if len(args) < 1 {
		return nil, errors.New("Not enough arguments in the invoke:update instruction")
	}
	arg := args[0]
	if arg.Name != "value" {
		return nil, errors.New("The first argument wasn't the project list value")
	}
	project_list_byte := arg.Value
	test_project_list_byte, err := getUpdatedProjectListBytes(project_darc)
	if err != nil {
		return nil, err
	}
	if string(project_list_byte) != string(test_project_list_byte) {
		return nil, errors.New("The updated project list is not consistent")
	}
	return project_list_byte, nil
}

func checkUpdateUserProjectMapInstruction(instruction service.Instruction, project_metadata *metadata.Project) ([]byte, error) {
	if instruction.InstanceID != metaData.UserProjectsMapInstanceID {
		return nil, errors.New("Not updating the right user project map instance")
	}
	invoke := instruction.Invoke
	if invoke == nil {
		return nil, errors.New("First instruction wasn't an invoke")
	}
	if invoke.Command != "update" {
		return nil, errors.New("Invoke instruction wasn't invoke:update")
	}
	args := invoke.Args
	if len(args) < 2 {
		return nil, errors.New("Not enough arguments in the invoke:update instruction")
	}
	arg := args[0]
	if arg.Name != "allProjectsListInstanceID" {
		return nil, errors.New("The first argument wasn't the project list id")
	}
	project_list_id := arg.Value
	test_id := string([]byte(metaData.AllProjectsListInstanceID.Slice()))
	if string(project_list_id) != test_id {
		return nil, errors.New("The given project list id in the new map is incorrect")
	}
	arg = args[1]
	if arg.Name != "users" {
		return nil, errors.New("The second argument wasn't the new users bytes")
	}
	user_bytes := arg.Value
	test_bytes := string(getUpdatedUserMapBytes(project_metadata.Users))
	if string(user_bytes) != test_bytes {
		return nil, errors.New("The user_bytes do not correspond to the new users")
	}

	return user_bytes, nil
}

func submitTransactionForNewProject(transaction *service.ClientTransaction, project_darc *darc.Darc, project_list_bytes, user_bytes []byte) error {

	// Commit transaction
	if _, err := cl.AddTransaction(*transaction); err != nil {
		return err
	}

	// // Verify DARC creation
	instID := service.NewInstanceID(project_darc.GetBaseID())
	pr, err := cl.WaitProof(instID, time.Duration(20)*metaData.GenesisMsg.BlockInterval, nil)
	if err != nil || pr.InclusionProof.Match() == false {
		return errors.New("Could not get proof of the darc creation")
	}

	// Verify project list is updated
	instID = metaData.AllProjectsListInstanceID

	pr, err = cl.WaitProof(instID, metaData.GenesisMsg.BlockInterval, project_list_bytes)
	if err != nil || pr.InclusionProof.Match() == false {
		return errors.New("Could not get proof of the project list update")
	}

	return nil
}

func adaptMetadataForNewProject(project_darc *darc.Darc) (*metadata.Project, error) {
	project_darc_base_id := medChainUtils.IDToB64String(project_darc.GetBaseID())
	project_metadata, ok := metaData.ProjectsWaitingForCreation[project_darc_base_id]
	if !ok {
		return nil, errors.New("The commited project needs to be added first")
	}
	project_metadata.DarcBaseId = addDarcToMaps(project_darc, metaData)
	project_metadata.IsCreated = true
	delete(metaData.ProjectsWaitingForCreation, project_darc_base_id)
	return project_metadata, nil
}
