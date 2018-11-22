package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"

	"github.com/dedis/cothority/omniledger/darc"
	"github.com/dedis/cothority/omniledger/darc/expression"
	"github.com/talhaparacha/medChain/medChainUtils"
)

func managerLanding(w http.ResponseWriter, r *http.Request) {
	getUserInfoAndDisplayIt(w, r, "manager", "user")
}

func createUser(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(32 << 20)
	var Buf1 bytes.Buffer
	file, _, err := r.FormFile("publickey")
	if err != nil {
		http.Redirect(w, r, "/manager", http.StatusSeeOther)
		return
	}
	io.Copy(&Buf1, file)
	user_identity := medChainUtils.LoadIdentityEd25519FromBytes(Buf1.Bytes())
	fmt.Println("new user:", user_identity.String())
	createNewUserDarc(user_identity)
	http.Redirect(w, r, "/manager", http.StatusSeeOther)
}

func createNewUserDarc(user_identity darc.Identity) {
	// Get information from MedChain
	response, err := http.Get(medchainURL + "/info/manager?identity=" + signer.Identity().String())
	medChainUtils.Check(err)
	body, err := ioutil.ReadAll(response.Body)
	medChainUtils.Check(err)
	var reply medChainUtils.UserInfoReply
	err = json.Unmarshal(body, &reply)
	managerDarc := reply.MainDarc
	userListDarc := reply.SubordinatesDarc
	owners := []darc.Identity{darc.NewIdentityDarc(managerDarc.GetID())}
	signers := []darc.Identity{user_identity}
	rules := darc.InitRules(owners, signers)
	tempDarc := createDarcAndSubmit(managerDarc, rules, "Single User darc", signer)
	fmt.Println(tempDarc.GetIdentityString())
	newDarc := addSignerToDarcAndEvolve(managerDarc, userListDarc, tempDarc.GetIdentityString(), signer)
	fmt.Println(newDarc.GetIdentityString())
}

func updateDarcAndSubmit(baseDarc, oldDarc, newDarc *darc.Darc, signers ...darc.Signer) {
	// Create a transaction to evolve a DARC
	transaction := medChainUtils.CreateEvolveDarcTransaction(baseDarc, oldDarc, newDarc, signers)
	fmt.Println("sending evolve transaction", transaction)
	request, err := http.NewRequest("GET", medchainURL+"/evolve/darc", nil)
	medChainUtils.Check(err)
	request.Header.Set("transaction", transaction)
	response, err := client.Do(request)
	medChainUtils.Check(err)
	body, err := ioutil.ReadAll(response.Body)
	medChainUtils.Check(err)
	result := string(body[:])
	fmt.Println("Result", result)
}

func addSignerToDarcAndEvolve(baseDarc, oldDarc *darc.Darc, identityString string, signers ...darc.Signer) *darc.Darc {
	newDarc := oldDarc.Copy()
	newDarc.EvolveFrom(oldDarc)
	sign_expr := oldDarc.Rules.GetSignExpr()
	new_sign_expr := expression.InitOrExpr(string(sign_expr), identityString)
	newDarc.Rules.UpdateSign(new_sign_expr)
	updateDarcAndSubmit(baseDarc, oldDarc, newDarc, signers...)
	return newDarc
}

func createDarcAndSubmit(baseDarc *darc.Darc, rules darc.Rules, description string, signers ...darc.Signer) *darc.Darc {
	// Create a transaction to spawn a DARC
	tempDarc := darc.NewDarc(rules, []byte(description))
	transaction := medChainUtils.CreateNewDarcTransaction(baseDarc, tempDarc, signers)
	fmt.Println("sending spawn transaction", transaction)
	request, err := http.NewRequest("GET", medchainURL+"/add/darc", nil)
	medChainUtils.Check(err)
	request.Header.Set("transaction", transaction)
	response, err := client.Do(request)
	medChainUtils.Check(err)
	body, err := ioutil.ReadAll(response.Body)
	medChainUtils.Check(err)
	result := string(body[:])
	fmt.Println("Result", result)
	return tempDarc
}
