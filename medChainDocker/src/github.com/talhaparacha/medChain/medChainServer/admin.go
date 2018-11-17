package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/talhaparacha/medChain/medChainUtils"
)

func GetAdminInfo(w http.ResponseWriter, r *http.Request) {
	identity := strings.Join(r.URL.Query()["identity"], "")
	fmt.Println("manager info", identity)
	adminDarc := adminsDarcsMap[identity]
	managerListDarc := managersListDarcsMap[identity]
	reply := medChainUtils.AdminInfoReply{AdminDarc: adminDarc, ManagerListDarc: managerListDarc}
	jsonVal, err := json.Marshal(reply)
	if err != nil {
		// TOD0 Return 400 error code
		panic(err)
	}
	w.Write(jsonVal)
}
