package main

import (
	"errors"
	"strings"

	"github.com/dedis/cothority/omniledger/contracts"
	"github.com/dedis/cothority/omniledger/darc"
	"github.com/dedis/cothority/omniledger/darc/expression"
	"github.com/dedis/cothority/omniledger/service"
	"github.com/talhaparacha/medChain/medChainServer/conf"
	"github.com/talhaparacha/medChain/medChainServer/metadata"
	"github.com/talhaparacha/medChain/medChainUtils"
)

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
	rules.AddRule("invoke:update", expression.InitOrExpr(all_super_admin_darc.GetIdentityString()))
	rules.AddRule("invoke:update", expression.InitOrExpr(all_admin_darc.GetIdentityString()))
	rules.AddRule("invoke:update", expression.InitOrExpr(all_manager_darc.GetIdentityString()))

	projectCreatorDarc, err := createDarc(cl, genesisDarc, metaData.GenesisMsg.BlockInterval, rules, "Project Creator Darc", signers...)
	if err != nil {
		panic(err)
	}
	metaData.ProjectCreatorDarcBaseId = addDarcToMaps(projectCreatorDarc, metaData)
}

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

func createProjectDarcs(configuration *conf.Configuration, metaData *metadata.Metadata, signers []darc.Signer) {

	genesisDarc, ok := metaData.BaseIdToDarcMap[metaData.GenesisDarcBaseId]
	if !ok {
		panic(errors.New("Could not load the genesisDarc"))
	}

	var allProjectsListInstanceID service.InstanceID

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
			manager_metadata.Projects = append(manager_metadata.Projects, project_metadata)
			project_hospitals[manager_metadata.Hospital.SuperAdmin.Id.String()] = manager_metadata.Hospital
		}

		darc_admins := []string{}
		for _, hospital_metadata := range project_hospitals {
			for _, admin_metadata := range hospital_metadata.Admins {
				admin_darc, ok := metaData.BaseIdToDarcMap[admin_metadata.DarcBaseId]
				if !ok {
					panic(errors.New("Could not load admin darc"))
				}
				darc_admins = append(darc_admins, admin_darc.GetIdentityString())
			}
			hospital_metadata.SuperAdmin.Projects = append(hospital_metadata.SuperAdmin.Projects, project_metadata)
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
			user_metadata.Projects = append(user_metadata.Projects, project_metadata)
		}

		projectDarcRules := darc.InitRulesWith([]darc.Identity{}, []darc.Identity{}, "invoke:evolve")
		projectDarcRules.UpdateRule("invoke:evolve", expression.InitOrExpr(string(medChainUtils.InitAtLeastTwoExpr(darc_managers)), string(medChainUtils.InitAtLeastTwoExpr(darc_admins))))
		projectDarcRules.AddRule("spawn:AuthGrant", projectDarcRules.GetSignExpr())
		projectDarcRules.AddRule("spawn:CreateQuery", projectDarcRules.GetSignExpr())

		for _, rule := range project.Rules {
			usersIdString := []string{}
			for _, user_index := range rule.Users {
				user_metadata := project_metadata.Users[user_index]
				user_darc, ok := metaData.BaseIdToDarcMap[user_metadata.DarcBaseId]
				if !ok {
					panic(errors.New("Could not load user darc"))
				}
				idString := user_darc.GetIdentityString()
				usersIdString = append(usersIdString, idString)
			}
			expr := expression.InitOrExpr(usersIdString...)
			projectDarcRules.AddRule(darc.Action(rule.Action), expr)
		}

		projectDarc, err := createDarc(cl, genesisDarc, metaData.GenesisMsg.BlockInterval, projectDarcRules,
			project.Name, signers...)
		if err != nil {
			panic(err)
		}

		project_metadata.DarcBaseId = addDarcToMaps(projectDarc, metaData)

		metaData.Projects[project_metadata.Id] = project_metadata

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

	allProjectsListInstanceID = service.NewInstanceID(ctx.Instructions[0].Hash())
	pr, err := cl.WaitProof(allProjectsListInstanceID, metaData.GenesisMsg.BlockInterval, nil)
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
					Value: []byte(allProjectsListInstanceID.Slice()),
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
	userProjectsMapInstanceID = service.NewInstanceID(ctx.Instructions[0].Hash())
	pr, err = cl.WaitProof(userProjectsMapInstanceID, metaData.GenesisMsg.BlockInterval, nil)
	if pr.InclusionProof.Match() != true {
		panic(err)
	}
}
