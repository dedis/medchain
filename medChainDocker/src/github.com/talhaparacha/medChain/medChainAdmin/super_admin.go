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
	manager_identity := medChainUtils.LoadIdentityEd25519FromBytes(Buf1.Bytes())
	fmt.Println("new admin:", manager_identity.String())
	createNewManagerDarc(manager_identity)
	http.Redirect(w, r, "/super_admin", http.StatusSeeOther)
}

func createNewAdminDarc(manager_identity darc.Identity) {
	// Get information from MedChain
	response, err := http.Get(medchainURL + "/info/super_admin?identity=" + signer.Identity().String())
	medChainUtils.Check(err)
	body, err := ioutil.ReadAll(response.Body)
	medChainUtils.Check(err)
	var reply medChainUtils.UserInfoReply
	err = json.Unmarshal(body, &reply)
	adminDarc := reply.MainDarc
	// managerListDarc := reply.ManagerListDarc
	owners := []darc.Identity{darc.NewIdentityDarc(adminDarc.GetID())}
	signers := []darc.Identity{manager_identity}
	rules := darc.InitRules(owners, signers)
	tempDarc := createDarcAndSubmit(adminDarc, rules, "Single Admin darc", signer)
	fmt.Println(tempDarc.GetIdentityString())
}
