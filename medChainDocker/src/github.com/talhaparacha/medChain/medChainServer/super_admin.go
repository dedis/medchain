package main

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"

	"github.com/talhaparacha/medChain/medChainServer/messages"
	"github.com/talhaparacha/medChain/medChainUtils"
)

func GetSuperAdminInfo(w http.ResponseWriter, r *http.Request) {
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

// func NewAdminMetadata(w http.ResponseWriter, r *http.Request) {
// 	body, err := ioutil.ReadAll(r.Body)
// 	medChainUtils.Check(err)
// 	var newDarcs medChainUtils.NewDarcsMetadata
// 	err = json.Unmarshal(body, &newDarcs)
// 	medChainUtils.Check(err)
// 	id := newDarcs.Id
// 	darcMap := newDarcs.Darcs
// 	if id == "" || darcMap == nil {
// 		http.Error(w, "", http.StatusNotFound)
// 		return
// 	}
// 	adminDarc, ok1 := newDarcs.Darcs["admin_darc"]
// 	if ok1 {
// 		addDarcToMaps(adminDarc, id, adminsDarcsMap)
// 	}
// 	fmt.Println("add admin ", id)
// }
