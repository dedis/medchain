package main

import (
	"crypto/tls"
	"net/http"
)

func main() {
	// Setup HTTP client
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client = &http.Client{Transport: tr}
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		req.Header.Add("Authorization", via[0].Header.Get("Authorization"))
		return nil
	}

	// Register addresses
	http.Handle("/forms/", http.StripPrefix("/forms/", http.FileServer(http.Dir("templates/static"))))
	http.HandleFunc("/forms/landing", landing)
	http.HandleFunc("/forms/projects", projects)
	http.HandleFunc("/forms/logQuery", logQuery)

	// Start server
	if err := http.ListenAndServe(":8282", nil); err != nil {
		panic(err)
	}
}
