package main

import (
	b64 "encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/dedis/cothority"
	"github.com/dedis/cothority/omniledger/darc"
	"github.com/dedis/cothority/omniledger/service"
	"github.com/dedis/onet"
	"github.com/dedis/onet/network"
	"github.com/talhaparacha/medChain/medChainUtils"
)

// Configure cothority here...
var local = onet.NewTCPTest(cothority.Suite)
var _, roster, _ = local.GenTree(3, true)
var cl = service.NewClient()

// Admins, Managers and Users as per the context defined in system diagram
var admins = []darc.Signer{}
var managers = make(map[string][]darc.Signer)
var users = make(map[string][]darc.Identity)

var adminsDarcsMap = make(map[string]*darc.Darc)
var managersListDarcsMap = make(map[string]*darc.Darc)
var managersDarcsMap = make(map[string]*darc.Darc)
var usersDarcsMap = make(map[string]*darc.Darc)
var usersListDarcsMap = make(map[string]*darc.Darc)

var keysDirectory = "keys"

var configFileName = "conf/conf.json"

// Genesis block
var genesisMsg *service.CreateGenesisBlock
var genesisBlock *service.CreateGenesisBlockResponse

// Stuff required by the token introspection services
var allUsersDarc *darc.Darc
var allManagersDarc *darc.Darc
var userProjectsMapInstanceID service.InstanceID
var err error
var systemStart = false

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

func createTransactionForNewDARC(baseDarc *darc.Darc, rules darc.Rules, description string) (*service.ClientTransaction, *darc.Darc, error) {
	// Create a transaction to spawn a DARC
	tempDarc := darc.NewDarc(rules, []byte(description))
	tempDarcBuff, err := tempDarc.ToProto()
	if err != nil {
		return nil, nil, err
	}
	ctx := service.ClientTransaction{
		Instructions: []service.Instruction{{
			InstanceID: service.NewInstanceID(baseDarc.GetBaseID()),
			Nonce:      service.Nonce{},
			Index:      0,
			Length:     1,
			Spawn: &service.Spawn{
				ContractID: service.ContractDarcID,
				Args: []service.Argument{{
					Name:  "darc",
					Value: tempDarcBuff,
				}},
			},
		}},
	}
	return &ctx, tempDarc, nil
}

func createTransactionForUpdatingDARC(baseDarc *darc.Darc, rules darc.Rules, description string) (*service.ClientTransaction, *darc.Darc, error) {
	// Create a transaction to spawn a DARC
	tempDarc := darc.NewDarc(rules, []byte(description))
	tempDarcBuff, err := tempDarc.ToProto()
	if err != nil {
		return nil, nil, err
	}
	ctx := service.ClientTransaction{
		Instructions: []service.Instruction{{
			InstanceID: service.NewInstanceID(baseDarc.GetBaseID()),
			Nonce:      service.Nonce{},
			Index:      0,
			Length:     1,
			Spawn: &service.Spawn{
				ContractID: service.ContractDarcID,
				Args: []service.Argument{{
					Name:  "darc",
					Value: tempDarcBuff,
				}},
			},
		}},
	}
	return &ctx, tempDarc, nil
}

func submitSignedTransactionForNewDARC(client *service.Client, tempDarc *darc.Darc, interval time.Duration, ctx *service.ClientTransaction) (*darc.Darc, error) {
	// Commit transaction
	if _, err := client.AddTransaction(*ctx); err != nil {
		return nil, err
	}

	// Verify DARC creation before returning its reference
	instID := service.NewInstanceID(tempDarc.GetBaseID())
	pr, err := client.WaitProof(instID, interval, nil)
	if err != nil || pr.InclusionProof.Match() == false {
		fmt.Println("Error at transaction submission")
		return nil, err
	}

	return tempDarc, nil
}

func createDarc(client *service.Client, baseDarc *darc.Darc, interval time.Duration, rules darc.Rules, description string, signers ...darc.Signer) (*darc.Darc, error) {
	ctx, tempDarc, err := createTransactionForNewDARC(baseDarc, rules, description)
	if err != nil {
		fmt.Println("Error at transaction creation")
		return nil, err
	}
	if err = ctx.Instructions[0].SignBy(baseDarc.GetBaseID(), signers...); err != nil {
		fmt.Println("Error at transaction signature")
		return nil, err
	}
	return submitSignedTransactionForNewDARC(client, tempDarc, interval, ctx)
}

