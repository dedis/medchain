package main

import (
	"github.com/talhaparacha/medChain/medChainUtils"
	"net/http"
	"github.com/dedis/cothority/omniledger/service"
	"github.com/dedis/cothority/omniledger/darc/expression"
	"time"
	"github.com/dedis/cothority/omniledger/darc"
	"github.com/dedis/onet"
	"github.com/dedis/cothority"
	"github.com/dedis/cothority/omniledger/contracts"
	"strings"
     b64 "encoding/base64"
	"encoding/json"
	"io/ioutil"
	"strconv"
	"github.com/dedis/onet/network"
)

// Configure cothority here...
var local = onet.NewTCPTest(cothority.Suite)
var servers, roster, _ = local.GenTree(3, true)
var cl = service.NewClient()

// Admins, Managers and Users as per the context defined in system diagram
var admins = []darc.Signer{}
var managers = []darc.Signer{}
var users = []darc.Identity{}

var keysDirectory = "keys"

// Genesis block
var genesisMsg *service.CreateGenesisBlock
// Stuff required by the login service
var allUsersDarc *darc.Darc
var userProjectsMapInstanceID service.InstanceID
var err error
var systemStart = false

type introspectionResponseQuery struct {
	Active bool `json:"active"`
	Query string `json:"query"`
	QueryType string `json:"query_type"`
	UserId string `json:"user_id"`
	ProjectDesc string `json:"project_description"`
}

type introspectionResponseLogin struct {
	Active bool `json:"active"`
	ProjectsList string `json:"projects_list"`
	User string `json:"user"`
}

