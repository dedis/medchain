package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/DPPH/MedChain/medChainUtils"
	"github.com/dedis/cothority/omniledger/darc"
)

// These will be initialized after log-ing
var medchainURL string
var irctURL string
var signer darc.Signer

// HTTP Client with appropriate settings for later use
var client *http.Client

// To inject data in HTML templates
type ProjectsPageData struct {
	Username   string
	LoginToken string
	Projects   []Project
}

type LogQueryPageData struct {
	QueryToken string
}

type Project struct {
	Name       string
	ID         string
	QueryTypes []string
}

// Log query page
func logQuery(w http.ResponseWriter, r *http.Request) {
	// Read incoming POST data
	r.ParseForm()
	var Buf1 bytes.Buffer
	file, _, err := r.FormFile("element_1")
	medChainUtils.Check(err)
	io.Copy(&Buf1, file)
	projectDetails := strings.Split(r.FormValue("element_2"), " | ")
	projectDarc := projectDetails[0]
	queryType := projectDetails[1]

	// Create a query log transaction
	transaction := medChainUtils.CreateQueryTransaction(projectDarc, queryType, Buf1.String(), signer)

	// Apply the transaction through MedChain
	request, err := http.NewRequest("GET", medchainURL+"/applyTransaction", nil)
	medChainUtils.Check(err)
	request.Header.Set("transaction", transaction)
	response, err := client.Do(request)
	medChainUtils.Check(err)
	body, err := ioutil.ReadAll(response.Body)
	medChainUtils.Check(err)
	queryToken := string(body[:])
	data := LogQueryPageData{
		QueryToken: queryToken,
	}

	// Serve the page
	tmpl := template.Must(template.ParseFiles("templates/static/querySuccess.html"))
	tmpl.Execute(w, data)
}

// Projects page
func projects(w http.ResponseWriter, r *http.Request) {
	// Read incoming POST data
	medchainURL = r.FormValue("element_1")
	irctURL = r.FormValue("element_2")
	var Buf1 bytes.Buffer
	file, _, err := r.FormFile("element_3")
	medChainUtils.Check(err)
	io.Copy(&Buf1, file)
	var Buf2 bytes.Buffer
	file, _, err = r.FormFile("element_4")
	medChainUtils.Check(err)
	io.Copy(&Buf2, file)

	// Get information, necessary for a log-in transaction, from MedChain
	response, err := http.Get(medchainURL + "/info")
	medChainUtils.Check(err)
	body, err := ioutil.ReadAll(response.Body)
	medChainUtils.Check(err)
	var dat map[string]interface{}
	err = json.Unmarshal(body, &dat)
	medChainUtils.Check(err)
	allUsersDarcID := dat["all_users_darc"].(string)
	userProjectsMapID := dat["user_projects_maps"].(string)

	// Create a log-in transaction
	signer = medChainUtils.LoadSignerEd25519FromBytes(Buf1.Bytes(), Buf2.Bytes())
	transaction := medChainUtils.CreateLoginTransaction(allUsersDarcID, userProjectsMapID, signer)

	// Apply the transaction through MedChain
	request, err := http.NewRequest("GET", medchainURL+"/applyTransaction", nil)
	medChainUtils.Check(err)
	request.Header.Set("transaction", transaction)
	response, err = client.Do(request)
	medChainUtils.Check(err)
	body, err = ioutil.ReadAll(response.Body)
	medChainUtils.Check(err)
	loginToken := string(body[:])
	fmt.Println("token", loginToken)

	// Get the projects list from IRCT
	request, err = http.NewRequest("GET", irctURL+"/systemService/about", nil)
	medChainUtils.Check(err)
	request.Header.Set("Authorization", "Bearer "+loginToken)
	response, err = client.Do(request)
	fmt.Println("code", response.StatusCode)
	medChainUtils.Check(err)
	body, err = ioutil.ReadAll(response.Body)
	var dat2 []map[string]interface{}
	fmt.Println("body", body)
	err = json.Unmarshal(body, &dat2)
	fmt.Println("err", err != nil)
	fmt.Println("dat2", dat2)
	projects := []Project{}
	username := dat2[1]["username"].(string)
	tmp := strings.Split(dat2[1]["projects"].(string), "......")
	// Loop through each project
	for _, value := range tmp {
		tmp := strings.Split(value, "...")
		projectName := tmp[0]
		projectID := tmp[1]
		projectQueryTypes := []string{}

		// Loop through each query type
		for i := 2; i < len(tmp); i++ {
			projectQueryTypes = append(projectQueryTypes, tmp[i])
		}

		// Add to main slice
		projects = append(projects, Project{
			Name:       projectName,
			ID:         projectID,
			QueryTypes: projectQueryTypes,
		})
	}

	// Serve the page
	data := ProjectsPageData{
		Username:   username,
		LoginToken: loginToken,
		Projects:   projects,
	}
	tmpl := template.Must(template.ParseFiles("templates/static/projects.html"))
	tmpl.Execute(w, data)
}

// Landing page
func landing(w http.ResponseWriter, r *http.Request) {
	tmpl := template.Must(template.ParseFiles("templates/static/landing.html"))
	tmpl.Execute(w, nil)
}

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
