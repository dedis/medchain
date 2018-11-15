package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

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

func newUserPart1(w http.ResponseWriter, r *http.Request) {
	var newUserInfo NewUserInfo
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		// TOD0 Return 400 error code
		panic(err)
	}
	medChainUtils.Check(err)
	err = json.Unmarshal(body, &newUserInfo)
	if err != nil {
		// TOD0 Return 400 error code
		panic(err)
	}
	fmt.Printf("Manager %s adds user %s\n", newUserInfo.ManagerPublicKey, newUserInfo.UserPublicKey)
	managerId := medChainUtils.LoadIdentityEd25519FromBytes([]byte(newUserInfo.ManagerPublicKey))
	managerDarc, ok := managersDarcsMap[managerId.String()]
	if !ok {
		// TOD0 Return 400 error code
	}
	userId := medChainUtils.LoadIdentityEd25519FromBytes([]byte(newUserInfo.UserPublicKey))
	owners := []darc.Identity{darc.NewIdentityDarc(managerDarc.GetID())}
	signers := []darc.Identity{userId}
	rules := darc.InitRules(owners, signers)
	ctx, tempDarc, err := createTransactionForNewDARC(allManagersDarc, rules, "User darc")
	response := NewUserTransaction{UserPublicKey: newUserInfo.UserPublicKey, ManagerPublicKey: newUserInfo.ManagerPublicKey, Transaction: *ctx, Darc: *tempDarc}
	jsonVal, err := json.Marshal(response)
	if err != nil {
		// TOD0 Return 400 error code
		panic(err)
	}
	w.Write(jsonVal)
}

func newUserPart2(w http.ResponseWriter, r *http.Request) {
	var signedTransaction NewUserTransaction
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		// TOD0 Return 400 error code
		panic(err)
	}
	medChainUtils.Check(err)
	err = json.Unmarshal(body, &signedTransaction)
	if err != nil {
		// TOD0 Return 400 error code
		panic(err)
	}
	_, err = submitSignedTransactionForNewDARC(cl, &signedTransaction.Darc, genesisMsg.BlockInterval, &signedTransaction.Transaction)
	if err != nil {
		// TOD0 Return 400 error code
		panic(err)
	}
	//TODO: Update the collective user darc

	//TODO: update the user-projects map
}

type ManagerInfoRequest struct {
	ManagerPublicKey []byte `json:"manager_public_key"`
}

type ManagerInfoReply struct {
	ManagerDarc  *darc.Darc `json:"manager_darc"`
	UserListDarc *darc.Darc `json:"user_list_darc"`
}

func GetManagerInfo(w http.ResponseWriter, r *http.Request) {
	var inforequest ManagerInfoRequest
	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		// TOD0 Return 400 error code
		panic(err)
	}
	medChainUtils.Check(err)
	err = json.Unmarshal(body, &inforequest)
	if err != nil {
		// TOD0 Return 400 error code
		panic(err)
	}
	identity := medChainUtils.LoadIdentityEd25519FromBytes(inforequest.ManagerPublicKey)
	managerDarc := managersDarcsMap[identity.String()]
	userListDarc := usersListDarcsMap[identity.String()]
	reply := ManagerInfoReply{ManagerDarc: managerDarc, UserListDarc: userListDarc}
	jsonVal, err := json.Marshal(reply)
	if err != nil {
		// TOD0 Return 400 error code
		panic(err)
	}
	w.Write(jsonVal)
}
