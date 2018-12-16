package main

import (
	b64 "encoding/base64"
	"encoding/json"
	"errors"
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
	"github.com/talhaparacha/medChain/medChainServer/messages"
	"github.com/talhaparacha/medChain/medChainServer/metadata"
	"github.com/talhaparacha/medChain/medChainUtils"
)

// Configure cothority here...
var local = onet.NewTCPTest(cothority.Suite)
var _, roster, _ = local.GenTree(3, true)
var cl = service.NewClient()

// var baseIdToDarcMap = make(map[string]*darc.Darc)
// var darcIdToBaseIdMap = make(map[string]string)
// var IdToHospitalIdMap = make(map[string]string)
//
// var superAdminsDarcsMap = make(map[string]string)
//
// var adminsDarcsMap = make(map[string]string)
// var adminsListDarcsMap = make(map[string]string)
//
// var managersDarcsMap = make(map[string]string)
// var managersListDarcsMap = make(map[string]string)
//
// var usersDarcsMap = make(map[string]string)
// var usersListDarcsMap = make(map[string]string)
//
// var powerfulDarcsMap = make(map[string]string)
//
// var projectsDarcsMap = make(map[string]string)

var keysDirectory = "keys"

var configFileName = "conf/conf.json"

// // Genesis block
// var genesisMsg *service.CreateGenesisBlock
// var genesisBlock *service.CreateGenesisBlockResponse
//
// // Stuff required by the token introspection services
// var allSuperAdminsDarc *darc.Darc
// var allSuperAdminsBaseID string
// var allAdminsDarc *darc.Darc
// var allAdminsBaseID string
// var allManagersDarc *darc.Darc
// var allManagersBaseID string
// var allUsersDarc *darc.Darc
// var allUsersBaseID string

var metaData *metadata.Metadata

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

func submitSignedTransactionForEvolveDARC(client *service.Client, newDarc *darc.Darc, interval time.Duration, ctx *service.ClientTransaction) (*darc.Darc, error) {
	// Commit transaction
	if _, err := client.AddTransaction(*ctx); err != nil {
		return nil, err
	}
	// Verify DARC creation before returning its reference
	instID := service.NewInstanceID(newDarc.GetBaseID())
	darcBuf, err := newDarc.ToProto()
	if err != nil {
		return nil, err
	}
	pr, err := client.WaitProof(instID, interval, darcBuf)
	if err != nil || pr.InclusionProof.Match() == false {
		if err != nil {
			fmt.Println("error", err)
		} else {
			fmt.Println("wrong proof")
		}
		fmt.Println("Error at transaction submission")
		return nil, err
	}
	return newDarc, nil
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

// func applyNewDarcTransaction(w http.ResponseWriter, r *http.Request) {
// 	testTransactionRetrieved, err := extractTransactionFromRequest(w, r)
// 	if err != nil {
// 		fmt.Println("failed to retrieve transaction")
// 		http.Error(w, err.Error(), http.StatusInternalServerError)
// 		return
// 	}
// 	darc, err := extractNewDarcFromTransaction(testTransactionRetrieved)
// 	if err != nil {
// 		fmt.Println("failed to extract darc")
// 		http.Error(w, err.Error(), http.StatusInternalServerError)
// 		return
// 	}
// 	tempDarc, err := submitSignedTransactionForNewDARC(cl, darc, genesisMsg.BlockInterval, testTransactionRetrieved)
// 	if err != nil {
// 		fmt.Println("failed to submit new darc transaction")
// 		w.Write([]byte("Failed to commit the transaction to the MedChain"))
// 		return
// 	} else {
// 		darcBaseID := medChainUtils.IDToHexString(tempDarc.GetBaseID())
// 		baseIdToDarcMap[darcBaseID] = tempDarc
// 		darcIdToBaseIdMap[tempDarc.GetIdentityString()] = darcBaseID
// 		w.Write([]byte("Success " + tempDarc.GetIdentityString()))
// 	}
// }

// func applyEvolveDarcTransaction(w http.ResponseWriter, r *http.Request) {
// 	testTransactionRetrieved, err := extractTransactionFromRequest(w, r)
// 	if err != nil {
// 		fmt.Println("failed to retrieve transaction")
// 		http.Error(w, err.Error(), http.StatusInternalServerError)
// 		return
// 	}
// 	darc, err := extractEvolvedDarcFromTransaction(testTransactionRetrieved)
// 	if err != nil {
// 		fmt.Println("failed to extract darc")
// 		http.Error(w, err.Error(), http.StatusInternalServerError)
// 		return
// 	}
// 	tempDarc, err := submitSignedTransactionForEvolveDARC(cl, darc, genesisMsg.BlockInterval, testTransactionRetrieved)
// 	if err != nil {
// 		fmt.Println("failed to submit evolve darc transaction")
// 		w.Write([]byte("Failed to commit the transaction to the MedChain"))
// 		return
// 	} else {
// 		darcBaseID := medChainUtils.IDToHexString(tempDarc.GetBaseID())
// 		baseIdToDarcMap[darcBaseID] = tempDarc
// 		darcIdToBaseIdMap[tempDarc.GetIdentityString()] = darcBaseID
// 		w.Write([]byte("Success " + tempDarc.GetIdentityString()))
// 	}
// }

// func extractEvolvedDarcFromTransaction(transaction *service.ClientTransaction) (*darc.Darc, error) {
// 	instruction := transaction.Instructions[0]
// 	invoke := instruction.Invoke
// 	args := invoke.Args
// 	arg := args[0]
// 	darcBuf := arg.Value
// 	newDarc, err := darc.NewFromProtobuf(darcBuf)
// 	return newDarc, err
// }

// func extractNewDarcFromTransaction(transaction *service.ClientTransaction) (*darc.Darc, error) {
// 	instruction := transaction.Instructions[0]
// 	spawn := instruction.Spawn
// 	args := spawn.Args
// 	arg := args[0]
// 	darcBuf := arg.Value
// 	newDarc, err := darc.NewFromProtobuf(darcBuf)
// 	return newDarc, err
// }

func extractTransactionFromRequest(w http.ResponseWriter, r *http.Request) (*service.ClientTransaction, error) {
	// Fetch the transaction provided in the GET request
	transaction := r.Header.Get("transaction")
	fmt.Println("received transaction", transaction)
	transactionDecoded, err := b64.StdEncoding.DecodeString(transaction)
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
		allUsersDarc := metaData.BaseIdToDarcMap[metaData.AllUsersDarcBaseId]
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
		metaData = startSystem()
		systemStart = true
	}
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte("Success"))
}

