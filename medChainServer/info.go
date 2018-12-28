package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"

	"github.com/DPPH/MedChain/medChainServer/messages"
	"github.com/DPPH/MedChain/medChainServer/metadata"
	"github.com/DPPH/MedChain/medChainUtils"
)

// Communicate ID of the allUsersDarc if the system is running
func info(w http.ResponseWriter, r *http.Request) {

	if systemStart {
		allUsersDarc := metaData.AllUsersDarcBaseId
		allUsersDarcBaseId := metaData.AllUsersDarcBaseId
		allManagersDarcBaseId := metaData.AllManagersDarcBaseId
		allAdminsDarcBaseId := metaData.AllAdminsDarcBaseId
		allSuperAdminsDarcBaseId := metaData.AllSuperAdminsDarcBaseId
		userProjectsMapId := base64.StdEncoding.EncodeToString(metaData.UserProjectsMapInstanceID.Slice())
		genesisDarcBaseId := metaData.GenesisDarcBaseId
		reply := messages.GeneralInfoReply{SigningServiceUrl: metaData.SigningServiceUrl, GenesisDarcBaseId: genesisDarcBaseId, AllSuperAdminsDarcBaseId: allSuperAdminsDarcBaseId, AllAdminsDarcBaseId: allAdminsDarcBaseId, AllManagersDarcBaseId: allManagersDarcBaseId, AllUsersDarcBaseId: allUsersDarcBaseId, AllUsersDarc: allUsersDarc, UserProjectsMap: userProjectsMapId}
		json_val, err := json.Marshal(&reply)
		if medChainUtils.CheckError(err, w, r) {
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, err = w.Write(json_val)
		if medChainUtils.CheckError(err, w, r) {
			return
		}
	} else {
		temp := map[string]string{"error": "MedChain not started yet"}
		js, _ := json.Marshal(temp)
		w.Header().Set("Content-Type", "application/json")
		w.Write(js)
	}
}

func genericUserMetadataToInfoReply(user_metadata *metadata.GenericUser) messages.GenericUserInfoReply {
	return messages.GenericUserInfoReply{Id: user_metadata.Id.String(), Name: user_metadata.Name, DarcBaseId: user_metadata.DarcBaseId, SuperAdminId: user_metadata.Hospital.SuperAdmin.Id.String(), IsCreated: user_metadata.IsCreated, Role: user_metadata.Role}
}

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

func ListHospitals(w http.ResponseWriter, r *http.Request) {

	hospitalList := []messages.HospitalInfoReply{}

	for _, hospital_metadata := range metaData.Hospitals {
		hospitalReply := messages.HospitalInfoReply{SuperAdminId: hospital_metadata.SuperAdmin.Id.String(), HospitalName: hospital_metadata.Name, SuperAdminName: hospital_metadata.SuperAdmin.Name, IsCreated: hospital_metadata.SuperAdmin.IsCreated}
		hospitalList = append(hospitalList, hospitalReply)
	}

	reply := messages.ListHospitalReply{Hospitals: hospitalList}
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
