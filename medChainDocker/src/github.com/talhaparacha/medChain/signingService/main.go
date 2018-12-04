package main

import (
	"crypto/tls"
	"database/sql"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
	"github.com/talhaparacha/medChain/medChainUtils"
)

// HTTP Client with appropriate settings for later use
var client *http.Client

var db *sql.DB

var status_waiting = "WAITING"
var status_signed = "SIGNED"
var status_ready = "READY"
var status_submitted = "SUBMITTED"

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

	db, err := sql.Open("sqlite3", db_file)
	medChainUtils.Check(err)
	db.Prepare("SELECT * FROM ClientTransaction")

	// Register addresses
	http.HandleFunc("/new/transactions", newTransactions)
	http.HandleFunc("/transactions", getTransactions)
	http.HandleFunc("/sign/transactions", signTransactions)
	http.HandleFunc("/validate/transactions", validateTransactions)

	// Start server
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		panic(err)
	}
}

func validateTransactions(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	medChainUtils.Check(err)
	var transactions ExchangeMessage
	err = json.Unmarshal(body, &transactions)
	medChainUtils.Check(err)
	for _, transaction := range transactions.Transactions {
		setTransactionStatus(transaction.Uid, status_submitted)
	}
}

func newTransactions(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	medChainUtils.Check(err)
	var transactions ExchangeMessage
	err = json.Unmarshal(body, &transactions)
	medChainUtils.Check(err)
	for _, transaction := range transactions.Transactions {
		uid := registerNewTransaction(transaction)
		transaction.Uid = uid
	}
	reply, err := json.Marshal(transactions)
	medChainUtils.Check(err)
	w.Write(reply)
}

func registerNewTransaction(transaction TransactionData) string {
	uid := uuid.New().String()
	addNewTransToDB(uid, transaction.Transaction, transaction.Description, db)
	for _, signer := range transaction.Signers {
		addNewSignerToDB(uid, signer, db)
	}
	return uid
}

func addNewTransToDB(uid, transaction, description string, db *sql.DB) {
	stmt, err := db.Prepare("INSERT INTO ClientTransaction(uid, client_transaction, description, status) VALUES(?,?,?,?);")
	medChainUtils.Check(err)
	_, err = stmt.Exec(uid, transaction, description, status_waiting)
	medChainUtils.Check(err)
}

func addNewSignerToDB(uid, signer_identity string, db *sql.DB) {
	stmt, err := db.Prepare("INSERT INTO SignatureStatus(transaction_uid, signer_identity, status) VALUES(?,?,?);")
	medChainUtils.Check(err)
	_, err = stmt.Exec(uid, signer_identity, status_waiting)
	medChainUtils.Check(err)
}

func getTransactions(w http.ResponseWriter, r *http.Request) {
	identity := strings.Join(r.URL.Query()["identity"], "")
	transactions := getTransactionsWaitingForId(identity, db)
	reply := ExchangeMessage{Transactions: transactions}
	body, err := json.Marshal(reply)
	medChainUtils.Check(err)
	w.Write(body)
}

func getTransactionsWaitingForId(identity string, db *sql.DB) []TransactionData {
	statement := "SELECT uid, description, client_transaction FROM ClientTransaction WHERE EXISTS(SELECT * FROM SignatureStatus WHERE transaction_uid=uid AND signer_identity=? AND status=?);"
	stmt, err := db.Prepare(statement)
	medChainUtils.Check(err)
	rows, err := stmt.Query(identity, status_waiting)
	medChainUtils.Check(err)
	transactions := []TransactionData{}
	for rows.Next() {
		var uid, description, transactionString string
		err = rows.Scan(&uid, &description, &transactionString)
		medChainUtils.Check(err)
		signers := getSignersOfTransaction(uid)
		transaction := TransactionData{Uid: uid, Description: description, Transaction: transactionString, Signers: signers}
		transactions = append(transactions, transaction)
	}
	rows.Close()
	return transactions
}

