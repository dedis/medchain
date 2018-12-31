package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"

	"github.com/DPPH/MedChain/medChainUtils"
	"github.com/DPPH/MedChain/signingService/db_handler"
	"github.com/DPPH/MedChain/signingService/signing_messages"
	"github.com/DPPH/MedChain/signingService/status"
)

func DenyAction(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if medChainUtils.CheckError(err, w, r) {
		return
	}
	var request signing_messages.ActionUpdate
	err = json.Unmarshal(body, &request)
	if medChainUtils.CheckError(err, w, r) {
		return
	}
	err = CheckForDenial(&request)
	if medChainUtils.CheckError(err, w, r) {
		return
	}
	err = UpdateDatabaseWithDenial(&request)
	if medChainUtils.CheckError(err, w, r) {
		return
	}
	reply, err := db_handler.GetInfoForAction(request.ActionId, db)
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

func UpdateDatabaseWithDenial(request *signing_messages.ActionUpdate) error {

	err := db_handler.UdpdateActionStatus(request.ActionId, status.Denied, db)
	if err != nil {
		return err
	}

	signature_map, err := db_handler.GetActionSignatureMap(request.ActionId, db)
	if err != nil {
		return err
	}

	for signer_id, signer_status := range signature_map {
		if signer_status == status.SignerWaiting {
			if signer_id == request.SignerId {
				err = db_handler.UdpdateSignatureStatus(request.ActionId, signer_id, status.SignerDenied, db)
			} else {
				err = db_handler.UdpdateSignatureStatus(request.ActionId, signer_id, status.SignerNA, db)
			}
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func CheckForDenial(request *signing_messages.ActionUpdate) error {

	action_status, err := db_handler.GetActionStatus(request.ActionId, db)
	if err == sql.ErrNoRows {
		return errors.New("No Action With Such Id")
	} else if err != nil {
		return err
	}

	if action_status != status.Waiting {
		return errors.New("The action is not waiting for approval")
	}

	signature_map, err := db_handler.GetActionSignatureMap(request.ActionId, db)
	if err != nil {
		return err
	}
	status_value, ok := signature_map[request.SignerId]
	if !ok || status_value != status.SignerWaiting {
		return errors.New("The provided signer is not needed for approval")
	}
	return nil
}
