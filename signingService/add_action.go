package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/DPPH/MedChain/medChainUtils"
	"github.com/DPPH/MedChain/signingService/db_handler"
	"github.com/DPPH/MedChain/signingService/signing_messages"
)

func addAction(w http.ResponseWriter, r *http.Request) {
	fmt.Println("/add/action")
	body, err := ioutil.ReadAll(r.Body)
	if medChainUtils.CheckError(err, w, r) {
		return
	}
	var request signing_messages.AddNewActionRequest
	err = json.Unmarshal(body, &request)
	if medChainUtils.CheckError(err, w, r) {
		return
	}
	if request.Action == nil {
		medChainUtils.CheckError(errors.New("You need to provide an action to add"), w, r)
		return
	}
	id, err := db_handler.RegisterNewAction(request.Action, db)
	if medChainUtils.CheckError(err, w, r) {
		return
	}
	reply := signing_messages.AddNewActionReply{Id: id}
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
