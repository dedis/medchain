package main

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"

	"github.com/DPPH/MedChain/medChainServer/messages"
	"github.com/DPPH/MedChain/medChainServer/metadata"
	"github.com/DPPH/MedChain/medChainUtils"
)

/**
Translates the project metadata to a messages.ProjectInfoReply object
**/
func projectMetadataToInfoReply(project_metadata *metadata.Project) messages.ProjectInfoReply {
	users := []messages.GenericUserInfoReply{}
	for _, user_metadata := range project_metadata.Users {
		users = append(users, genericUserMetadataToInfoReplyShort(user_metadata))
	}
	managers := []messages.GenericUserInfoReply{}
	for _, manager_metadata := range project_metadata.Managers {
		managers = append(managers, genericUserMetadataToInfoReplyShort(manager_metadata))
	}
	queries := make(map[string][]messages.GenericUserInfoReply)
	for query_type, user_list := range project_metadata.Queries {
		queries[query_type] = make([]messages.GenericUserInfoReply, 0)
		for _, user_metadata := range user_list {
			queries[query_type] = append(queries[query_type], genericUserMetadataToInfoReply(user_metadata))
		}
	}
	return messages.ProjectInfoReply{Name: project_metadata.Name, DarcBaseId: project_metadata.DarcBaseId, Managers: managers, Users: users, Queries: queries, IsCreated: project_metadata.IsCreated}
}

func projectMetadataToInfoReplyShort(project_metadata *metadata.Project) messages.ProjectInfoReply {
	return messages.ProjectInfoReply{Name: project_metadata.Name, IsCreated: project_metadata.IsCreated}
}

/**
List all projects attached to a given user
It should receive a messages.ListProjectRequest in json in the request body
It returns a messages.ListProjectReply in the body of the response
**/
func ListProjects(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if medChainUtils.CheckError(err, w, r) {
		return
	}
	var request messages.ListProjectRequest
	err = json.Unmarshal(body, &request)
	if medChainUtils.CheckError(err, w, r) {
		return
	}

	var projects map[string]*metadata.Project
	if request.Id == "" {
		projects = metaData.Projects
	} else {
		user_metadata, ok := metaData.GenericUsers[request.Id]
		if !ok {
			medChainUtils.CheckError(errors.New("Could not find the user's metadata"), w, r)
			return
		}
		projects = user_metadata.Projects
	}

	project_replies := []messages.ProjectInfoReply{}
	for _, project_metadata := range projects {
		project_replies = append(project_replies, projectMetadataToInfoReplyShort(project_metadata))
	}

	reply := messages.ListProjectReply{Projects: project_replies}
	json_val, err := json.Marshal(&reply)
	if medChainUtils.CheckError(err, w, r) {
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(json_val)
	if medChainUtils.CheckError(err, w, r) {
		return
	}
}

/**
Get the information on a particular project
It should receive a messages.ProjectInfoRequest in json in the request body
It returns a messages.ProjectInfoReply in the body of the response
**/
func GetProjectInfo(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if medChainUtils.CheckError(err, w, r) {
		return
	}
	var request messages.ProjectInfoRequest
	err = json.Unmarshal(body, &request)
	if medChainUtils.CheckError(err, w, r) {
		return
	}

	project_metadata, ok := metaData.Projects[request.Name]
	if !ok {
		medChainUtils.CheckError(errors.New("Could not find the project's metadata"), w, r)
		return
	}

	reply := projectMetadataToInfoReply(project_metadata)
	json_val, err := json.Marshal(&reply)
	if medChainUtils.CheckError(err, w, r) {
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(json_val)
	if medChainUtils.CheckError(err, w, r) {
		return
	}
}
