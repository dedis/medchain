package main

import (
	"crypto/tls"
	"net/http"
)

var client *http.Client

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
	// http.Handle("/templates/static/", http.StripPrefix("/templates/static/", http.FileServer(http.Dir("templates/static"))))
	http.HandleFunc("/sign", processSignTransactionRequest)

	// Start server
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		panic(err)
	}
}
