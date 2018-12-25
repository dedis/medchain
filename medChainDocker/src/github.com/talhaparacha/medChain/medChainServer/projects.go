package main

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"

	"github.com/talhaparacha/medChain/medChainServer/messages"
	"github.com/talhaparacha/medChain/medChainServer/metadata"
	"github.com/talhaparacha/medChain/medChainUtils"
)

func ListProjects(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if medChainUtils.CheckError(err, w, r) {
		return
	}
	var request messages.ListProjectRequest
	err = json.Unmarshal(body, &request)
	if medChainUtils.CheckError(err, w, r) {
		return
	}
	var identity string
	if request.SuperAdminId != "" {
		identity = request.SuperAdminId
	} else {
		medChainUtils.CheckError(errors.New("No identity was given"), w, r)
		return
	}

	hospital_metadata, ok := metaData.Hospitals[identity]
	if !ok {
		medChainUtils.CheckError(errors.New("Hospital unknown"), w, r)
		return
	}

	var list []*metadata.GenericUser
	switch request.Role {
	case "admin":
		list = hospital_metadata.Admins
	case "manager":
		list = hospital_metadata.Managers
	case "user":
		list = hospital_metadata.Users
	default:
		medChainUtils.CheckError(errors.New("Unknown role"), w, r)
		return
	}

	userList := []messages.GenericUserInfoReply{}

	for _, user_metadata := range list {
		userReply := genericUserMetadataToInfoReply(user_metadata)
		userList = append(userList, userReply)
	}

	reply := messages.ListGenericUserReply{Users: userList}
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