func getSignersOfTransaction(uid string) []string {
	statement := "SELECT signer_identity FROM SignatureStatus WHERE transaction_uid=?;"
	stmt, err := db.Prepare(statement)
	medChainUtils.Check(err)
	rows, err := stmt.Query(uid)
	medChainUtils.Check(err)
	signers := []string{}
	for rows.Next() {
		var identity string
		err = rows.Scan(&identity)
		medChainUtils.Check(err)
		signers = append(signers, identity)
	}
	rows.Close()
	return signers
}

func signTransactions(w http.ResponseWriter, r *http.Request) {
	identity := strings.Join(r.URL.Query()["identity"], "")
	body, err := ioutil.ReadAll(r.Body)
	medChainUtils.Check(err)
	var transactions ExchangeMessage
	err = json.Unmarshal(body, &transactions)
	medChainUtils.Check(err)
	transactions_ready := []TransactionData{}
	for _, transaction := range transactions.Transactions {
		ready := signTransaction(identity, transaction)
		if ready {
			transactions_ready = append(transactions_ready, transaction)
		}
	}
	reply := ExchangeMessage{Transactions: transactions_ready}
	reply_body, err := json.Marshal(reply)
	medChainUtils.Check(err)
	w.Write(reply_body)
}

func getTransactionsReady() []TransactionData {
	statement := "SELECT uid,description,client_transaction FROM ClientTransaction WHERE status=?"
	stmt, err := db.Prepare(statement)
	medChainUtils.Check(err)
	rows, err := stmt.Query(status_ready)
	medChainUtils.Check(err)
	transactions := []TransactionData{}
	for rows.Next() {
		var uid, description, transactionString string
		err = rows.Scan(&uid, &description, &transactionString)
		medChainUtils.Check(err)
		signers := getSignersOfTransaction(uid)
		transaction := TransactionData{Uid: uid, Description: description, Transaction: transactionString, Signers: signers}
		transactions = append(transactions, transaction)
	}
	rows.Close()
	return transactions
}

func signTransaction(identity string, transaction TransactionData) bool {
	uid := transaction.Uid
	changeTransactionValue(uid, transaction.Transaction)
	changeStatusToSigned(uid, identity)
	return checkIfReady(uid)
}

func checkIfReady(uid string) bool {
	statuses := getSignatureStatus(uid)
	if isReady(statuses) {
		setTransactionStatus(uid, status_ready)
		return true
	}
	return false
}

func isReady(statuses []string) bool {
	for _, status := range statuses {
		if status != status_signed {
			return false
		}
	}
	return true
}

func setTransactionStatus(uid, status string) {
	statement := "UPDATE ClientTransaction SET status=? WHERE uid=?;"
	stmt, err := db.Prepare(statement)
	medChainUtils.Check(err)
	_, err = stmt.Exec(status, uid)
	medChainUtils.Check(err)
}

func getSignatureStatus(uid string) []string {
	statement := "SELECT status FROM SignatureStatus WHERE transaction_uid=?;"
	stmt, err := db.Prepare(statement)
	medChainUtils.Check(err)
	rows, err := stmt.Query(uid)
	medChainUtils.Check(err)
	statuses := []string{}
	for rows.Next() {
		var status string
		err = rows.Scan(&status)
		medChainUtils.Check(err)
		statuses = append(statuses, status)
	}
	rows.Close()
	return statuses
}

func changeStatusToSigned(uid, identity string) {
	statement := "UPDATE SignatureStatus SET status=? WHERE transaction_uid=? AND signer_identity=?;"
	stmt, err := db.Prepare(statement)
	medChainUtils.Check(err)
	_, err = stmt.Exec(status_signed, uid, identity)
	medChainUtils.Check(err)
}

func changeTransactionValue(uid, transaction string) {
	statement := "UPDATE ClientTransaction SET client_transaction=? WHERE uid=?;"
	stmt, err := db.Prepare(statement)
	medChainUtils.Check(err)
	_, err = stmt.Exec(transaction, uid)
	medChainUtils.Check(err)
}

type TransactionData struct {
	Uid         string
	Description string
	Signers     []string
	Transaction string
}

type ExchangeMessage struct {
	Transactions []TransactionData
}
