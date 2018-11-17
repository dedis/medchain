package main

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"html/template"
	"io"
	"net/http"

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
	fmt.Println(usertype)
	fmt.Println(medchainURL)
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
	http.HandleFunc("/add/user", createUser)
	http.HandleFunc("/add/manager", createManager)
	// Start server
	if err := http.ListenAndServe(":6161", nil); err != nil {
		panic(err)
	}
}
