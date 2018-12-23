package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/talhaparacha/medChain/medChainServer/messages"
	"github.com/talhaparacha/medChain/medChainUtils"
)

// Communicate ID of the allUsersDarc if the system is running
func info(w http.ResponseWriter, r *http.Request) {

	if systemStart {
		allUsersDarc := metaData.AllUsersDarcBaseId
		allUsersDarcBaseId := metaData.AllUsersDarcBaseId
		allManagersDarcBaseId := metaData.AllManagersDarcBaseId
		allAdminsDarcBaseId := metaData.AllAdminsDarcBaseId
		allSuperAdminsDarcBaseId := metaData.AllSuperAdminsDarcBaseId
		userProjectsMapId := base64.StdEncoding.EncodeToString(userProjectsMapInstanceID.Slice())
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

func GetUserType(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if medChainUtils.CheckError(err, w, r) {
		return
	}
	fmt.Println("info/type", string(body))
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

	_, ok1 := metaData.Hospitals[identity]
	_, ok2 := metaData.Admins[identity]
	_, ok3 := metaData.Managers[identity]
	_, ok4 := metaData.Users[identity]

	var user_type string
	if ok1 {
		user_type = "hospital"
	} else if ok2 {
		user_type = "admin"
	} else if ok3 {
		user_type = "manager"
	} else if ok4 {
		user_type = "user"
	} else {
		medChainUtils.CheckError(errors.New("Unknown user"), w, r)
		return
	}

	reply := messages.UserTypeReply{Type: user_type}
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
