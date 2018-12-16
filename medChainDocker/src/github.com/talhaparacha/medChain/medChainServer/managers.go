package main

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"

	"github.com/dedis/cothority/omniledger/darc"
	"github.com/dedis/cothority/omniledger/service"
	"github.com/talhaparacha/medChain/medChainServer/messages"
	"github.com/talhaparacha/medChain/medChainUtils"
)

type NewUserInfo struct {
	UserPublicKey    string `json:"user_public_key"`
	ManagerPublicKey string `json:"manager_public_key"`
}

type NewUserTransaction struct {
	UserPublicKey    string                    `json:"user_public_key"`
	ManagerPublicKey string                    `json:"manager_public_key"`
	Transaction      service.ClientTransaction `json:"transaction"`
	Darc             darc.Darc                 `json:"darc"`
}

func GetManagerInfo(w http.ResponseWriter, r *http.Request) {
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
	hopital_metadata, ok := metaData.Managers[identity]
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

func GetUserInfo(w http.ResponseWriter, r *http.Request) {
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
	hopital_metadata, ok := metaData.Users[identity]
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

// func NewUserMetadata(w http.ResponseWriter, r *http.Request) {
// 	body, err := ioutil.ReadAll(r.Body)
// 	medChainUtils.Check(err)
// 	var newDarcs medChainUtils.NewDarcsMetadata
// 	err = json.Unmarshal(body, &newDarcs)
// 	medChainUtils.Check(err)
// 	id := newDarcs.Id
// 	userDarc, ok := newDarcs.Darcs["user_darc"]
// 	if !ok || id == "" || userDarc == nil {
// 		http.Error(w, "", http.StatusNotFound)
// 		return
// 	}
// 	fmt.Println("add user ", id)
// 	addDarcToMaps(userDarc, id, usersDarcsMap)
// }
