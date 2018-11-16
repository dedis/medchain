package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/dedis/cothority/omniledger/darc"
	"github.com/talhaparacha/medChain/medChainUtils"
)

type ManagerLandingData struct {
	ManagerId string
	UserIds   []string
}

func managerLanding(w http.ResponseWriter, r *http.Request) {
	fmt.Println("managerlanding")
	// Get information, necessary for a log-in transaction, from MedChain
	response, err := http.Get(medchainURL + "/info/manager?identity=" + signer.Identity().String())
	medChainUtils.Check(err)
	body, err := ioutil.ReadAll(response.Body)
	medChainUtils.Check(err)
	var reply medChainUtils.ManagerInfoReply
	err = json.Unmarshal(body, &reply)
	managerDarc := reply.ManagerDarc
	userListDarc := reply.UserListDarc
	fmt.Println("managerdarc", managerDarc.GetIdentityString())
	fmt.Println("userListDarc", userListDarc.GetIdentityString())
	rules := userListDarc.Rules
	expr := rules.GetSignExpr()
	fmt.Println("signing expr", string(expr))
	expr_string := string(expr)
	signer_darcs := strings.Split(expr_string, " | ")
	UserIds := []string{}
	for _, signer_darc := range signer_darcs {
		response, err := http.Get(medchainURL + "/info/user?identity=" + signer_darc)
		medChainUtils.Check(err)
		body, err := ioutil.ReadAll(response.Body)
		medChainUtils.Check(err)
		var reply medChainUtils.UserInfoReply
		err = json.Unmarshal(body, &reply)
		medChainUtils.Check(err)
		userDarc := reply.UserDarc
		fmt.Println("user", userDarc.GetIdentityString())
		signing_expr := string(userDarc.Rules.GetSignExpr())
		fmt.Println("user id :", signing_expr)
		UserIds = append(UserIds, signing_expr)
	}
	tmpl := template.Must(template.ParseFiles("templates/static/manager_landing.html"))
	data := ManagerLandingData{ManagerId: signer.Identity().String(), UserIds: UserIds}
	tmpl.Execute(w, data)
}

func createUser(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(32 << 20)
	var Buf1 bytes.Buffer
	file, _, err := r.FormFile("publickey")
	medChainUtils.Check(err)
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
	var reply medChainUtils.ManagerInfoReply
	err = json.Unmarshal(body, &reply)
	managerDarc := reply.ManagerDarc
	// userListDarc := reply.UserListDarc
	owners := []darc.Identity{darc.NewIdentityDarc(managerDarc.GetID())}
	signers := []darc.Identity{user_identity}
	rules := darc.InitRules(owners, signers)
	tempDarc := createDarcAndSubmit(managerDarc, rules, "Single User darc", signer)
	fmt.Println(tempDarc.GetIdentityString())
}

func createDarcAndSubmit(baseDarc *darc.Darc, rules darc.Rules, description string, signers ...darc.Signer) *darc.Darc {
	// Create a transaction to spawn a DARC
	tempDarc := darc.NewDarc(rules, []byte(description))
	transaction := medChainUtils.CreateNewDarcTransaction(baseDarc, tempDarc, signers)
	request, err := http.NewRequest("GET", medchainURL+"/applyTransaction", nil)
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
