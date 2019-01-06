package main

import (
	"crypto/tls"
	"database/sql"
	"net/http"

	"github.com/DPPH/MedChain/medChainUtils"
	_ "github.com/mattn/go-sqlite3"
)

/**
The signature service is a centralized database
to share transactions that need to be signed
**/

// HTTP Client with appropriate settings for later use
var client *http.Client

// The database connection
var db *sql.DB

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

	port, db_file := getFlags()

	var err error
	db, err = sql.Open("sqlite3", db_file)
	medChainUtils.Check(err)

	// Register addresses
	http.HandleFunc("/add/action", addAction)
	http.HandleFunc("/approve/action", ApproveAction)
	http.HandleFunc("/deny/action", DenyAction)
	http.HandleFunc("/update/action/done", DoneAction)
	http.HandleFunc("/update/action/cancel", CancelAction)
	http.HandleFunc("/info/action", getActionInfo)
	http.HandleFunc("/list/actions", getUserActions)
	http.HandleFunc("/list/actions/waiting", getActionsWaiting)

	// Start server
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		panic(err)
	}
}
