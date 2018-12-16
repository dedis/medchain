package main

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"

	"github.com/talhaparacha/medChain/medChainServer/messages"
	"github.com/talhaparacha/medChain/medChainUtils"
)

func GetAdminInfo(w http.ResponseWriter, r *http.Request) {
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
	hopital_metadata, ok := metaData.Admins[identity]
	if !ok {
		medChainUtils.CheckError(errors.New("Identity unknown"), w, r)
		return
	}
	reply := messages.UserInfoReply{DarcBaseId: hopital_metadata.DarcBaseId}
	json_val, err := json.Marshal(&reply)
	if medChainUtils.CheckError(err, w, r) {
		return
	}
	_, err = w.Write(json_val)
	if medChainUtils.CheckError(err, w, r) {
		return
	}
}

// func NewManagerMetadata(w http.ResponseWriter, r *http.Request) {
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
// 	managerDarc, ok1 := newDarcs.Darcs["manager_darc"]
// 	if ok1 {
// 		addDarcToMaps(managerDarc, id, managersDarcsMap)
// 	}
// 	fmt.Println("add manager ", id)
// }
