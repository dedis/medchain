package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/DPPH/MedChain/medChainServer/conf"
	"github.com/DPPH/MedChain/medChainServer/metadata"
	"github.com/DPPH/MedChain/medChainUtils"
	"github.com/dedis/cothority/omniledger/contracts"
	"github.com/dedis/cothority/omniledger/darc"
	"github.com/dedis/cothority/omniledger/darc/expression"
	"github.com/dedis/cothority/omniledger/service"
)

/**
This filetakes care of the bootstrapping process for the project darcs
**/

/**
This function creates the darc that is used to spawn and update the project list value and the user-project map
**/
func createProjectCreatorDarc(metaData *metadata.Metadata, signers []darc.Signer) {
	genesisDarc, ok := metaData.BaseIdToDarcMap[metaData.GenesisDarcBaseId]
	if !ok {
		panic(errors.New("Could not load the genesisDarc"))
	}

	all_super_admin_darc, ok := metaData.BaseIdToDarcMap[metaData.AllSuperAdminsDarcBaseId]
	if !ok {
		panic(errors.New("Could not load the all super admins darc"))
	}

	all_admin_darc, ok := metaData.BaseIdToDarcMap[metaData.AllAdminsDarcBaseId]
	if !ok {
		panic(errors.New("Could not load the all admins darc"))
	}

	all_manager_darc, ok := metaData.BaseIdToDarcMap[metaData.AllManagersDarcBaseId]
	if !ok {
		panic(errors.New("Could not load the all managers darc"))
	}

	owners := []darc.Identity{darc.NewIdentityDarc(genesisDarc.GetID())}
	rules := darc.InitRulesWith(owners, []darc.Identity{}, "invoke:evolve")
	rules.UpdateSign(expression.InitOrExpr(all_super_admin_darc.GetIdentityString(), all_admin_darc.GetIdentityString(), all_manager_darc.GetIdentityString()))
	rules.AddRule("spawn:value", expression.InitOrExpr(all_super_admin_darc.GetIdentityString()))
	rules.AddRule("spawn:UserProjectsMap", expression.InitOrExpr(all_super_admin_darc.GetIdentityString()))
	rules.AddRule("invoke:update", expression.InitOrExpr(all_super_admin_darc.GetIdentityString(), all_admin_darc.GetIdentityString(), all_manager_darc.GetIdentityString()))

	projectCreatorDarc, err := medChainUtils.CreateDarc(cl, genesisDarc, metaData.GenesisMsg.BlockInterval, rules, "Project Creator Darc", signers...)
	if err != nil {
		panic(err)
	}
	metaData.ProjectCreatorDarcBaseId = addDarcToMaps(projectCreatorDarc, metaData)
}

/**
This is a helper function to find the corresponding metadata given
a set of coordinates from the configuration file.
**/
func findUserMetadata(configuration *conf.Configuration, metaData *metadata.Metadata, userCoordinates conf.Coordinates) *metadata.GenericUser {
	hospital_conf := configuration.Hospitals[userCoordinates.I]
	user_conf := hospital_conf.Users[userCoordinates.J]
	user_identity := medChainUtils.LoadIdentityEd25519(configuration.KeyDirectory + user_conf.PublicKey)
	user_metadata, ok := metaData.GenericUsers[user_identity.String()]
	if !ok {
		panic(errors.New("Could not find user metadata"))
	}
	return user_metadata
}

/**
This is a helper function to find the corresponding metadata
given a set of coordinates from the configuration file.
**/
func findManagerMetadata(configuration *conf.Configuration, metaData *metadata.Metadata, userCoordinates conf.Coordinates) *metadata.GenericUser {
	hospital_conf := configuration.Hospitals[userCoordinates.I]
	manager_conf := hospital_conf.Managers[userCoordinates.J]
	manager_identity := medChainUtils.LoadIdentityEd25519(configuration.KeyDirectory + manager_conf.PublicKey)
	manager_metadata, ok := metaData.GenericUsers[manager_identity.String()]
	if !ok {
		panic(errors.New("Could not find manager metadata"))
	}
	return manager_metadata
}

