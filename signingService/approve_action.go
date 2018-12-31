package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"

	"github.com/DPPH/MedChain/medChainAdmin/admin_messages"
	"github.com/DPPH/MedChain/medChainServer/messages"
	"github.com/DPPH/MedChain/medChainUtils"
	"github.com/DPPH/MedChain/signingService/db_handler"
	"github.com/DPPH/MedChain/signingService/status"
)

func ApproveAction(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if medChainUtils.CheckError(err, w, r) {
		return
	}
	var request admin_messages.SignReply
	err = json.Unmarshal(body, &request)
	if medChainUtils.CheckError(err, w, r) {
		return
	}
	err = CheckForApproval(&request)
	if medChainUtils.CheckError(err, w, r) {
		return
	}
	err = UpdateDatabaseWithApproval(&request)
	if medChainUtils.CheckError(err, w, r) {
		return
	}
	reply, err := db_handler.GetInfoForAction(request.ActionInfo.Id, db)
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

func UpdateDatabaseWithApproval(request *admin_messages.SignReply) error {
	new_action := *(request.ActionInfo.Action)
	// fmt.Println("Old Transaction :" + new_action.Transaction)
	new_action.Transaction = request.SignedTransaction
	// fmt.Println("New Transaction :" + new_action.Transaction)
	new_action_string, err := actionReplyToString(&new_action)
	if err != nil {
		return err
	}
	// fmt.Println("Action String :" + new_action_string)
	err = db_handler.UpdateActionValue(request.ActionInfo.Id, new_action_string, db)
	if err != nil {
		return err
	}
	err = db_handler.UdpdateSignatureStatus(request.ActionInfo.Id, request.SignerId, status.Approved, db)
	if err != nil {
		return err
	}
	signature_map, err := db_handler.GetActionSignatureMap(request.ActionInfo.Id, db)
	if err != nil {
		return err
	}
	approved := true
	for _, status_val := range signature_map {
		if status_val != status.SignerApproved {
			approved = false
			break
		}
	}
	if approved {
		err = db_handler.UdpdateActionStatus(request.ActionInfo.Id, status.Approved, db)
		if err != nil {
			return err
		}
	}
	return nil
}

func actionReplyToString(action *messages.ActionReply) (string, error) {
	action_bytes, err := json.Marshal(action)
	if err != nil {
		return "", err
	}
	action_string := string(action_bytes)
	return action_string, nil
}

func CheckForApproval(request *admin_messages.SignReply) error {
	if request.ActionInfo == nil {
		return errors.New("You need to provide the information of the action to approve")
	}

	action_status, err := db_handler.GetActionStatus(request.ActionInfo.Id, db)
	if err == sql.ErrNoRows {
		return errors.New("No Action With Such Id")
	} else if err != nil {
		return err
	}

	if action_status != status.Waiting {
		return errors.New("The action is not waiting for approval")
	}

	local_action_string, err := db_handler.GetActionString(request.ActionInfo.Id, db)
	if err == sql.ErrNoRows {
		return errors.New("No Action With Such Id")
	} else if err != nil {
		return err
	}

	request_action_string, err := actionReplyToString(request.ActionInfo.Action)
	if err != nil {
		return err
	}
	if request_action_string != local_action_string {
		return errors.New("The action sent doesn't match the action stored")
	}
	signature_map, err := db_handler.GetActionSignatureMap(request.ActionInfo.Id, db)
	if err != nil {
		return err
	}
	status_value, ok := signature_map[request.SignerId]
	if !ok || status_value != status.SignerWaiting {
		return errors.New("The provided signer is not needed for approval")
	}
	return nil
}
