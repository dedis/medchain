package main

import (
	"crypto/tls"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"

	"github.com/talhaparacha/medChain/medChainAdmin/admin_messages"
	"github.com/talhaparacha/medChain/medChainUtils"
)

var client *http.Client

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

	identity := medChainUtils.LoadIdentityEd25519FromBytes([]byte(request.PublicKey))

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

	port := getFlags()

	// Register addresses
	http.HandleFunc("/sign", processSignTransactionRequest)
	http.HandleFunc("/id", PublicKeyToIdString)

	// Start server
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		panic(err)
	}
}
