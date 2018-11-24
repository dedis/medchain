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

type SuperAdminLandingData struct {
	AdminId    string
	ManagerIds []string
}

func superAdminLanding(w http.ResponseWriter, r *http.Request) {
	getUserInfoAndDisplayIt(w, r, "super_admin", "admin")
}

func createAdmin(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(32 << 20)
	var Buf1 bytes.Buffer
	file, _, err := r.FormFile("publickey")
	if err != nil {
		http.Redirect(w, r, "/super_admin", http.StatusSeeOther)
		return
	}
	medChainUtils.Check(err)
	io.Copy(&Buf1, file)
	admin_identity := medChainUtils.LoadIdentityEd25519FromBytes(Buf1.Bytes())
	fmt.Println("new admin:", admin_identity.String())
	createNewManagerDarc(admin_identity)
	http.Redirect(w, r, "/super_admin", http.StatusSeeOther)
}

func createNewAdminDarc(admin_identity darc.Identity) {
	// Get information from MedChain
	response, err := http.Get(medchainURL + "/info/super_admin?identity=" + signer.Identity().String())
	medChainUtils.Check(err)
	body, err := ioutil.ReadAll(response.Body)
	medChainUtils.Check(err)
	var reply medChainUtils.UserInfoReply
	err = json.Unmarshal(body, &reply)
	superAdminDarc := reply.MainDarc
	adminsListDarc := reply.SubordinatesDarcsList[0]
	managersListLevel1Darc := reply.SubordinatesDarcsList[1]
	usersListLevel2Darc := reply.SubordinatesDarcsList[2]

	owners := []darc.Identity{darc.NewIdentityDarc(superAdminDarc.GetID())}
	signers := []darc.Identity{admin_identity}
	rules := darc.InitRulesWith(owners, signers, "invoke:evolve")
	rules.AddRule("spawn:darc", rules.GetSignExpr())
	adminDarc := createDarcAndSubmit(superAdminDarc, rules, "Single Admin darc", signer)
	fmt.Println(adminDarc.GetIdentityString())

	updatedAdminsListDarc := addSignerToDarcAndEvolve(superAdminDarc, adminsListDarc, adminDarc.GetIdentityString(), signer)
	fmt.Println(updatedAdminsListDarc.GetIdentityString())

	owners = []darc.Identity{darc.NewIdentityDarc(adminDarc.GetID())}
	signers = []darc.Identity{}
	rules.AddRule("spawn:darc", rules.GetSignExpr())
	rules = darc.InitRulesWith(owners, signers, "invoke:evolve")
	managersListLevel0Darc := createDarcAndSubmit(superAdminDarc, rules, "Managers List Level 0, Admin :"+admin_identity.String(), signer)
	fmt.Println(managersListLevel0Darc.GetIdentityString())

	updatedManagersListLevel1Darc := addSignerToDarcAndEvolve(superAdminDarc, managersListLevel1Darc, managersListLevel0Darc.GetIdentityString(), signer)
	fmt.Println(updatedManagersListLevel1Darc.GetIdentityString())

	owners = []darc.Identity{darc.NewIdentityDarc(adminDarc.GetID())}
	signers = []darc.Identity{}
	rules = darc.InitRulesWith(owners, signers, "invoke:evolve")
	usersListLevel1Darc := createDarcAndSubmit(superAdminDarc, rules, "Users List Level1 , Admin :"+admin_identity.String(), signer)
	fmt.Println(usersListLevel1Darc.GetIdentityString())

	updatedUsersListLevel2Darc := addSignerToDarcAndEvolve(superAdminDarc, usersListLevel2Darc, usersListLevel1Darc.GetIdentityString(), signer)
	fmt.Println(updatedUsersListLevel2Darc.GetIdentityString())

	darcsMap := map[string]*darc.Darc{"admin_darc": adminDarc, "user_list_darc": usersListLevel1Darc, "manager_list_darc": managersListLevel0Darc}
	postNewDarcsMetadata(darcsMap, admin_identity.String(), "admin")
}
