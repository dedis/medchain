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
var signer_identity string
var public_key string
var private_key string

// HTTP Client with appropriate settings for later use
var client *http.Client

func landing(w http.ResponseWriter, r *http.Request) {
	clearInfo()
	fmt.Println("landing")
	tmpl := template.Must(template.ParseFiles("templates/static/landing.html"))
	tmpl.Execute(w, nil)
}

func logOut(w http.ResponseWriter, r *http.Request) {
	clearInfo()
}

func clearInfo() {
	medchainURL, usertype, public_key, private_key = "", "", "", ""
	signer = *new(darc.Signer)
	signer_identity = ""
}

func wrongLogin(w http.ResponseWriter, r *http.Request) {
	fmt.Println("wrongLogin")
	clearInfo()
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
	signer_identity = signer.Identity().String()
	var next_url string
	// fmt.Println(usertype)
	// fmt.Println(medchainURL)
	switch usertype {
	case "Manager":
		next_url = "/manager"
	case "Administrator":
		next_url = "/admin"
	case "SuperAdministrator":
		next_url = "/super_admin"
	default:
		next_url = "/"
	}
	http.Redirect(w, r, next_url, http.StatusSeeOther)
}

type UserLandingData struct {
	UserId              string
	SubordinatesIdsList [][]string
}

func getUserInfoAndDisplayIt(w http.ResponseWriter, r *http.Request, user_type, subordinate_type string) {
	fmt.Println(user_type + " landing")
	response, err := http.Get(medchainURL + "/info/" + user_type + "?identity=" + signer_identity)
	medChainUtils.Check(err)
	body, err := ioutil.ReadAll(response.Body)
	medChainUtils.Check(err)
	var reply medChainUtils.UserInfoReply
	err = json.Unmarshal(body, &reply)
	mainDarc := reply.MainDarc
	subordinatesDarcsList := reply.SubordinatesDarcsList
	if mainDarc == nil || subordinatesDarcsList == nil || len(subordinatesDarcsList) == 0 {
		wrongLogin(w, r)
		return
	}
	fmt.Println("mainDarc", mainDarc.GetIdentityString())
	SubordinatesIdsList := [][]string{}
	for _, subordinatesDarc := range subordinatesDarcsList {
		fmt.Println("subordinatesDarc", subordinatesDarc.GetIdentityString())
		SubordinatesIds := ExtractSignerIdsFromListDarc(subordinatesDarc)
		SubordinatesIdsList = append(SubordinatesIdsList, SubordinatesIds)
	}
	tmpl := template.Must(template.ParseFiles("templates/static/" + user_type + "_landing.html"))
	data := UserLandingData{UserId: signer.Identity().String(), SubordinatesIdsList: SubordinatesIdsList}
	tmpl.Execute(w, data)
}

func ExtractSignerIdsFromListDarc(listDarc *darc.Darc) []string {
	rules := listDarc.Rules
	expr := rules.GetSignExpr()
	fmt.Println("signing expr", string(expr))
	expr_string := string(expr)
	signer_darcs := splitAndOr(expr_string)
	SubordinatesIds := []string{}
	for _, signer_darc := range signer_darcs {
		switch {
		case strings.HasPrefix(signer_darc, "darc:"):
			response, err := http.Get(medchainURL + "/info/darc?darc_identity=" + signer_darc)
			medChainUtils.Check(err)
			body, err := ioutil.ReadAll(response.Body)
			medChainUtils.Check(err)
			var reply medChainUtils.UserInfoReply
			err = json.Unmarshal(body, &reply)
			medChainUtils.Check(err)
			subordinateDarc := reply.MainDarc
			if subordinateDarc != nil {
				extracted_ids := ExtractSignerIdsFromListDarc(subordinateDarc)
				SubordinatesIds = append(SubordinatesIds, extracted_ids...)
			}
		case strings.HasPrefix(signer_darc, "ed25519:"):
			SubordinatesIds = append(SubordinatesIds, signer_darc)
		default:

		}
	}
	return SubordinatesIds
}

func splitAndOr(expr string) []string {
	result := []string{}
	or_splitted := strings.Split(expr, " | ")
	for _, substring := range or_splitted {
		and_splitted := strings.Split(substring, " & ")
		result = append(result, and_splitted...)
	}
	return result
}

func postNewDarcsMetadata(new_darcs map[string]*darc.Darc, id, role string) {
	// Get the projects list from IRCT
	metaData := medChainUtils.NewDarcsMetadata{Darcs: new_darcs, Id: id}
	jsonVal, err := json.Marshal(metaData)
	if err != nil {
		panic(err)
	}
	request, err := http.NewRequest("POST", medchainURL+"/metadata/add/"+role, bytes.NewBuffer(jsonVal))
	request.Header.Set("Content-Type", "application/json")
	_, err = client.Do(request)
	medChainUtils.Check(err)
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

	port := getFlags()

	// Register addresses
	http.Handle("/templates/static/", http.StripPrefix("/templates/static/", http.FileServer(http.Dir("templates/static"))))
	http.HandleFunc("/", landing)
	http.HandleFunc("/init", initSigner)
	http.HandleFunc("/logout", logOut)
	http.HandleFunc("/manager", managerLanding)
	http.HandleFunc("/admin", adminLanding)
	http.HandleFunc("/super_admin", superAdminLanding)
	http.HandleFunc("/add/user", createUser)
	http.HandleFunc("/add/manager", createManager)
	http.HandleFunc("/add/admin", createAdmin)
	// Start server
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		panic(err)
	}
}
