package main

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"

	"github.com/DPPH/MedChain/medChainServer/messages"
	"github.com/DPPH/MedChain/medChainServer/metadata"
	"github.com/DPPH/MedChain/medChainUtils"
)

/**
This file takes care of getting the information on generic users (admins, managers, users)
**/

/**
Translates the metadata of the user to a messages.GenericUserInfoReply object
**/
func genericUserMetadataToInfoReply(user_metadata *metadata.GenericUser) messages.GenericUserInfoReply {
	return messages.GenericUserInfoReply{
		Id:           user_metadata.Id.String(),
		Name:         user_metadata.Name,
		DarcBaseId:   user_metadata.DarcBaseId,
		SuperAdminId: user_metadata.Hospital.SuperAdmin.Id.String(),
		HospitalName: user_metadata.Hospital.Name,
		IsCreated:    user_metadata.IsCreated,
		Role:         user_metadata.Role,
	}
}

func genericUserMetadataToInfoReplyShort(user_metadata *metadata.GenericUser) messages.GenericUserInfoReply {
	return messages.GenericUserInfoReply{
		Id:        user_metadata.Id.String(),
		Name:      user_metadata.Name,
		IsCreated: user_metadata.IsCreated,
	}
}

/**
Get the information of a given user.
It should receive a messages.UserInfoRequest (encoded in json) in the body of the request
It returns a messages.GenericUserInfoReply in the body of the response
**/
func GetGenericUserInfo(w http.ResponseWriter, r *http.Request) {
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
		id, err := medChainUtils.LoadIdentityEd25519FromBytesWithErr([]byte(request.PublicKey))
		if medChainUtils.CheckError(err, w, r) {
			return
		}
		identity = id.String()
	} else {
		medChainUtils.CheckError(errors.New("No identity Nor public key was given"), w, r)
		return
	}
	user_metadata, ok := metaData.GenericUsers[identity]
	if !ok {
		medChainUtils.CheckError(errors.New("Identity unknown"), w, r)
		return
	}
	reply := genericUserMetadataToInfoReply(user_metadata)
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

/**
List all the users of a certain type in a given hospital
It should receive a messages.ListGenericUserRequest (encoded in json) in the body of the request
It returns a messages.ListGenericUserReply in the body of the response
**/
func ListGenericUser(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if medChainUtils.CheckError(err, w, r) {
		return
	}
	var request messages.ListGenericUserRequest
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
		userReply := genericUserMetadataToInfoReplyShort(user_metadata)
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
