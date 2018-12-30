package main

import (
	"crypto/tls"
	"database/sql"
	"net/http"

	"github.com/DPPH/MedChain/medChainUtils"
	_ "github.com/mattn/go-sqlite3"
)

// HTTP Client with appropriate settings for later use
var client *http.Client

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
	http.HandleFunc("/info/action", getActionInfo)
	http.HandleFunc("/list/actions", getUserActions)
	http.HandleFunc("/list/actions/waiting", getActionsWaiting)

	// Start server
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		panic(err)
	}
}

// func validateTransactions(w http.ResponseWriter, r *http.Request) {
// 	fmt.Println("/validate/transactions")
// 	body, err := ioutil.ReadAll(r.Body)
// 	medChainUtils.Check(err)
// 	var transactions medChainUtils.ExchangeMessage
// 	err = json.Unmarshal(body, &transactions)
// 	medChainUtils.Check(err)
// 	for _, transaction := range transactions.Transactions {
// 		setTransactionStatus(transaction.Uid, status_submitted)
// 	}
// }
//
// func newTransactions(w http.ResponseWriter, r *http.Request) {
// 	fmt.Println("/add/transactions")
// 	body, err := ioutil.ReadAll(r.Body)
// 	medChainUtils.Check(err)
// 	var transactions medChainUtils.ExchangeMessage
// 	err = json.Unmarshal(body, &transactions)
// 	medChainUtils.Check(err)
// 	for _, transaction := range transactions.Transactions {
// 		uid := registerNewTransaction(transaction)
// 		transaction.Uid = uid
// 	}
// 	reply, err := json.Marshal(transactions)
// 	medChainUtils.Check(err)
// 	w.Write(reply)
// }
//
//
//
// func addNewSignerToDB(uid, signer_identity string, db *sql.DB) {
// 	stmt, err := db.Prepare("INSERT INTO SignatureStatus(transaction_uid, signer_identity, status) VALUES(?,?,?);")
// 	medChainUtils.Check(err)
// 	_, err = stmt.Exec(uid, signer_identity, status_waiting)
// 	medChainUtils.Check(err)
// }
//
// func getTransactions(w http.ResponseWriter, r *http.Request) {
// 	fmt.Println("/transactions")
// 	identity := strings.Join(r.URL.Query()["identity"], "")
// 	transactions := getTransactionsWaitingForId(identity, db)
// 	reply := medChainUtils.ExchangeMessage{Transactions: transactions}
// 	body, err := json.Marshal(reply)
// 	medChainUtils.Check(err)
// 	w.Write(body)
// }
//
// func getTransactionsWaitingForId(identity string, db *sql.DB) []medChainUtils.TransactionData {
// 	statement := "SELECT uid, description, client_transaction, threshold FROM ClientTransaction WHERE EXISTS(SELECT * FROM SignatureStatus WHERE transaction_uid=uid AND signer_identity=? AND status=?);"
// 	stmt, err := db.Prepare(statement)
// 	medChainUtils.Check(err)
// 	rows, err := stmt.Query(identity, status_waiting)
// 	medChainUtils.Check(err)
// 	transactions := []medChainUtils.TransactionData{}
// 	for rows.Next() {
// 		var uid, description, transactionString string
// 		var threshold int
// 		err = rows.Scan(&uid, &description, &transactionString, &threshold)
// 		medChainUtils.Check(err)
// 		signers := getSignersOfTransaction(uid)
// 		transaction := medChainUtils.TransactionData{Uid: uid, Description: description, Transaction: transactionString, Signers: signers, Threshold: threshold}
// 		transactions = append(transactions, transaction)
// 	}
// 	rows.Close()
// 	return transactions
// }
//
// func getSignersOfTransaction(uid string) []string {
// 	statement := "SELECT signer_identity FROM SignatureStatus WHERE transaction_uid=?;"
// 	stmt, err := db.Prepare(statement)
// 	medChainUtils.Check(err)
// 	rows, err := stmt.Query(uid)
// 	medChainUtils.Check(err)
// 	signers := []string{}
// 	for rows.Next() {
// 		var identity string
// 		err = rows.Scan(&identity)
// 		medChainUtils.Check(err)
// 		signers = append(signers, identity)
// 	}
// 	rows.Close()
// 	return signers
// }
//
// func signTransactions(w http.ResponseWriter, r *http.Request) {
// 	fmt.Println("/sign/transactions")
// 	identity := strings.Join(r.URL.Query()["identity"], "")
// 	body, err := ioutil.ReadAll(r.Body)
// 	medChainUtils.Check(err)
// 	var transactions medChainUtils.ExchangeMessage
// 	err = json.Unmarshal(body, &transactions)
// 	medChainUtils.Check(err)
// 	for _, transaction := range transactions.Transactions {
// 		signTransaction(identity, transaction)
// 	}
// 	transactions_ready := getTransactionsReady()
// 	reply := medChainUtils.ExchangeMessage{Transactions: transactions_ready}
// 	reply_body, err := json.Marshal(reply)
// 	medChainUtils.Check(err)
// 	w.Write(reply_body)
// }
//
// func getTransactionsReady() []medChainUtils.TransactionData {
// 	statement := "SELECT uid,description,client_transaction FROM ClientTransaction WHERE status=?"
// 	stmt, err := db.Prepare(statement)
// 	medChainUtils.Check(err)
// 	rows, err := stmt.Query(status_ready)
// 	medChainUtils.Check(err)
// 	transactions := []medChainUtils.TransactionData{}
// 	for rows.Next() {
// 		var uid, description, transactionString string
// 		err = rows.Scan(&uid, &description, &transactionString)
// 		medChainUtils.Check(err)
// 		signers := getSignersOfTransaction(uid)
// 		transaction := medChainUtils.TransactionData{Uid: uid, Description: description, Transaction: transactionString, Signers: signers}
// 		transactions = append(transactions, transaction)
// 	}
// 	rows.Close()
// 	return transactions
// }
//
// func signTransaction(identity string, transaction medChainUtils.TransactionData) bool {
// 	uid := transaction.Uid
// 	changeTransactionValue(uid, transaction.Transaction)
// 	changeStatusToSigned(uid, identity)
// 	return checkIfReady(uid, transaction.Threshold)
// }
//
// func checkIfReady(uid string, threshold int) bool {
// 	statuses := getSignatureStatus(uid)
// 	if isReady(statuses, threshold) {
// 		setTransactionStatus(uid, status_ready)
// 		return true
// 	}
// 	return false
// }
//
// func isReady(statuses []string, threshold int) bool {
// 	count := 0
// 	for _, status := range statuses {
// 		if status == status_signed {
// 			count += 1
// 		}
// 	}
// 	return count >= threshold
// }
//
// func setTransactionStatus(uid, status string) {
// 	statement := "UPDATE ClientTransaction SET status=? WHERE uid=?;"
// 	stmt, err := db.Prepare(statement)
// 	medChainUtils.Check(err)
// 	_, err = stmt.Exec(status, uid)
// 	medChainUtils.Check(err)
// }
//
// func getSignatureStatus(uid string) []string {
// 	statement := "SELECT status FROM SignatureStatus WHERE transaction_uid=?;"
// 	stmt, err := db.Prepare(statement)
// 	medChainUtils.Check(err)
// 	rows, err := stmt.Query(uid)
// 	medChainUtils.Check(err)
// 	statuses := []string{}
// 	for rows.Next() {
// 		var status string
// 		err = rows.Scan(&status)
// 		medChainUtils.Check(err)
// 		statuses = append(statuses, status)
// 	}
// 	rows.Close()
// 	return statuses
// }
//
// func changeStatusToSigned(uid, identity string) {
// 	statement := "UPDATE SignatureStatus SET status=? WHERE transaction_uid=? AND signer_identity=?;"
// 	stmt, err := db.Prepare(statement)
// 	medChainUtils.Check(err)
// 	_, err = stmt.Exec(status_signed, uid, identity)
// 	medChainUtils.Check(err)
// }
//
// func changeTransactionValue(uid, transaction string) {
// 	statement := "UPDATE ClientTransaction SET client_transaction=?, count=count+1 WHERE uid=?;"
// 	stmt, err := db.Prepare(statement)
// 	medChainUtils.Check(err)
// 	_, err = stmt.Exec(transaction, uid)
// 	medChainUtils.Check(err)
// }
