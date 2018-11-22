package main

import (
	"net/http"

	"github.com/dedis/cothority/omniledger/darc"
	"github.com/dedis/cothority/omniledger/service"
)

type NewUserInfo struct {
	UserPublicKey    string `json:"user_public_key"`
	ManagerPublicKey string `json:"manager_public_key"`
}

type NewUserTransaction struct {
	UserPublicKey    string                    `json:"user_public_key"`
	ManagerPublicKey string                    `json:"manager_public_key"`
	Transaction      service.ClientTransaction `json:"transaction"`
	Darc             darc.Darc                 `json:"darc"`
}

func GetManagerInfo(w http.ResponseWriter, r *http.Request) {
	getInfo(w, r, baseIdToDarcMap, managersDarcsMap, darcIdToBaseIdMap, usersListDarcsMap, true)
}

func GetUserInfo(w http.ResponseWriter, r *http.Request) {
	getInfo(w, r, baseIdToDarcMap, usersDarcsMap, darcIdToBaseIdMap, nil, false)
}
