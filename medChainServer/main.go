package main

import (
	"net/http"
	"net/http/httputil"

	"github.com/DPPH/MedChain/medChainServer/metadata"
	"github.com/dedis/cothority/omniledger/service"
	"github.com/dedis/onet"
)

// Configure cothority here...
var local *onet.LocalTest
var roster *onet.Roster
var cl *service.Client

// used by the bootstraping
var configFileName string

// common metadata object
var metaData *metadata.Metadata

// used to make the server as a proxy for the signing service
// To avoid CORS problems with the gui
var signingProxy *httputil.ReverseProxy

var err error

// to know if the system was already bootstraped
var systemStart = false

// bootstraps the system only once
func start(w http.ResponseWriter, r *http.Request) {
	if !systemStart {
		startSystem(metaData, configFileName)
		systemStart = true
	}
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte("Success"))
}

// func main() {
//
// 	// Run one time to generate all the keys for our context
// 	// medChainUtils.InitKeys(6, "keys/super_admins")
// 	// medChainUtils.InitKeys(6, "keys/admins")
// 	// medChainUtils.InitKeys(6, "keys/managers")
// 	// medChainUtils.InitKeys(6, "keys/users")
//
// 	// get the flags
// 	port, conf, signing_url := getFlags()
//
// 	// set up a simulation of several nodes, using onet
// 	local = onet.NewTCPTest(cothority.Suite)
// 	_, roster, _ = local.GenTree(3, true)
//
// 	// omniledger client
// 	cl = service.NewClient()
//
// 	configFileName = conf
//
// 	// initialize the common metadata object
// 	metaData = metadata.NewMetadata()
// 	metaData.SigningServiceUrl = signing_url
//
// 	// set up the medchain server as a proxy for the signing service
// 	proxy_url, err := url.Parse(signing_url)
// 	if err != nil {
// 		panic(err)
// 	}
// 	signingProxy = httputil.NewSingleHostReverseProxy(proxy_url)
//
// 	// For the graphical user interface
// 	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("admin_gui/templates/static"))))
// 	http.HandleFunc("/gui", admin_gui.GUI_landing)
//
// 	// For the bootstrapping
// 	http.HandleFunc("/start", start)
//
// 	// Api to get information on a particular element
// 	http.HandleFunc("/info", info)
// 	http.HandleFunc("/info/hospital", GetHospitalInfo)
// 	http.HandleFunc("/info/user", GetGenericUserInfo)
// 	http.HandleFunc("/info/project", GetProjectInfo)
// 	http.HandleFunc("/info/darc", GetDarcInfo)
//
// 	// Api to list elements
// 	http.HandleFunc("/list/darc", ListDarcUsers)
// 	http.HandleFunc("/list/users", ListGenericUser)
// 	http.HandleFunc("/list/hospitals", ListHospitals)
// 	http.HandleFunc("/list/projects", ListProjects)
//
// 	// Api to add new elements
// 	http.HandleFunc("/add/user", AddUser)
// 	http.HandleFunc("/add/manager", AddManager)
// 	http.HandleFunc("/add/admin", AddAdmin)
// 	http.HandleFunc("/add/hospital", AddHospital)
// 	http.HandleFunc("/add/project", AddProject)
//
// 	// Api to commit or cancel new elements
// 	http.HandleFunc("/commit/action", CommitAction)
// 	http.HandleFunc("/cancel/action", CancelAction)
//
// 	// Forwards it to the signing service, as a proxy
// 	http.HandleFunc("/add/action", forwardToSigning)
// 	http.HandleFunc("/info/action", forwardToSigning)
// 	http.HandleFunc("/list/actions", forwardToSigning)
// 	http.HandleFunc("/list/actions/waiting", forwardToSigning)
// 	http.HandleFunc("/approve/action", forwardToSigning)
// 	http.HandleFunc("/deny/action", forwardToSigning)
// 	http.HandleFunc("/update/action/done", forwardToSigning)
// 	http.HandleFunc("/update/action/cancel", forwardToSigning)
//
// 	// Api to login and register queries, get token
// 	// And api to verify tokens
// 	http.HandleFunc("/applyTransaction", applyTransaction)
// 	http.HandleFunc("/tokenIntrospectionLogin", tokenIntrospectionLogin)
// 	http.HandleFunc("/tokenIntrospectionQuery", tokenIntrospectionQuery)
//
// 	if err := http.ListenAndServe(":"+port, nil); err != nil {
// 		panic(err)
// 	}
// 	// Wrap Omniledger service
// 	local.WaitDone(metaData.GenesisMsg.BlockInterval)
// 	local.CloseAll()
// }
