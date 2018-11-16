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

type ManagerInfoRequest struct {
	ManagerPublicKey []byte `json:"manager_public_key"`
}

func GetManagerInfo(w http.ResponseWriter, r *http.Request) {
	identity := strings.Join(r.URL.Query()["identity"], "")
	fmt.Println("manager info", identity)
	managerDarc := managersDarcsMap[identity]
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
	fmt.Println("user query ", identity)
	userDarc, ok := usersDarcsMap[identity]
	if !ok {
		userDarc, ok = usersDarcsMapWithDarcId[identity]
	}
	reply := medChainUtils.UserInfoReply{UserDarc: userDarc}
	jsonVal, err := json.Marshal(reply)
	if err != nil {
		// TOD0 Return 400 error code
		panic(err)
	}
	w.Write(jsonVal)
}
