package main

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"sort"
	"strings"

	"github.com/dedis/cothority/omniledger/darc"
	"github.com/dedis/cothority/omniledger/darc/expression"
	"github.com/dedis/cothority/omniledger/service"
	"github.com/talhaparacha/medChain/medChainServer/messages"
	"github.com/talhaparacha/medChain/medChainServer/metadata"
	"github.com/talhaparacha/medChain/medChainUtils"
)

func AddProject(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if medChainUtils.CheckError(err, w, r) {
		return
	}
	var request messages.AddProjectRequest
	err = json.Unmarshal(body, &request)
	if medChainUtils.CheckError(err, w, r) {
		return
	}

	identity, transaction, signers, digests, err := prepareNewProject(&request)
	if medChainUtils.CheckError(err, w, r) {
		return
	}
	reply := messages.ActionReply{Initiator: request.Initiator, ActionType: "add new project", Ids: []string{identity}, Transaction: transaction, Signers: signers, InstructionDigests: digests}
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

func prepareNewProject(request *messages.AddProjectRequest) (string, string, map[string]int, map[int][]byte, error) {

	initiator_metadata, ok := metaData.GenericUsers[request.Initiator]
	if !ok {
		return "", "", nil, nil, errors.New("Could not find the initiator metadata")
	}
	if !initiator_metadata.IsCreated {
		return "", "", nil, nil, errors.New("The initiator was not approved")
	}
	if initiator_metadata.Role != "admin" {
		return "", "", nil, nil, errors.New("You need to be an Admin to add a project")
	}

	manager_metadata_list, err := loadProjectManagers(request)
	if err != nil {
		return "", "", nil, nil, err
	}

	spawner_darc, signers_ids, signers, err := getSpawnProjectDarcSigners(initiator_metadata, manager_metadata_list)
	if err != nil {
		return "", "", nil, nil, err
	}

	user_metadata_list, query_mapping, err := loadProjectUsers(request)

	project_darc, project_metadata, err := createProjectDarc(request.Name, manager_metadata_list, user_metadata_list, query_mapping)
	if err != nil {
		return "", "", nil, nil, err
	}

	project_bytes, err := getUpdatedProjectListBytes(project_darc)

	user_bytes := getUpdatedUserMapBytes(user_metadata_list)

	project_creator_darc, ok := metaData.BaseIdToDarcMap[metaData.ProjectCreatorDarcBaseId]
	if !ok {
		return "", "", nil, nil, errors.New("Could not load the project creator darc")
	}

	transaction, err := createTransactionForNewProject(spawner_darc, project_creator_darc, project_darc, project_bytes, user_bytes)
	base_darcs := []*darc.Darc{spawner_darc, project_creator_darc, project_creator_darc}
	digests, err := computeTransactionDigests(transaction, signers_ids, base_darcs)
	if err != nil {
		return "", "", nil, nil, err
	}

	addProjectToMetadata(project_metadata, project_darc)

	transaction_string, err := transactionToString(transaction)
	if err != nil {
		return "", "", nil, nil, err
	}

	return project_metadata.Id, transaction_string, signers, digests, nil
}

func loadProjectManagers(request *messages.AddProjectRequest) ([]*metadata.GenericUser, error) {
	result := []*metadata.GenericUser{}
	for _, manager_id := range request.Managers {
		manager_meta, ok := metaData.GenericUsers[manager_id]
		if !ok {
			return nil, errors.New("Could not find the manager's data")
		}
		if manager_meta.Role != "manager" {
			return nil, errors.New("Given id " + manager_id + " is not a manager")
		}
		if !manager_meta.IsCreated {
			return nil, errors.New("Given manager was not approved")
		}
		result = append(result, manager_meta)
	}
	return result, nil
}

func loadProjectUsers(request *messages.AddProjectRequest) ([]*metadata.GenericUser, map[string][]*metadata.GenericUser, error) {
	tmp_map := make(map[string]*metadata.GenericUser)
	query_mapping := make(map[string][]*metadata.GenericUser)
	for query_type, user_ids := range request.Queries {
		query_mapping[query_type] = []*metadata.GenericUser{}
		for _, user_id := range user_ids {
			user_meta, ok := metaData.GenericUsers[user_id]
			if !ok {
				return nil, nil, errors.New("Could not find the user's data")
			}
			if user_meta.Role != "user" {
				return nil, nil, errors.New("Given id is not a user")
			}
			if !user_meta.IsCreated {
				return nil, nil, errors.New("Given user was not approved")
			}
			query_mapping[query_type] = append(query_mapping[query_type], user_meta)
			tmp_map[user_meta.Id.String()] = user_meta
		}
	}
	user_metadata_list := []*metadata.GenericUser{}
	for _, user_metadata := range tmp_map {
		user_metadata_list = append(user_metadata_list, user_metadata)
	}
	return user_metadata_list, query_mapping, nil
}

func createProjectDarc(name string, manager_metadata_list, user_metadata_list []*metadata.GenericUser, query_mapping map[string][]*metadata.GenericUser) (*darc.Darc, *metadata.Project, error) {

	darc_managers := []string{}
	project_metadata, err := metadata.NewProject(name)
	if err != nil {
		return nil, nil, err
	}

	project_hospitals := make(map[string]*metadata.Hospital)
	for _, manager_metadata := range manager_metadata_list {
		manager_darc, ok := metaData.BaseIdToDarcMap[manager_metadata.DarcBaseId]
		if !ok {
			return nil, nil, errors.New("Could not load manager darc")
		}
		id := manager_darc.GetIdentityString()
		darc_managers = append(darc_managers, id)
		project_metadata.Managers = append(project_metadata.Managers, manager_metadata)
		project_hospitals[manager_metadata.Hospital.SuperAdmin.Id.String()] = manager_metadata.Hospital
	}

	admin_list_darcs := []string{}
	for _, hospital_metadata := range project_hospitals {
		admin_list_darc, ok := metaData.BaseIdToDarcMap[hospital_metadata.AdminListDarcBaseId]
		if !ok {
			return nil, nil, errors.New("Could not load admin list darc")
		}
		admin_list_darcs = append(admin_list_darcs, admin_list_darc.GetIdentityString())
	}

	darc_signers := []string{}
	for _, user_metadata := range user_metadata_list {
		user_darc, ok := metaData.BaseIdToDarcMap[user_metadata.DarcBaseId]
		if !ok {
			return nil, nil, errors.New("Could not load user darc")
		}
		id := user_darc.GetIdentityString()
		darc_signers = append(darc_signers, id)
		project_metadata.Users = append(project_metadata.Users, user_metadata)
	}

	projectDarcRules := darc.InitRulesWith([]darc.Identity{}, []darc.Identity{}, "invoke:evolve")
	projectDarcRules.UpdateRule("invoke:evolve", expression.InitOrExpr(string(medChainUtils.InitAtLeastTwoExpr(darc_managers)), string(expression.InitOrExpr(admin_list_darcs...))))
	projectDarcRules.UpdateSign(expression.InitOrExpr(darc_signers...))
	projectDarcRules.AddRule("spawn:AuthGrant", projectDarcRules.GetSignExpr())
	projectDarcRules.AddRule("spawn:CreateQuery", projectDarcRules.GetSignExpr())

	for query_type, users := range query_mapping {
		usersIdString := []string{}
		project_metadata.Queries[query_type] = []*metadata.GenericUser{}
		for _, user_metadata := range users {
			user_darc, ok := metaData.BaseIdToDarcMap[user_metadata.DarcBaseId]
			if !ok {
				return nil, nil, errors.New("Could not load user darc")
			}
			idString := user_darc.GetIdentityString()
			usersIdString = append(usersIdString, idString)
			project_metadata.Queries[query_type] = append(project_metadata.Queries[query_type], user_metadata)
		}
		expr := expression.InitOrExpr(usersIdString...)
		projectDarcRules.AddRule(darc.Action(query_type), expr)
	}

	projectDarc := darc.NewDarc(projectDarcRules, []byte("Project darc for project "+name))
	return projectDarc, project_metadata, nil
}

func getUpdatedUserMapBytes(user_metadata_list []*metadata.GenericUser) []byte {
	user_ids := []string{}
	for _, user_metadata := range user_metadata_list {
		user_ids = append(user_ids, user_metadata.Id.String())
	}
	return []byte(strings.Join(user_ids, ";"))
}

func getUpdatedProjectListBytes(new_project_darc *darc.Darc) ([]byte, error) {
	list_of_projects := []string{}
	new_id := new_project_darc.GetIdentityString()
	for _, project := range metaData.Projects {
		if project.IsCreated {
			projectDarc, ok := metaData.BaseIdToDarcMap[project.DarcBaseId]
			if !ok {
				return nil, errors.New("Could not load the darc of an exisiting project")
			}
			list_of_projects = append(list_of_projects, projectDarc.GetIdentityString())
			if projectDarc.GetIdentityString() == new_id {
				return nil, errors.New("The project is already existing")
			}
		}
	}
	list_of_projects = append(list_of_projects, new_id)
	sort.Strings(list_of_projects)
	return []byte(strings.Join(list_of_projects, ";")), nil
}

func getSpawnProjectDarcSigners(initiator_metadata *metadata.GenericUser, manager_metadata_list []*metadata.GenericUser) (*darc.Darc, map[int]darc.Identity, map[string]int, error) {

	if len(manager_metadata_list) < 1 {
		return nil, nil, nil, errors.New("You need to give at least one manager for the project")
	}

	hospital_metadata := manager_metadata_list[0].Hospital
	for _, manager_meta := range manager_metadata_list {
		if manager_meta.Hospital.SuperAdmin.Id.String() == initiator_metadata.Hospital.SuperAdmin.Id.String() {
			hospital_metadata = manager_meta.Hospital
		}
	}
	return getSigners(hospital_metadata, "User", []string{initiator_metadata.Id.String()})
}

func createTransactionForNewProject(spawner_darc, project_creator_darc, project_darc *darc.Darc, project_bytes, users_bytes []byte) (*service.ClientTransaction, error) {

	project_darc_buff, err := project_darc.ToProto()
	if err != nil {
		return nil, err
	}
	ctx := service.ClientTransaction{
		Instructions: []service.Instruction{
			service.Instruction{
				InstanceID: service.NewInstanceID(spawner_darc.GetBaseID()),
				Nonce:      service.Nonce{},
				Index:      0,
				Length:     3,
				Spawn: &service.Spawn{
					ContractID: service.ContractDarcID,
					Args: []service.Argument{{
						Name:  "darc",
						Value: project_darc_buff,
					}},
				},
			},
			service.Instruction{
				InstanceID: metaData.AllProjectsListInstanceID,
				Nonce:      service.Nonce{},
				Index:      1,
				Length:     3,
				Invoke: &service.Invoke{
					Command: "update",
					Args: []service.Argument{{
						Name:  "value",
						Value: project_bytes,
					}},
				},
			},
			service.Instruction{
				InstanceID: metaData.UserProjectsMapInstanceID,
				Nonce:      service.Nonce{},
				Index:      2,
				Length:     3,
				Invoke: &service.Invoke{
					Command: "update",
					Args: []service.Argument{{
						Name:  "allProjectsListInstanceID",
						Value: []byte(metaData.AllProjectsListInstanceID.Slice()),
					}, {
						Name:  "users",
						Value: users_bytes,
					}},
				},
			},
		},
	}
	return &ctx, nil
}

func addProjectToMetadata(project_metadata *metadata.Project, project_darc *darc.Darc) {
	for _, user_metadata := range project_metadata.Users {
		user_metadata.Projects[project_metadata.Id] = project_metadata
	}
	for _, manager_metadata := range project_metadata.Managers {
		manager_metadata.Projects[project_metadata.Id] = project_metadata
		hospital_metadata := manager_metadata.Hospital
		hospital_metadata.SuperAdmin.Projects[project_metadata.Id] = project_metadata
		for _, admin_metadata := range hospital_metadata.Admins {
			admin_metadata.Projects[project_metadata.Id] = project_metadata
		}
	}
	metaData.Projects[project_metadata.Id] = project_metadata
	base_id := medChainUtils.IDToB64String(project_darc.GetBaseID())
	metaData.ProjectsWaitingForCreation[base_id] = project_metadata
}