func startSystem() {
	// We need to load suitable keys to initialize the system DARCs as per our context
	for i := 0; i < 3; i++ {
		admins = append(admins, medChainUtils.LoadSignerEd25519(keysDirectory + "/admins/" + strconv.Itoa(i) + "_public",
			keysDirectory + "/admins/" + strconv.Itoa(i) + "_private"))
		managers = append(managers, medChainUtils.LoadSignerEd25519(keysDirectory + "/managers/" + strconv.Itoa(i) + "_public",
			keysDirectory + "/managers/" + strconv.Itoa(i) + "_private"))
		users = append(users, medChainUtils.LoadIdentityEd25519(keysDirectory + "/users/" + strconv.Itoa(i) + "_public"))
	}

	// Create Genesis block
	genesisMsg, err = service.DefaultGenesisMsg(service.CurrentVersion, roster,
		[]string{}, admins[0].Identity(), admins[1].Identity(), admins[2].Identity())
	if err != nil {
		panic(err)
	}
	gDarc := &genesisMsg.GenesisDarc
	gDarc.Rules.UpdateSign(expression.InitAndExpr(admins[0].Identity().String(),
		admins[1].Identity().String(), admins[2].Identity().String()))
	gDarc.Rules.AddRule("spawn:darc", gDarc.Rules.GetSignExpr())

	genesisMsg.BlockInterval = time.Second
	_, err = cl.CreateGenesisBlock(genesisMsg)
	if err != nil {
		panic(err)
	}

	// Create a DARC for managers of each hospital
	managersDarcs := []*darc.Darc{}
	for i := 0; i < len(managers); i++ {
		rules := darc.InitRules([]darc.Identity{admins[i].Identity()},
			[]darc.Identity{managers[i].Identity()})
		tempDarc, err := createDarc(cl, gDarc, genesisMsg.BlockInterval, rules,
			"Managers darc", admins...)
		if err != nil {
			panic(err)
		}
		managersDarcs = append(managersDarcs, tempDarc)
	}

	// Create a collective managers DARC
	rules := darc.InitRules([]darc.Identity{admins[0].Identity(), admins[1].Identity(),
		admins[2].Identity()}, []darc.Identity{})
	rules.UpdateSign(expression.InitAndExpr(managersDarcs[0].GetIdentityString(),
		managersDarcs[1].GetIdentityString(), managersDarcs[2].GetIdentityString()))
	rules.AddRule("spawn:darc", rules.GetSignExpr())
	rules.AddRule("spawn:value", rules.GetSignExpr())
	rules.AddRule("spawn:UserProjectsMap", expression.InitOrExpr(managersDarcs[0].GetIdentityString(),
		managersDarcs[1].GetIdentityString(), managersDarcs[2].GetIdentityString()))
	rules.AddRule("invoke:update", rules["spawn:UserProjectsMap"])
	allManagersDarc, err := createDarc(cl, gDarc, genesisMsg.BlockInterval, rules,
		"AllManagers darc", admins...)
	if err != nil {
		panic(err)
	}
	// Create a DARC for users of each hospital
	usersDarcs := []*darc.Darc{}
	for i := 0; i < len(users); i++ {
		rules := darc.InitRules([]darc.Identity{darc.NewIdentityDarc(managersDarcs[i].GetID())},
			[]darc.Identity{users[i]})
		tempDarc, err := createDarc(cl, allManagersDarc, genesisMsg.BlockInterval, rules,
			"Users darc", managers...)
		if err != nil {
			panic(err)
		}
		usersDarcs = append(usersDarcs, tempDarc)
	}

	// Create a collective users DARC
	rules = darc.InitRules([]darc.Identity{darc.NewIdentityDarc(allManagersDarc.GetID())},
		[]darc.Identity{darc.NewIdentityDarc(usersDarcs[0].GetID()), darc.NewIdentityDarc(usersDarcs[1].GetID()),
			darc.NewIdentityDarc(usersDarcs[2].GetID())})
	rules.AddRule("spawn:ProjectList", rules.GetSignExpr())
	allUsersDarc, err = createDarc(cl, allManagersDarc, genesisMsg.BlockInterval, rules,
		"AllUsers darc", managers...)
	if err != nil {
		panic(err)
	}

	// Create a sample project DARC
	projectXDarcRules := darc.InitRules([]darc.Identity{darc.NewIdentityDarc(managersDarcs[0].GetID()),
		darc.NewIdentityDarc(managersDarcs[2].GetID())}, []darc.Identity{darc.NewIdentityDarc(usersDarcs[0].GetID()),
		darc.NewIdentityDarc(usersDarcs[2].GetID())})
	// Define access control rules for the project DARC
	projectXDarcRules.AddRule("spawn:AuthGrant", projectXDarcRules.GetSignExpr())
	projectXDarcRules.AddRule("spawn:CreateQuery", projectXDarcRules.GetSignExpr())
	projectXDarcRules.AddRule(darc.Action("spawn:"+contracts.QueryTypes[0]), projectXDarcRules.GetSignExpr())
	projectXDarcRules.AddRule(darc.Action("spawn:"+contracts.QueryTypes[1]), expression.InitOrExpr(usersDarcs[0].GetIdentityString()))
	projectXDarc, err := createDarc(cl, allManagersDarc, genesisMsg.BlockInterval, projectXDarcRules,
		"ProjectX", managers...)
	if err != nil {
		panic(err)
	}

	// Register the sample project DARC with the value contract
	myvalue := []byte(projectXDarc.GetIdentityString())
	ctx := service.ClientTransaction{
		Instructions: []service.Instruction{{
			InstanceID: service.InstanceID{
				DarcID: allManagersDarc.GetBaseID(),
				SubID:  service.SubID{},
			},
			Nonce:  service.Nonce{},
			Index:  0,
			Length: 1,
			Spawn: &service.Spawn{
				ContractID: contracts.ContractValueID,
				Args: []service.Argument{{
					Name:  "value",
					Value: myvalue,
				}},
			},
		}},
	}
	err = ctx.Instructions[0].SignBy(managers[0], managers[1], managers[2])
	if err != nil {
		panic(err)
	}

	_, err = cl.AddTransaction(ctx)
	if err != nil {
		panic(err)
	}

	allProjectsListInstanceID := service.InstanceID{
		DarcID: ctx.Instructions[0].InstanceID.DarcID,
		SubID:  service.NewSubID(ctx.Instructions[0].Hash()),
	}

	pr, err := cl.WaitProof(allProjectsListInstanceID, genesisMsg.BlockInterval, nil)
	if pr.InclusionProof.Match() != true {
		panic(err)
	}

	// Create a users-projects map contract instance
	usersByte := []byte(users[2].String() + ";" + users[0].String())
	ctx = service.ClientTransaction{
		Instructions: []service.Instruction{{
			InstanceID: service.InstanceID{
				DarcID: allManagersDarc.GetBaseID(),
				SubID:  service.SubID{},
			},
			Nonce:  service.Nonce{},
			Index:  0,
			Length: 1,
			Spawn: &service.Spawn{
				ContractID: contracts.ContractUserProjectsMapID,
				Args: []service.Argument{{
					Name:  "allProjectsListInstanceID",
					Value: []byte(allProjectsListInstanceID.Slice()),
				}, {
					Name:  "users",
					Value: usersByte,
				}},
			},
		}},
	}
	err = ctx.Instructions[0].SignBy(managers[0], managers[2])
	if err != nil {
		panic(err)
	}
	_, err = cl.AddTransaction(ctx)
	if err != nil {
		panic(err)
	}
	userProjectsMapInstanceID = service.InstanceID{
		DarcID: ctx.Instructions[0].InstanceID.DarcID,
		SubID:  service.NewSubID(ctx.Instructions[0].Hash()),
	}
	pr, err = cl.WaitProof(userProjectsMapInstanceID, genesisMsg.BlockInterval, nil)
	if pr.InclusionProof.Match() != true {
		panic(err)
	}
}