func getDarcFromId(id string, baseIdToDarcMap map[string]*darc.Darc, mapId map[string]string) (*darc.Darc, bool) {
	darcId, ok := mapId[id]
	if ok {
		darc, ok := baseIdToDarcMap[darcId]
		return darc, ok
	}
	return nil, ok
}

func GetDarcInfo(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if medChainUtils.CheckError(err, w, r) {
		return
	}
	var request messages.DarcInfoRequest
	err = json.Unmarshal(body, &request)
	if medChainUtils.CheckError(err, w, r) {
		return
	}
	var baseId string
	if request.BaseId != "" {
		baseId = request.BaseId
	} else if request.DarcId != "" {
		var ok bool
		baseId, ok = metaData.DarcIdToBaseIdMap[request.DarcId]
		if !ok {
			medChainUtils.CheckError(errors.New("No darc with this id"), w, r)
			return
		}
	} else {
		medChainUtils.CheckError(errors.New("No darc id Nor base id was given"), w, r)
		return
	}
	darc, ok := metaData.BaseIdToDarcMap[baseId]
	if !ok {
		medChainUtils.CheckError(errors.New("No darc with this base id"), w, r)
		return
	}
	reply := messages.DarcInfoReply{Description: string(darc.Description)}
	json_val, err := json.Marshal(&reply)
	if medChainUtils.CheckError(err, w, r) {
		return
	}
	_, err = w.Write(json_val)
	if medChainUtils.CheckError(err, w, r) {
		return
	}
}

func getInfo(w http.ResponseWriter, r *http.Request, baseIdToDarcMap map[string]*darc.Darc, map_id map[string]string, list_of_map_subordinates []map[string]string) {
	identity := strings.Join(r.URL.Query()["identity"], "")
	mainDarc, ok := getDarcFromId(identity, baseIdToDarcMap, map_id)
	if !ok {
		fmt.Println("Failed To Retrieve Info")
		http.Error(w, "", http.StatusNotFound)
		return
	}
	subordinatesDarcsList := []*darc.Darc{}
	for _, map_subordinates := range list_of_map_subordinates {
		subordinatesDarc, ok := getDarcFromId(identity, baseIdToDarcMap, map_subordinates)
		if !ok {
			fmt.Println("Failed To Retrieve Info")
			http.Error(w, "", http.StatusNotFound)
			return
		}
		subordinatesDarcsList = append(subordinatesDarcsList, subordinatesDarc)
	}

	reply := medChainUtils.UserInfoReply{MainDarc: mainDarc, SubordinatesDarcsList: subordinatesDarcsList}
	jsonVal, err := json.Marshal(reply)
	if err != nil {
		panic(err)
	}
	w.Write(jsonVal)
}

func main() {
	// Run one time to generate all the keys for our context
	//medChainUtils.InitKeys(3, "keys/admins")
	//medChainUtils.InitKeys(3, "keys/managers")
	//medChainUtils.InitKeys(3, "keys/users")
	port, testConf := getFlags()
	if testConf {
		configFileName = "conf/test_conf.json"
	}
	http.HandleFunc("/", sayHello)
	http.HandleFunc("/start", start)
	http.HandleFunc("/info", info)
	http.HandleFunc("/info/super_admin", GetSuperAdminInfo)
	http.HandleFunc("/info/admin", GetAdminInfo)
	http.HandleFunc("/info/manager", GetManagerInfo)
	http.HandleFunc("/info/user", GetUserInfo)
	http.HandleFunc("/info/darc", GetDarcInfo)
	// http.HandleFunc("/add/darc", applyNewDarcTransaction)
	// http.HandleFunc("/evolve/darc", applyEvolveDarcTransaction)
	// http.HandleFunc("/metadata/add/user", NewUserMetadata)
	// http.HandleFunc("/metadata/add/manager", NewManagerMetadata)
	// http.HandleFunc("/metadata/add/admin", NewAdminMetadata)
	http.HandleFunc("/applyTransaction", applyTransaction)
	http.HandleFunc("/tokenIntrospectionLogin", tokenIntrospectionLogin)
	http.HandleFunc("/tokenIntrospectionQuery", tokenIntrospectionQuery)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		panic(err)
	}
	// Wrap Omniledger service
	local.WaitDone(metaData.GenesisMsg.BlockInterval)
	local.CloseAll()
}
