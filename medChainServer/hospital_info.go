package main

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"

	"github.com/DPPH/MedChain/medChainServer/messages"
	"github.com/DPPH/MedChain/medChainUtils"
)

/**
This file takes care of getting information on the hospitals
**/

/**
Get the information of a given hospital.
It should receive a messages.UserInfoRequest (encoded in json) in the body of the request
  with the id set to the one of the super admin of the hospital
It returns a messages.HospitalInfoReply in the body of the response
**/
func GetHospitalInfo(w http.ResponseWriter, r *http.Request) {
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
	hospital_metadata, ok := metaData.Hospitals[identity]
	if !ok {
		medChainUtils.CheckError(errors.New("Identity unknown"), w, r)
		return
	}
	reply := messages.HospitalInfoReply{SuperAdminId: hospital_metadata.SuperAdmin.Id.String(), HospitalName: hospital_metadata.Name, SuperAdminName: hospital_metadata.SuperAdmin.Name, AdminListDarcBaseId: hospital_metadata.AdminListDarcBaseId, ManagerListDarcBaseId: hospital_metadata.ManagerListDarcBaseId, UserListDarcBaseId: hospital_metadata.UserListDarcBaseId, IsCreated: hospital_metadata.SuperAdmin.IsCreated}
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
List all the hospitals in the system
It returns a messages.ListHospitalReply in the body of the response
**/
func ListHospitals(w http.ResponseWriter, r *http.Request) {

	hospitalList := []messages.HospitalInfoReply{}

	for _, hospital_metadata := range metaData.Hospitals {
		hospitalReply := messages.HospitalInfoReply{
			SuperAdminId:   hospital_metadata.SuperAdmin.Id.String(),
			HospitalName:   hospital_metadata.Name,
			SuperAdminName: hospital_metadata.SuperAdmin.Name,
			IsCreated:      hospital_metadata.SuperAdmin.IsCreated,
		}
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
