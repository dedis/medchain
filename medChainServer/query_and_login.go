package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/DPPH/MedChain/medChainUtils"
	"github.com/dedis/cothority"
	"github.com/dedis/cothority/omniledger/service"
	"github.com/dedis/onet/network"
)

type introspectionResponseQuery struct {
	Active      bool   `json:"active"`
	Query       string `json:"query"`
	QueryType   string `json:"query_type"`
	UserId      string `json:"user_id"`
	ProjectDesc string `json:"project_description"`
}

type introspectionResponseLogin struct {
	Active       bool   `json:"active"`
	ProjectsList string `json:"projects_list"`
	User         string `json:"user"`
}

/**
In the MedChainClient, the transaction is given in the header as a bas64 encoding
This function extracts the transaction
**/
func extractTransactionFromRequest(w http.ResponseWriter, r *http.Request) (*service.ClientTransaction, error) {
	// Fetch the transaction provided in the GET request
	transaction := r.Header.Get("transaction")
	fmt.Println("received transaction", transaction)
	transactionDecoded, err := base64.StdEncoding.DecodeString(transaction)
	if err != nil && transaction != "" {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return nil, err
	}
	// Load the transaction
	var testTransactionRetrieved *service.ClientTransaction
	_, tmp, err := network.Unmarshal(transactionDecoded, cothority.Suite)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return nil, err
	}
	testTransactionRetrieved, ok := tmp.(*service.ClientTransaction)
	if !ok {
		return nil, errors.New("could not retrieve the transaction")
	}
	return testTransactionRetrieved, nil
}

/**
Submits the transaction given in the header to the omniledger service
**/
func applyTransaction(w http.ResponseWriter, r *http.Request) {
	testTransactionRetrieved, err := extractTransactionFromRequest(w, r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Commit the transaction
	_, err = cl.AddTransaction(*testTransactionRetrieved)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	instID := service.NewInstanceID((*testTransactionRetrieved).Instructions[0].Hash())

	// Respond if the transaction succeeded
	pr, err := cl.WaitProof(instID, metaData.GenesisMsg.BlockInterval, nil)
	w.Header().Set("Content-Type", "text/plain")
	if err != nil || pr.InclusionProof.Match() != true {
		if err != nil {
			fmt.Println("wait proof failed ", err)
		} else {
			fmt.Println("proof failed")
		}
		w.Write([]byte("Failed to commit the transaction to the MedChain"))
	} else {
		w.Write([]byte(base64.StdEncoding.EncodeToString(instID.Slice())))
	}
}

type Message struct {
	Token string `json:"token"`
}

// validates the token
func doMedChainValidation(msg Message) (bool, string) {
	incomingTokenValue := msg.Token
	instIDbytes, err := base64.StdEncoding.DecodeString(incomingTokenValue)
	if err == nil && incomingTokenValue != "" {
		instID := service.NewInstanceID(instIDbytes)
		pr, err := cl.WaitProof(instID, metaData.GenesisMsg.BlockInterval, nil)
		if err == nil && pr.InclusionProof.Match() == true && pr.Verify(metaData.GenesisBlock.Skipblock.Hash) == nil {
			values, err := pr.InclusionProof.RawValues()
			if err == nil {
				return true, string(values[0][:])
			}
		}
	}
	return false, ""
}

// extracts the token from the body
func readToken(w http.ResponseWriter, r *http.Request) (*Message, error) {
	b, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}
	msg := new(Message)
	err = json.Unmarshal(b, msg)
	if err != nil {
		return nil, err
	}
	return msg, err
}

// Validate token through Omniledger client
func tokenIntrospectionQuery(w http.ResponseWriter, r *http.Request) {
	// Read the incoming token
	msg, err := readToken(w, r)
	medChainUtils.Check(err)

	// Do validation
	isActive, data := doMedChainValidation(*msg)

	// Retrieve data, if any
	query := ""
	queryType := ""
	userId := ""
	projectDesc := ""
	if data != "" {
		splitted := strings.Split(data, "......")
		query = splitted[0]
		queryType = splitted[1]
		userId = splitted[2]
		projectDesc = splitted[3]
	}

	// Respond according to the specs
	response := introspectionResponseQuery{isActive, query, queryType, userId, projectDesc}
	js, err := json.Marshal(response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

// Validate token through Omniledger client
func tokenIntrospectionLogin(w http.ResponseWriter, r *http.Request) {
	// Read the incoming token
	msg, err := readToken(w, r)
	medChainUtils.Check(err)

	// Do validation
	isActive, data := doMedChainValidation(*msg)

	// Retrieve data, if any
	user := ""
	projectsList := ""
	if data != "" {
		splitted := strings.Split(data, "......")
		user = splitted[0]
		projectsList = splitted[1]
	}

	// Respond according to the specs
	response := introspectionResponseLogin{isActive, projectsList, user}
	js, err := json.Marshal(response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}
