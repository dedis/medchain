package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/DPPH/MedChain/medChainUtils"
	"github.com/DPPH/MedChain/signingService/db_handler"
	"github.com/DPPH/MedChain/signingService/signing_messages"
)

func getUserActions(w http.ResponseWriter, r *http.Request) {
	getActionList(w, r, true)
}

func getActionsWaiting(w http.ResponseWriter, r *http.Request) {
	getActionList(w, r, false)
}

func getActionList(w http.ResponseWriter, r *http.Request, is_initiator bool) {
	body, err := ioutil.ReadAll(r.Body)
	if medChainUtils.CheckError(err, w, r) {
		return
	}
	var request signing_messages.ListRequest
	err = json.Unmarshal(body, &request)
	if medChainUtils.CheckError(err, w, r) {
		return
	}
	if request.Id == "" {
		medChainUtils.CheckError(errors.New("You need to provide an id"), w, r)
		return
	}
	fmt.Println("list for :" + request.Id)
	var list_action_info []*signing_messages.ActionInfoReply
	if is_initiator {
		list_action_info, err = db_handler.GetActionsInitiatedBy(request.Id, db)
		if medChainUtils.CheckError(err, w, r) {
			return
		}
	} else {
		list_action_info, err = db_handler.GetActionsWaitingFor(request.Id, db)
		if medChainUtils.CheckError(err, w, r) {
			return
		}
	}

	reply := signing_messages.ListReply{Actions: list_action_info}
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

func getActionInfo(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if medChainUtils.CheckError(err, w, r) {
		return
	}
	var request signing_messages.ActionInfoRequest
	err = json.Unmarshal(body, &request)
	if medChainUtils.CheckError(err, w, r) {
		return
	}
	if request.Id == "" {
		medChainUtils.CheckError(errors.New("You need to provide an id"), w, r)
		return
	}

	action_info, err := db_handler.GetInfoForAction(request.Id, db)
	if err == sql.ErrNoRows {
		medChainUtils.CheckError(errors.New("No action has this id"), w, r)
		return
	} else if err != nil {
		medChainUtils.CheckError(err, w, r)
		return
	}

	json_val, err := json.Marshal(action_info)
	if medChainUtils.CheckError(err, w, r) {
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(json_val)
	if medChainUtils.CheckError(err, w, r) {
		return
	}
}
