package main

import (
	"encoding/base64"
	"encoding/json"
	"net/http"

	"github.com/DPPH/MedChain/medChainServer/messages"
	"github.com/DPPH/MedChain/medChainUtils"
)

/**
This file is used to provide general infomation on the system.
**/

/**
Communicates general information about the system
**/
func info(w http.ResponseWriter, r *http.Request) {

	if systemStart {
		userProjectsMapId := base64.StdEncoding.EncodeToString(metaData.UserProjectsMapInstanceID.Slice())
		reply := messages.GeneralInfoReply{
			SigningServiceUrl:        metaData.SigningServiceUrl,
			GenesisDarcBaseId:        metaData.GenesisDarcBaseId,
			AllSuperAdminsDarcBaseId: metaData.AllSuperAdminsDarcBaseId,
			AllAdminsDarcBaseId:      metaData.AllAdminsDarcBaseId,
			AllManagersDarcBaseId:    metaData.AllManagersDarcBaseId,
			AllUsersDarcBaseId:       metaData.AllUsersDarcBaseId,
			AllUsersDarc:             metaData.AllUsersDarcBaseId,
			UserProjectsMap:          userProjectsMapId,
		}
		json_val, err := json.Marshal(&reply)
		if medChainUtils.CheckError(err, w, r) {
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, err = w.Write(json_val)
		if medChainUtils.CheckError(err, w, r) {
			return
		}
	} else {
		temp := map[string]string{"error": "MedChain not started yet"}
		js, _ := json.Marshal(temp)
		w.Header().Set("Content-Type", "application/json")
		w.Write(js)
	}
}