func createDarc(client *service.Client, baseDarc *darc.Darc, interval time.Duration, rules darc.Rules, description string, signers ...darc.Signer) (*darc.Darc, error) {
	// Create a transaction to spawn a DARC
	tempDarc := darc.NewDarc(rules, []byte(description))
	tempDarcBuff, err := tempDarc.ToProto()
	if err != nil {
		return nil, err
	}
	ctx := service.ClientTransaction{
		Instructions: []service.Instruction{{
			InstanceID: service.InstanceID{
				DarcID: baseDarc.GetBaseID(),
				SubID:  service.SubID{},
			},
			Nonce:  service.Nonce{},
			Index:  0,
			Length: 1,
			Spawn: &service.Spawn{
				ContractID: service.ContractDarcID,
				Args: []service.Argument{{
					Name:  "darc",
					Value: tempDarcBuff,
				}},
			},
		}},
	}
	if err := ctx.Instructions[0].SignBy(signers...); err != nil {
		return nil, err
	}

	// Commit transaction
	if _, err := client.AddTransaction(ctx); err != nil {
		return nil, err
	}

	// Verify DARC creation before returning its reference
	instID := service.InstanceID{
		DarcID: tempDarc.GetBaseID(),
		SubID:  service.SubID{},
	}
	pr, err := client.WaitProof(instID, interval, nil)
	if err != nil || pr.InclusionProof.Match() == false {
		return nil, err
	}

	return tempDarc, nil
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
	instID := service.InstanceID{
		DarcID: (*testTransactionRetrieved).Instructions[0].InstanceID.DarcID,
		SubID:  service.NewSubID((*testTransactionRetrieved).Instructions[0].Hash()),
	}

	// Respond if the transaction succeeded
	w.Header().Set("Content-Type", "text/plain")
	pr, err := cl.WaitProof(instID, genesisMsg.BlockInterval, nil)
	if err != nil || pr.InclusionProof.Match() != true {
		w.Write([]byte("Failed to commit the transaction to the MedChain"))
	} else {
		w.Write([]byte(b64.StdEncoding.EncodeToString(instID.Slice())))
	}
}

type Message struct {
	Token   string  `json:"token"`
}

func doMedChainValidation(msg Message) (bool, string) {
	incomingTokenValue := msg.Token
	instIDbytes, err := b64.StdEncoding.DecodeString(incomingTokenValue)
	if err == nil && incomingTokenValue != "" {
		instID := service.NewInstanceID(instIDbytes)
		pr, err := cl.WaitProof(instID, genesisMsg.BlockInterval, nil)
		if err == nil && pr.InclusionProof.Match() == true {
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
			"all_users_darc": b64.StdEncoding.EncodeToString(allUsersDarc.GetBaseID()),
			"user_projects_maps" : b64.StdEncoding.EncodeToString(userProjectsMapInstanceID.Slice()),
			"error":"",
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
	http.HandleFunc("/applyTransaction", applyTransaction)
	http.HandleFunc("/tokenIntrospectionLogin", tokenIntrospectionLogin)
	http.HandleFunc("/tokenIntrospectionQuery", tokenIntrospectionQuery)
	if err := http.ListenAndServe(":8989", nil); err != nil {
		panic(err)
	}

	// Wrap Omniledger service
	services := local.GetServices(servers, service.OmniledgerID)
	for _, s := range services {
		s.(*service.Service).ClosePolling()
	}
	local.WaitDone(genesisMsg.BlockInterval)
	local.CloseAll()
}