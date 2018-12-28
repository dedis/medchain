package main

import (
	"fmt"
	"net/http"

	"github.com/dedis/cothority/omniledger/darc"
	"github.com/dedis/cothority/omniledger/service"
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

func AddManager(w http.ResponseWriter, r *http.Request) {
	fmt.Println("/add/manager")
	replyNewGenericUserRequest(w, r, "Manager")
}

func CommitManager(w http.ResponseWriter, r *http.Request) {
	fmt.Println("/commit/manager")
	commitNewGenericUserToChain(w, r, "Manager")
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