/**
This function creates the project darcs
following the information from the configuration file.
It also spawns the project list value and the user-project map
**/
func createProjectDarcs(configuration *conf.Configuration, metaData *metadata.Metadata, signers []darc.Signer) {

	genesisDarc, ok := metaData.BaseIdToDarcMap[metaData.GenesisDarcBaseId]
	if !ok {
		panic(errors.New("Could not load the genesisDarc"))
	}

	list_of_projects := []string{}

	user_ids := []string{}

	for _, project := range configuration.Projects {

		project_metadata, err := metadata.NewProject(project.Name)

		if err != nil {
			panic(err)
		}

		darc_managers := []string{}
		project_hospitals := make(map[string]*metadata.Hospital)
		for _, managerCoordinates := range project.ManagerOwners {

			manager_metadata := findManagerMetadata(configuration, metaData, managerCoordinates)
			manager_darc, ok := metaData.BaseIdToDarcMap[manager_metadata.DarcBaseId]
			if !ok {
				panic(errors.New("Could not load manager darc"))
			}
			id := manager_darc.GetIdentityString()
			darc_managers = append(darc_managers, id)
			project_metadata.Managers = append(project_metadata.Managers, manager_metadata)
			manager_metadata.Projects[project_metadata.Name] = project_metadata
			project_hospitals[manager_metadata.Hospital.SuperAdmin.Id.String()] = manager_metadata.Hospital
		}

		admin_list_darcs := []string{}
		for _, hospital_metadata := range project_hospitals {
			admin_list_darc, ok := metaData.BaseIdToDarcMap[hospital_metadata.AdminListDarcBaseId]
			if !ok {
				panic(errors.New("Could not load admin list darc"))
			}
			admin_list_darcs = append(admin_list_darcs, admin_list_darc.GetIdentityString())
			for _, admin_metadata := range hospital_metadata.Admins {
				admin_metadata.Projects[project_metadata.Name] = project_metadata
			}
			hospital_metadata.SuperAdmin.Projects[project_metadata.Name] = project_metadata
		}

		darc_signers := []string{}
		for _, userCoordinates := range project.SigningUsers {
			user_metadata := findUserMetadata(configuration, metaData, userCoordinates)
			user_darc, ok := metaData.BaseIdToDarcMap[user_metadata.DarcBaseId]
			if !ok {
				panic(errors.New("Could not load user darc"))
			}
			id := user_darc.GetIdentityString()
			darc_signers = append(darc_signers, id)
			user_ids = append(user_ids, user_metadata.Id.String())
			project_metadata.Users = append(project_metadata.Users, user_metadata)
			user_metadata.Projects[project_metadata.Name] = project_metadata
		}

		projectDarcRules := darc.InitRulesWith([]darc.Identity{}, []darc.Identity{}, "invoke:evolve")
		projectDarcRules.UpdateRule("invoke:evolve", expression.InitOrExpr("("+string(medChainUtils.InitAtLeastTwoExpr(darc_managers))+")", "("+string(medChainUtils.InitAtLeastTwoExpr(admin_list_darcs))+")"))
		projectDarcRules.UpdateSign(expression.InitOrExpr(darc_signers...))
		projectDarcRules.AddRule("spawn:AuthGrant", projectDarcRules.GetSignExpr())
		projectDarcRules.AddRule("spawn:CreateQuery", projectDarcRules.GetSignExpr())

		for _, rule := range project.Rules {
			usersIdString := []string{}
			project_metadata.Queries[rule.Action] = []*metadata.GenericUser{}
			for _, user_index := range rule.Users {
				user_metadata := project_metadata.Users[user_index]
				user_darc, ok := metaData.BaseIdToDarcMap[user_metadata.DarcBaseId]
				if !ok {
					panic(errors.New("Could not load user darc"))
				}
				idString := user_darc.GetIdentityString()
				usersIdString = append(usersIdString, idString)
				project_metadata.Queries[rule.Action] = append(project_metadata.Queries[rule.Action], user_metadata)
			}
			expr := expression.InitOrExpr(usersIdString...)
			projectDarcRules.AddRule(darc.Action("spawn:"+rule.Action), expr)
		}

		projectDarc, err := medChainUtils.CreateDarc(cl, genesisDarc, metaData.GenesisMsg.BlockInterval, projectDarcRules,
			project.Name, signers...)
		if err != nil {
			panic(err)
		}

		project_metadata.DarcBaseId = addDarcToMaps(projectDarc, metaData)
		project_metadata.IsCreated = true
		fmt.Printf("Added project: %s\n", project_metadata.Name)
		metaData.Projects[project_metadata.Name] = project_metadata

		list_of_projects = append(list_of_projects, projectDarc.GetIdentityString())
	}

	projectCreatorDarc, ok := metaData.BaseIdToDarcMap[metaData.ProjectCreatorDarcBaseId]
	if !ok {
		panic(errors.New("Could not load project creator darc"))
	}
	// Register the sample project DARC with the value contract
	myvalue := []byte(strings.Join(list_of_projects, ";"))
	ctx := service.ClientTransaction{
		Instructions: []service.Instruction{{
			InstanceID: service.NewInstanceID(projectCreatorDarc.GetBaseID()),
			Nonce:      service.Nonce{},
			Index:      0,
			Length:     1,
			Spawn: &service.Spawn{
				ContractID: contracts.ContractValueID,
				Args: []service.Argument{{
					Name:  "value",
					Value: myvalue,
				}},
			},
		}},
	}
	err = ctx.Instructions[0].SignBy(projectCreatorDarc.GetBaseID(), signers...)
	if err != nil {
		panic(err)
	}

	_, err = cl.AddTransaction(ctx)
	if err != nil {
		panic(err)
	}

	metaData.AllProjectsListInstanceID = service.NewInstanceID(ctx.Instructions[0].Hash())
	pr, err := cl.WaitProof(metaData.AllProjectsListInstanceID, metaData.GenesisMsg.BlockInterval, nil)
	if pr.InclusionProof.Match() != true {
		panic(err)
	}

	// Create a users-projects map contract instance
	usersByte := []byte(strings.Join(user_ids, ";"))
	ctx = service.ClientTransaction{
		Instructions: []service.Instruction{{
			InstanceID: service.NewInstanceID(projectCreatorDarc.GetBaseID()),
			Nonce:      service.Nonce{},
			Index:      0,
			Length:     1,
			Spawn: &service.Spawn{
				ContractID: contracts.ContractUserProjectsMapID,
				Args: []service.Argument{{
					Name:  "allProjectsListInstanceID",
					Value: []byte(metaData.AllProjectsListInstanceID.Slice()),
				}, {
					Name:  "users",
					Value: usersByte,
				}},
			},
		}},
	}
	err = ctx.Instructions[0].SignBy(projectCreatorDarc.GetBaseID(), signers...)
	if err != nil {
		panic(err)
	}
	_, err = cl.AddTransaction(ctx)
	if err != nil {
		panic(err)
	}
	metaData.UserProjectsMapInstanceID = service.NewInstanceID(ctx.Instructions[0].Hash())
	pr, err = cl.WaitProof(metaData.UserProjectsMapInstanceID, metaData.GenesisMsg.BlockInterval, nil)
	if pr.InclusionProof.Match() != true {
		panic(err)
	}
}
