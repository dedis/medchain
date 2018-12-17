package main

import (
	"fmt"
	"net/http"
)

func GetAdminInfo(w http.ResponseWriter, r *http.Request) {
	fmt.Println("/info/admin")
	getGenericUserInfo(w, r, metaData.Admins)
}

// func NewManagerMetadata(w http.ResponseWriter, r *http.Request) {
// 	body, err := ioutil.ReadAll(r.Body)
// 	medChainUtils.Check(err)
// 	var newDarcs medChainUtils.NewDarcsMetadata
// 	err = json.Unmarshal(body, &newDarcs)
// 	medChainUtils.Check(err)
// 	id := newDarcs.Id
// 	darcMap := newDarcs.Darcs
// 	if id == "" || darcMap == nil {
// 		http.Error(w, "", http.StatusNotFound)
// 		return
// 	}
// 	managerDarc, ok1 := newDarcs.Darcs["manager_darc"]
// 	if ok1 {
// 		addDarcToMaps(managerDarc, id, managersDarcsMap)
// 	}
// 	fmt.Println("add manager ", id)
// }
