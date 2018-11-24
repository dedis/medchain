package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/talhaparacha/medChain/medChainUtils"
)

func GetAdminInfo(w http.ResponseWriter, r *http.Request) {
	list_of_maps := []map[string]string{managersListLevel0DarcsMap, usersListLevel1DarcsMap}
	getInfo(w, r, baseIdToDarcMap, adminsDarcsMap, list_of_maps)
}

func NewManagerMetadata(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	medChainUtils.Check(err)
	var newDarcs medChainUtils.NewDarcsMetadata
	err = json.Unmarshal(body, &newDarcs)
	medChainUtils.Check(err)
	id := newDarcs.Id
	darcMap := newDarcs.Darcs
	if id == "" || darcMap == nil {
		http.Error(w, "", http.StatusNotFound)
		return
	}
	managerDarc, ok1 := newDarcs.Darcs["manager_darc"]
	usersListDarcLevel0, ok2 := newDarcs.Darcs["user_list_darc"]
	if ok1 && ok2 {
		addDarcToMaps(managerDarc, id, managersDarcsMap)
		addDarcToMaps(usersListDarcLevel0, id, usersListLevel0DarcsMap)
	}
	fmt.Println("add manager ", id)
}
