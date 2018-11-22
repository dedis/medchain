package main

import (
	"net/http"
)

func GetAdminInfo(w http.ResponseWriter, r *http.Request) {
	getInfo(w, r, baseIdToDarcMap, adminsDarcsMap, darcIdToBaseIdMap, managersListDarcsMap, true)
}
