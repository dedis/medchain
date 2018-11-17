package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/dedis/cothority/omniledger/darc"
	"github.com/dedis/cothority/omniledger/service"
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
	identity := strings.Join(r.URL.Query()["identity"], "")
	darc_identity := strings.Join(r.URL.Query()["darc_identity"], "")
	var managerDarc *darc.Darc
	var ok bool
	if identity != "" {
		fmt.Println("manager info ", identity)
		managerDarc, ok = managersDarcsMap[identity]
	} else if darc_identity != "" {
		fmt.Println("manager darc query ", darc_identity)
		managerDarc, ok = managersDarcsMapWithDarcId[darc_identity]
		if ok {
			identity = string(managerDarc.Rules.GetSignExpr())
		}
	} else {
		ok = false
	}
	if !ok {
		// TODO return 404
		return
	}
	fmt.Println("manager info", identity)
	userListDarc := usersListDarcsMap[identity]
	reply := medChainUtils.ManagerInfoReply{ManagerDarc: managerDarc, UserListDarc: userListDarc}
	jsonVal, err := json.Marshal(reply)
	if err != nil {
		// TOD0 Return 400 error code
		panic(err)
	}
	w.Write(jsonVal)
}

func GetUserInfo(w http.ResponseWriter, r *http.Request) {
	identity := strings.Join(r.URL.Query()["identity"], "")
	darc_identity := strings.Join(r.URL.Query()["darc_identity"], "")
	var userDarc *darc.Darc
	var ok bool
	if identity != "" {
		fmt.Println("user query ", identity)
		userDarc, ok = usersDarcsMap[identity]
	} else if darc_identity != "" {
		fmt.Println("user darc query ", darc_identity)
		userDarc, ok = usersDarcsMapWithDarcId[darc_identity]
	} else {
		ok = false
	}
	if !ok {
		// TODO return 404
	}
	reply := medChainUtils.UserInfoReply{UserDarc: userDarc}
	jsonVal, err := json.Marshal(reply)
	if err != nil {
		// TOD0 Return 400 error code
		panic(err)
	}
	w.Write(jsonVal)
}
