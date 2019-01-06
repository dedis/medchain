package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/DPPH/MedChain/medChainServer/messages"
	"github.com/DPPH/MedChain/medChainUtils"
)

/**
This file is used as a switch for committing or cancelling different types of actions.
It reads the action type and the transaction and forwards to the right function.
It should receive a messages.CommitRequest (encoded in json) in the body of the http request r
**/

func CommitAction(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if medChainUtils.CheckError(err, w, r) {
		return
	}
	var request messages.CommitRequest
	err = json.Unmarshal(body, &request)
	if medChainUtils.CheckError(err, w, r) {
		fmt.Println("Can't make it from json")
		return
	}
	switch request.ActionType {
	case "add new project":
		CommitProject(w, r, request.Transaction)
		return
	case "add new Admin":
		commitNewGenericUserToChain(w, r, request.Transaction, "Admin")
		return
	case "add new Manager":
		commitNewGenericUserToChain(w, r, request.Transaction, "Manager")
		return
	case "add new User":
		commitNewGenericUserToChain(w, r, request.Transaction, "User")
		return
	case "add new hospital":
		CommitHospital(w, r, request.Transaction)
		return
	default:
		fmt.Println("Commit type", request.ActionType)
		medChainUtils.CheckError(errors.New("Unknown Action Type"), w, r)
		return
	}
}

func CancelAction(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if medChainUtils.CheckError(err, w, r) {
		return
	}
	var request messages.CommitRequest
	err = json.Unmarshal(body, &request)
	if medChainUtils.CheckError(err, w, r) {
		fmt.Println("Can't make it from json")
		return
	}
	switch request.ActionType {
	case "add new project":
		cancelNewProject(w, r, request.Transaction)
		return
	case "add new Admin":
		cancelNewGenericUser(w, r, request.Transaction, "Admin")
		return
	case "add new Manager":
		cancelNewGenericUser(w, r, request.Transaction, "Manager")
		return
	case "add new User":
		cancelNewGenericUser(w, r, request.Transaction, "User")
		return
	case "add new hospital":
		cancelNewHospital(w, r, request.Transaction)
		return
	default:
		fmt.Println("Commit type", request.ActionType)
		medChainUtils.CheckError(errors.New("Unknown Action Type"), w, r)
		return
	}
}
