package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/DPPH/cothority/ocs/darc"
	"github.com/talhaparacha/medChain/medChainServer/messages"
	"github.com/talhaparacha/medChain/medChainServer/metadata"
	"github.com/talhaparacha/medChain/medChainUtils"
)

func GetUserInfo(w http.ResponseWriter, r *http.Request) {
	fmt.Println("/info/user")
	getGenericUserInfo(w, r, metaData.Users)
}

func getGenericUserInfo(w http.ResponseWriter, r *http.Request, user_metadata_map map[string]*metadata.GenericUser) {
	body, err := ioutil.ReadAll(r.Body)
	if medChainUtils.CheckError(err, w, r) {
		return
	}
	var request messages.UserInfoRequest
	err = json.Unmarshal(body, &request)
	if medChainUtils.CheckError(err, w, r) {
		return
	}
	var identity string
	if request.Identity != "" {
		identity = request.Identity
	} else if request.PublicKey != "" {
		id := medChainUtils.LoadIdentityEd25519FromBytes([]byte(request.PublicKey))
		identity = id.String()
	} else {
		medChainUtils.CheckError(errors.New("No identity Nor public key was given"), w, r)
		return
	}
	user_metadata, ok := user_metadata_map[identity]
	if !ok {
		medChainUtils.CheckError(errors.New("Identity unknown"), w, r)
		return
	}
	reply := messages.GenericUserInfoReply{Id: user_metadata.Id.String(), Name: user_metadata.Name, DarcBaseId: user_metadata.DarcBaseId, SuperAdminId: user_metadata.Hospital.Id.String()}
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

func AddUser(w http.ResponseWriter, r *http.Request) {
	extractNewUserRequest(w, r, "User")
}

func extractNewUserRequest(w http.ResponseWriter, r *http.Request, user_type string) {
	body, err := ioutil.ReadAll(r.Body)
	if medChainUtils.CheckError(err, w, r) {
		return
	}
	var request messages.AddGenericUserRequest
	err = json.Unmarshal(body, &request)
	if medChainUtils.CheckError(err, w, r) {
		return
	}
	transaction, signers, threshold, err := prepareNewUser(request, user_type)
	if medChainUtils.CheckError(err, w, r) {
		return
	}
	reply := messages.AddgenericUserReply{Transaction: transaction, Signers: signers, Threshold:threshold}
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

func prepareNewUser(request *messages.AddUserRequest, user_type string) (string, []string, int, error) {
	var identity string
	if request.Identity != "" {
		identity = request.Identity
	} else if request.PublicKey != "" {
		id := medChainUtils.LoadIdentityEd25519FromBytes([]byte(request.PublicKey))
		identity = id.String()
	} else {
		return "", nil, errors.New("No identity Nor public key was given for the new user")
	}
	super_admin_id := request.SuperAdminIdentity
	if super_admin_id == "" {
		return "", nil, errors.New("No identity was given for the super admin")
	}
	hospital_metadata, ok := metaData.Hospitals[super_admin_id]
	if !ok {
		return "", nil, errors.New("No super admin with this id")
	}
	var owner_darc *darc.Darc
	switch user_type {
	case "Admin":
		owner_darc, ok = metaData.BaseIdToDarcMap[hospital_metadata.DarcBaseId]
	default:
		owner_darc, ok = metaData.BaseIdToDarcMap[hospital_metadata.DarcBaseId]
	}
	if !ok {
		return "", nil, errors.New("Could not find the owner darc")
	}
	hash_map := make(map[string]bool)
	err := recursivelyFindUsersOfDarc(owner_darc, &hash_map)
	if  err != nil {
		return "", nil, err
	}
	signers := []string{}
	for user_id, _ := range hash_map {
		signers = append(signers, user_id)
	}
	signers, err :=
	user_metadata, ok := user_metadata_map[identity]
	if !ok {
		medChainUtils.CheckError(errors.New("Identity unknown"), w, r)
		return
	}
}