// Simple web server
func sayHello(w http.ResponseWriter, r *http.Request) {
	message := r.URL.Path
	message = strings.TrimPrefix(message, "/")
	message = "Hello " + message
	w.Write([]byte(message))
}

func applyTransaction(w http.ResponseWriter, r *http.Request) {
	// Fetch the transaction provided in the GET request
	transaction := r.Header.Get("transaction")
	transactionDecoded, err := b64.StdEncoding.DecodeString(transaction)
	if err != nil && transaction != "" {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Load the transaction
	var testTransactionRetrieved *service.ClientTransaction
	_, tmp, err := network.Unmarshal(transactionDecoded, cothority.Suite)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	testTransactionRetrieved, ok := tmp.(*service.ClientTransaction)
	if !ok {
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
	pr, err := cl.WaitProof(instID, genesisMsg.BlockInterval, nil)
	w.Header().Set("Content-Type", "text/plain")
	if err != nil || pr.InclusionProof.Match() != true {
		w.Write([]byte("Failed to commit the transaction to the MedChain"))
	} else {
		w.Write([]byte(b64.StdEncoding.EncodeToString(instID.Slice())))
	}
}

type Message struct {
	Token string `json:"token"`
}

func doMedChainValidation(msg Message) (bool, string) {
	incomingTokenValue := msg.Token
	instIDbytes, err := b64.StdEncoding.DecodeString(incomingTokenValue)
	if err == nil && incomingTokenValue != "" {
		instID := service.NewInstanceID(instIDbytes)
		pr, err := cl.WaitProof(instID, genesisMsg.BlockInterval, nil)
		if err == nil && pr.InclusionProof.Match() == true && pr.Verify(genesisBlock.Skipblock.Hash) == nil {
			values, err := pr.InclusionProof.RawValues()
			if err == nil {
				return true, string(values[0][:])
			}
		}
	}
	return false, ""
}

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

// Communicate ID of the allUsersDarc if the system is running
func info(w http.ResponseWriter, r *http.Request) {
	var js []byte
	var temp map[string]string

	if systemStart {
		temp = map[string]string{
			"all_users_darc":     b64.StdEncoding.EncodeToString(allUsersDarc.GetBaseID()),
			"user_projects_maps": b64.StdEncoding.EncodeToString(userProjectsMapInstanceID.Slice()),
			"error":              "",
		}
	} else {
		temp = map[string]string{"error": "MedChain not started yet"}
	}

	js, _ = json.Marshal(temp)
	w.Header().Set("Content-Type", "application/json")
	w.Write(js)
}

// Simple web server
func start(w http.ResponseWriter, r *http.Request) {
	if !systemStart {
		// Initiate Omniledger with the MedCo context
		startSystem()
		systemStart = true
	}
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte("Success"))
}

func main() {
	// Run one time to generate all the keys for our context
	//medChainUtils.InitKeys(3, "keys/admins")
	//medChainUtils.InitKeys(3, "keys/managers")
	//medChainUtils.InitKeys(3, "keys/users")

	http.HandleFunc("/", sayHello)
	http.HandleFunc("/start", start)
	http.HandleFunc("/info", info)
	http.HandleFunc("/info/manager", GetManagerInfo)
	http.HandleFunc("/add/user", newUserPart1)
	http.HandleFunc("/add/manager", newManager)
	http.HandleFunc("/add/administrator", newAdministrator)
	http.HandleFunc("/applyTransaction", applyTransaction)
	http.HandleFunc("/tokenIntrospectionLogin", tokenIntrospectionLogin)
	http.HandleFunc("/tokenIntrospectionQuery", tokenIntrospectionQuery)
	if err := http.ListenAndServe(":8989", nil); err != nil {
		panic(err)
	}

	// Wrap Omniledger service
	local.WaitDone(genesisMsg.BlockInterval)
	local.CloseAll()
}
