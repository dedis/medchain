package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/dedis/cothority/omniledger/darc"
	"github.com/talhaparacha/medChain/medChainUtils"
)

type AdminLandingData struct {
	AdminId    string
	ManagerIds []string
}

func adminLanding(w http.ResponseWriter, r *http.Request) {
	getUserInfoAndDisplayIt(w, r, "admin", "manager")
}

func createManager(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(32 << 20)
	var Buf1 bytes.Buffer
	file, _, err := r.FormFile("publickey")
	if err != nil {
		http.Redirect(w, r, "/manager", http.StatusSeeOther)
		return
	}
	medChainUtils.Check(err)
	io.Copy(&Buf1, file)
	manager_identity := medChainUtils.LoadIdentityEd25519FromBytes(Buf1.Bytes())
	fmt.Println("new manager:", manager_identity.String())
	createNewManagerDarc(manager_identity)
	http.Redirect(w, r, "/admin", http.StatusSeeOther)
}

func createNewManagerDarc(manager_identity darc.Identity) {
	// Get information from MedChain
	response, err := http.Get(medchainURL + "/info/admin?identity=" + signer.Identity().String())
	medChainUtils.Check(err)
	body, err := ioutil.ReadAll(response.Body)
	medChainUtils.Check(err)
	var reply medChainUtils.UserInfoReply
	err = json.Unmarshal(body, &reply)
	adminDarc := reply.MainDarc
	managersListLevel0Darc := reply.SubordinatesDarcsList[0]
	usersListLevel1Darc := reply.SubordinatesDarcsList[1]

	owners := []darc.Identity{darc.NewIdentityDarc(adminDarc.GetID())}
	signers := []darc.Identity{manager_identity}
	rules := darc.InitRulesWith(owners, signers, "invoke:evolve")
	rules.AddRule("spawn:darc", rules.GetSignExpr())
	managerDarc := createDarcAndSubmit(adminDarc, rules, "Single Manager darc", signer)
	fmt.Println(managerDarc.GetIdentityString())

	updatedManagersListDarc := addSignerToDarcAndEvolve(adminDarc, managersListLevel0Darc, managerDarc.GetIdentityString(), signer)
	fmt.Println(updatedManagersListDarc.GetIdentityString())

	owners = []darc.Identity{darc.NewIdentityDarc(managerDarc.GetID())}
	signers = []darc.Identity{}
	rules = darc.InitRulesWith(owners, signers, "invoke:evolve")
	usersListLevel0Darc := createDarcAndSubmit(adminDarc, rules, "Users List Level 0, Manager :"+manager_identity.String(), signer)
	fmt.Println(usersListLevel0Darc.GetIdentityString())

	updatedUsersListLevel1Darc := addSignerToDarcAndEvolve(adminDarc, usersListLevel1Darc, usersListLevel0Darc.GetIdentityString(), signer)
	fmt.Println(updatedUsersListLevel1Darc.GetIdentityString())

	darcsMap := map[string]*darc.Darc{"manager_darc": managerDarc, "user_list_darc": usersListLevel0Darc}
	postNewDarcsMetadata(darcsMap, manager_identity.String(), "manager")
}

// func createNewManagerDarc(manager_identity darc.Identity) {
// 	// Get information from MedChain
// 	response, err := http.Get(medchainURL + "/info/admin?identity=" + signer.Identity().String())
// 	medChainUtils.Check(err)
// 	body, err := ioutil.ReadAll(response.Body)
// 	medChainUtils.Check(err)
// 	var reply medChainUtils.UserInfoReply
// 	err = json.Unmarshal(body, &reply)
// 	adminDarc := reply.MainDarc
// 	// managerListDarc := reply.ManagerListDarc
// 	owners := []darc.Identity{darc.NewIdentityDarc(adminDarc.GetID())}
// 	signers := []darc.Identity{manager_identity}
// 	rules := darc.InitRules(owners, signers)
// 	tempDarc := createDarcAndSubmit(adminDarc, rules, "Single Manager darc", signer)
// 	fmt.Println(tempDarc.GetIdentityString())
// }
