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

type AdminLandingData struct {
	AdminId    string
	ManagerIds []string
}

func adminLanding(w http.ResponseWriter, r *http.Request) {
	fmt.Println("adminlanding")
	response, err := http.Get(medchainURL + "/info/admin?identity=" + signer.Identity().String())
	medChainUtils.Check(err)
	body, err := ioutil.ReadAll(response.Body)
	medChainUtils.Check(err)
	var reply medChainUtils.AdminInfoReply
	err = json.Unmarshal(body, &reply)
	adminDarc := reply.AdminDarc
	managerListDarc := reply.ManagerListDarc
	fmt.Println("admindarc", adminDarc.GetIdentityString())
	fmt.Println("managerListDarc", managerListDarc.GetIdentityString())
	rules := managerListDarc.Rules
	expr := rules.GetSignExpr()
	fmt.Println("signing expr", string(expr))
	expr_string := string(expr)
	signer_darcs := strings.Split(expr_string, " | ")
	ManagerIds := []string{}
	for _, signer_darc := range signer_darcs {
		response, err := http.Get(medchainURL + "/info/manager?darc_identity=" + signer_darc)
		medChainUtils.Check(err)
		body, err := ioutil.ReadAll(response.Body)
		medChainUtils.Check(err)
		var reply medChainUtils.ManagerInfoReply
		err = json.Unmarshal(body, &reply)
		medChainUtils.Check(err)
		managerDarc := reply.ManagerDarc
		fmt.Println("manager", managerDarc.GetIdentityString())
		signing_expr := string(managerDarc.Rules.GetSignExpr())
		fmt.Println("manager id :", signing_expr)
		ManagerIds = append(ManagerIds, signing_expr)
	}
	tmpl := template.Must(template.ParseFiles("templates/static/admin_landing.html"))
	data := AdminLandingData{AdminId: signer.Identity().String(), ManagerIds: ManagerIds}
	tmpl.Execute(w, data)
}

func createManager(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(32 << 20)
	var Buf1 bytes.Buffer
	file, _, err := r.FormFile("publickey")
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
	var reply medChainUtils.AdminInfoReply
	err = json.Unmarshal(body, &reply)
	adminDarc := reply.AdminDarc
	// managerListDarc := reply.ManagerListDarc
	owners := []darc.Identity{darc.NewIdentityDarc(adminDarc.GetID())}
	signers := []darc.Identity{manager_identity}
	rules := darc.InitRules(owners, signers)
	tempDarc := createDarcAndSubmit(adminDarc, rules, "Single Manager darc", signer)
	fmt.Println(tempDarc.GetIdentityString())
}
