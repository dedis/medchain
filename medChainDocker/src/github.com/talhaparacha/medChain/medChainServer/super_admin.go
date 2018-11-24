package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/talhaparacha/medChain/medChainUtils"
)

func GetSuperAdminInfo(w http.ResponseWriter, r *http.Request) {
	list_of_maps := []map[string]string{adminsListDarcsMap, managersListLevel1DarcsMap, usersListLevel2DarcsMap}
	getInfo(w, r, baseIdToDarcMap, superAdminsDarcsMap, list_of_maps)
}

func NewAdminMetadata(w http.ResponseWriter, r *http.Request) {
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
	adminDarc, ok1 := newDarcs.Darcs["admin_darc"]
	usersListDarcLevel1, ok2 := newDarcs.Darcs["user_list_darc"]
	managersListDarcLevel0, ok3 := newDarcs.Darcs["manager_list_darc"]
	if ok1 && ok2 && ok3 {
		addDarcToMaps(adminDarc, id, adminsDarcsMap)
		addDarcToMaps(usersListDarcLevel1, id, usersListLevel1DarcsMap)
		addDarcToMaps(managersListDarcLevel0, id, managersListLevel0DarcsMap)
	}
	fmt.Println("add admin ", id)
}
