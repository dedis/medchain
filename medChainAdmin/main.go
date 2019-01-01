package main

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/DPPH/MedChain/medChainAdmin/admin_messages"
	"github.com/DPPH/MedChain/medChainUtils"
	"github.com/gorilla/mux"
)

var client *http.Client

// CORSRouterDecorator applies CORS headers to a mux.Router
type CORSRouterDecorator struct {
	R               *mux.Router
	AcceptedOrigins []string
}

// ServeHTTP wraps the HTTP server enabling CORS headers.
// For more info about CORS, visit https://www.w3.org/TR/cors/
func (c *CORSRouterDecorator) ServeHTTP(rw http.ResponseWriter, req *http.Request) {
	if origin := req.Header.Get("Origin"); origin != "" {
		rw.Header().Set("Access-Control-Allow-Origin", strings.Join(c.AcceptedOrigins, ", "))
		rw.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
		rw.Header().Set("Access-Control-Allow-Headers", "Accept, Accept-Language, Content-Type, content-type")
	}
	// Stop here if its Preflighted OPTIONS request
	if req.Method == "OPTIONS" {
		return
	}

	c.R.ServeHTTP(rw, req)
}

func PublicKeyToIdString(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if medChainUtils.CheckError(err, w, r) {
		return
	}
	var request admin_messages.IdRequest
	err = json.Unmarshal(body, &request)
	if medChainUtils.CheckError(err, w, r) {
		return
	}
	if request.PublicKey == "" {
		medChainUtils.CheckError(errors.New("No public key was given"), w, r)
		return
	}

	identity, err := medChainUtils.LoadIdentityEd25519FromBytesWithErr([]byte(request.PublicKey))
	if medChainUtils.CheckError(err, w, r) {
		return
	}

	reply := admin_messages.IdReply{Identity: identity.String()}
	json_val, err := json.Marshal(&reply)
	if medChainUtils.CheckError(err, w, r) {
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(json_val)
	if medChainUtils.CheckError(err, w, r) {
		return
	}
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

	port, medchain_url := getFlags()

	router := mux.NewRouter()

	router.HandleFunc("/sign", processSignTransactionRequest)
	router.HandleFunc("/id", PublicKeyToIdString)

	http.ListenAndServe(":"+port, &CORSRouterDecorator{R: router, AcceptedOrigins: []string{medchain_url}})
}
