package main

import (
	"bytes"
	"crypto/tls"
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

var medchainURL string
var usertype string
var signer darc.Signer
var public_key string
var private_key string

// HTTP Client with appropriate settings for later use
var client *http.Client

func landing(w http.ResponseWriter, r *http.Request) {
	fmt.Println("landing")
	tmpl := template.Must(template.ParseFiles("templates/static/landing.html"))
	tmpl.Execute(w, nil)
}

func wrongLogin(w http.ResponseWriter, r *http.Request) {
	fmt.Println("wrongLogin")
	tmpl := template.Must(template.ParseFiles("templates/static/wrong_login.html"))
	tmpl.Execute(w, nil)
}

func initSigner(w http.ResponseWriter, r *http.Request) {
	r.ParseMultipartForm(32 << 20) //Parse url parameters passed, then parse the response packet for the POST body (request body)
	// attention: If you do not call ParseForm method, the following data can not be obtained form
	usertype = r.FormValue("usertype")
	medchainURL = r.FormValue("medchainurl")
	var Buf1 bytes.Buffer
	file, _, err := r.FormFile("publickey")
	medChainUtils.Check(err)
	io.Copy(&Buf1, file)
	var Buf2 bytes.Buffer
	file, _, err = r.FormFile("privatekey")
	medChainUtils.Check(err)
	io.Copy(&Buf2, file)
	signer = medChainUtils.LoadSignerEd25519FromBytes(Buf1.Bytes(), Buf2.Bytes())
	var next_url string
	// fmt.Println(usertype)
	// fmt.Println(medchainURL)
	switch usertype {
	case "Manager":
		next_url = "/manager"
	case "Administrator":
		next_url = "/admin"
	default:
		next_url = "/"
	}
	http.Redirect(w, r, next_url, http.StatusSeeOther)
}

type UserLandingData struct {
	UserId          string
	SubordinatesIds []string
}

func getUserInfoAndDisplayIt(w http.ResponseWriter, r *http.Request, user_type, subordinate_type string) {
	fmt.Println(user_type + " landing")
	response, err := http.Get(medchainURL + "/info/" + user_type + "?identity=" + signer.Identity().String())
	medChainUtils.Check(err)
	body, err := ioutil.ReadAll(response.Body)
	medChainUtils.Check(err)
	var reply medChainUtils.UserInfoReply
	err = json.Unmarshal(body, &reply)
	mainDarc := reply.MainDarc
	subordinatesDarc := reply.SubordinatesDarc
	if mainDarc == nil || subordinatesDarc == nil {
		wrongLogin(w, r)
		return
	}
	fmt.Println("mainDarc", mainDarc.GetIdentityString())
	fmt.Println("subordinatesDarc", subordinatesDarc.GetIdentityString())
	rules := subordinatesDarc.Rules
	expr := rules.GetSignExpr()
	fmt.Println("signing expr", string(expr))
	expr_string := string(expr)
	signer_darcs := strings.Split(expr_string, " | ")
	SubordinatesIds := []string{}
	for _, signer_darc := range signer_darcs {
		response, err := http.Get(medchainURL + "/info/" + subordinate_type + "?darc_identity=" + signer_darc)
		medChainUtils.Check(err)
		body, err := ioutil.ReadAll(response.Body)
		medChainUtils.Check(err)
		var reply medChainUtils.UserInfoReply
		err = json.Unmarshal(body, &reply)
		medChainUtils.Check(err)
		subordinateDarc := reply.MainDarc
		if subordinateDarc != nil {
			fmt.Println(subordinate_type, subordinateDarc.GetIdentityString())
			signing_expr := string(subordinateDarc.Rules.GetSignExpr())
			fmt.Println(subordinate_type+" id :", signing_expr)
			SubordinatesIds = append(SubordinatesIds, signing_expr)
		}
	}
	tmpl := template.Must(template.ParseFiles("templates/static/" + user_type + "_landing.html"))
	data := UserLandingData{UserId: signer.Identity().String(), SubordinatesIds: SubordinatesIds}
	tmpl.Execute(w, data)
}

func main() {
	// // Setup HTTP client
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client = &http.Client{Transport: tr}
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		req.Header.Add("Authorization", via[0].Header.Get("Authorization"))
		return nil
	}

	// Register addresses
	http.Handle("/templates/static/", http.StripPrefix("/templates/static/", http.FileServer(http.Dir("templates/static"))))
	http.HandleFunc("/", landing)
	http.HandleFunc("/init", initSigner)
	http.HandleFunc("/manager", managerLanding)
	http.HandleFunc("/admin", adminLanding)
	http.HandleFunc("/super_admin", superAdminLanding)
	http.HandleFunc("/add/user", createUser)
	http.HandleFunc("/add/manager", createManager)
	http.HandleFunc("/add/admin", createAdmin)
	// Start server
	if err := http.ListenAndServe(":6161", nil); err != nil {
		panic(err)
	}
}
