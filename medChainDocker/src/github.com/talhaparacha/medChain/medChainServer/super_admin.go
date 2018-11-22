package main

import (
	"net/http"
)

func GetSuperAdminInfo(w http.ResponseWriter, r *http.Request) {
	getInfo(w, r, baseIdToDarcMap, superAdminsDarcsMap, darcIdToBaseIdMap, adminsListDarcsMap, true)
}
