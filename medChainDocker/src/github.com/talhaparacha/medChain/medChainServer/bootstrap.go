package main

import (
	"time"

	"github.com/dedis/cothority/omniledger/contracts"
	"github.com/dedis/cothority/omniledger/darc"
	"github.com/dedis/cothority/omniledger/darc/expression"
	"github.com/dedis/cothority/omniledger/service"
	"github.com/talhaparacha/medChain/medChainServer/conf"
	"github.com/talhaparacha/medChain/medChainUtils"
)

func findUser(userCoordinates conf.UserCoordinates) *darc.Identity {
	admin_signer := admins[userCoordinates.I]
	manager_signer := managers[admin_signer.Identity().String()][userCoordinates.J]
	user_identity := users[manager_signer.Identity().String()][userCoordinates.K]
	return &user_identity
}

func findManager(userCoordinates conf.ManagerCoordinates) *darc.Signer {
	admin_signer := admins[userCoordinates.I]
	manager_signer := managers[admin_signer.Identity().String()][userCoordinates.J]
	return &manager_signer
}

func startSystem() {
	configuration, err := conf.ReadConf(configFileName)
	if err != nil {
		panic(err)
	}
	// We need to load suitable keys to initialize the system DARCs as per our context

	adminIds := []darc.Identity{}
	adminIdStrings := []string{}
	managerSigners := []darc.Signer{}
	for _, admin := range configuration.Admins {
		admin_signer := medChainUtils.LoadSignerEd25519(configuration.KeyDirectory+admin.PublicKey,
			configuration.KeyDirectory+admin.PrivateKey)
		id := admin_signer.Identity()
		adminIds = append(adminIds, id)
		adminIdString := id.String()
		adminIdStrings = append(adminIdStrings, adminIdString)
		admins = append(admins, admin_signer)
		managers[adminIdString] = []darc.Signer{}
		for _, manager := range admin.Managers {
			manager_signer := medChainUtils.LoadSignerEd25519(configuration.KeyDirectory+manager.PublicKey,
				configuration.KeyDirectory+manager.PrivateKey)
			managers[adminIdString] = append(managers[adminIdString], manager_signer)
			managerSigners = append(managerSigners, manager_signer)
			managerIdString := manager_signer.Identity().String()
			users[managerIdString] = []darc.Identity{}
			for _, user := range manager.Users {
				user_signer := medChainUtils.LoadSignerEd25519(configuration.KeyDirectory+user.PublicKey,
					configuration.KeyDirectory+user.PrivateKey)
				users[managerIdString] = append(users[managerIdString], user_signer.Identity())
			}
		}
	}

	// Create Genesis block
	genesisMsg, err = service.DefaultGenesisMsg(service.CurrentVersion, roster,
		[]string{}, adminIds...)
	if err != nil {
		panic(err)
	}
	gDarc := &genesisMsg.GenesisDarc
	gDarc.Rules.UpdateSign(expression.InitAndExpr(adminIdStrings...))
	gDarc.Rules.AddRule("spawn:darc", gDarc.Rules.GetSignExpr())

	genesisMsg.BlockInterval = time.Second
	genesisBlock, err = cl.CreateGenesisBlock(genesisMsg)
	if err != nil {
		panic(err)
	}

	// Create a DARC for admins of each hospital
	for _, admin_signer := range admins {
		owners := []darc.Identity{darc.NewIdentityDarc(gDarc.GetID())}
		signers := []darc.Identity{admin_signer.Identity()}
		rules := darc.InitRules(owners, signers)
		rules.AddRule("spawn:darc", rules.GetSignExpr())
		tempDarc, err := createDarc(cl, gDarc, genesisMsg.BlockInterval, rules, "Single Admin darc", admins...)
		if err != nil {
			panic(err)
		}
		adminsDarcsMap[admin_signer.Identity().String()] = tempDarc
	}

	// Create a DARC for managers of each hospital
	managersDarcsIdString := []string{}
	for _, admin_signer := range admins {
		overall_owners := []darc.Identity{admin_signer.Identity()}
		overall_signers := []string{}
		admin_darc := adminsDarcsMap[admin_signer.Identity().String()]
		for _, manager_signer := range managers[admin_signer.Identity().String()] {
			owners := []darc.Identity{admin_signer.Identity()}
			signers := []darc.Identity{manager_signer.Identity()}
			rules := darc.InitRules(owners, signers)
			rules.AddRule("spawn:darc", rules.GetSignExpr())
			tempDarc, err := createDarc(cl, admin_darc, genesisMsg.BlockInterval, rules,
				"Single Manager darc", admin_signer)
			if err != nil {
				panic(err)
			}
			managersDarcsMap[manager_signer.Identity().String()] = tempDarc
			managersDarcsMapWithDarcId[tempDarc.GetIdentityString()] = tempDarc
			overall_signers = append(overall_signers, tempDarc.GetIdentityString())
		}
		rules := darc.InitRules(overall_owners, []darc.Identity{})
		rules.UpdateSign(expression.InitAndExpr(overall_signers...))
		tempDarc, err := createDarc(cl, admin_darc, genesisMsg.BlockInterval, rules,
			"Managers darc", admin_signer)
		if err != nil {
			panic(err)
		}
		managersListDarcsMap[admin_signer.Identity().String()] = tempDarc
		managersDarcsIdString = append(managersDarcsIdString, tempDarc.GetIdentityString())
	}

	// Create a collective managers DARC
	rules := darc.InitRules(adminIds, []darc.Identity{})
	rules.UpdateSign(expression.InitAndExpr(managersDarcsIdString...))
	rules.AddRule("spawn:darc", rules.GetSignExpr())
	rules.AddRule("spawn:value", rules.GetSignExpr())
	rules.AddRule("spawn:UserProjectsMap", expression.InitOrExpr(managersDarcsIdString...))
	rules.AddRule("invoke:update", rules["spawn:UserProjectsMap"])
	allManagersDarc, err = createDarc(cl, gDarc, genesisMsg.BlockInterval, rules,
		"AllManagers darc", admins...)
	if err != nil {
		panic(err)
	}

	// Create a DARC for users of each hospital
	userDarcsIds := []darc.Identity{}

	for _, admin_signer := range admins {
		for _, manager_signer := range managers[admin_signer.Identity().String()] {
			overall_owners := []darc.Identity{manager_signer.Identity()}
			overall_signers := []string{}
			manager_darc := managersDarcsMap[manager_signer.Identity().String()]
			for _, user_identity := range users[manager_signer.Identity().String()] {
				owners := []darc.Identity{darc.NewIdentityDarc(managersDarcsMap[manager_signer.Identity().String()].GetID())}
				signers := []darc.Identity{user_identity}
				rules := darc.InitRules(owners, signers)
				tempDarc, err := createDarc(cl, manager_darc, genesisMsg.BlockInterval, rules,
					"Single User darc", manager_signer)
				if err != nil {
					panic(err)
				}
				usersDarcsMap[user_identity.String()] = tempDarc
				usersDarcsMapWithDarcId[tempDarc.GetIdentityString()] = tempDarc
				overall_signers = append(overall_signers, tempDarc.GetIdentityString())
			}
			rules := darc.InitRules(overall_owners, []darc.Identity{})
			rules.UpdateSign(expression.InitOrExpr(overall_signers...))
			tempDarc, err := createDarc(cl, manager_darc, genesisMsg.BlockInterval, rules,
				"Users darc", manager_signer)
			if err != nil {
				panic(err)
			}
			usersListDarcsMap[manager_signer.Identity().String()] = tempDarc
			userDarcsIds = append(userDarcsIds, darc.NewIdentityDarc(tempDarc.GetID()))
		}
	}

	// Create a collective users DARC
	collectiveUserDarcOwner := []darc.Identity{darc.NewIdentityDarc(allManagersDarc.GetID())}
	rules = darc.InitRules(collectiveUserDarcOwner, userDarcsIds)
	rules.AddRule("spawn:ProjectList", rules.GetSignExpr())
	allUsersDarc, err = createDarc(cl, allManagersDarc, genesisMsg.BlockInterval, rules,
		"AllUsers darc", managerSigners...)
	if err != nil {
		panic(err)
	}

	var allProjectsListInstanceID service.InstanceID
	for _, project := range configuration.Projects {
		owners := []darc.Identity{}
		for _, managerCoordinates := range project.ManagerOwners {
			manager_signer := findManager(managerCoordinates)
			id := darc.NewIdentityDarc(managersDarcsMap[manager_signer.Identity().String()].GetID())
			owners = append(owners, id)
		}
		signers := []darc.Identity{}
		for _, userCoordinates := range project.SigningUsers {
			user_identity := findUser(userCoordinates)
			id := darc.NewIdentityDarc(usersDarcsMap[user_identity.String()].GetID())
			signers = append(owners, id)
		}
		projectDarcRules := darc.InitRules(owners, signers)
		for _, rule := range project.Rules {
			usersIdString := []string{}
			for _, userCoordinates := range rule.Users {
				user_identity := findUser(userCoordinates)
				idString := usersDarcsMap[user_identity.String()].GetIdentityString()
				usersIdString = append(usersIdString, idString)
			}
			var expr expression.Expr
			switch rule.ExprType {
			case "SIGNERS":
				expr = projectDarcRules.GetSignExpr()
			case "OR":
				expr = expression.InitOrExpr(usersIdString...)
			case "AND":
				expr = expression.InitAndExpr(usersIdString...)
			}
			projectDarcRules.AddRule(darc.Action(rule.Action), expr)
		}
		projectDarc, err := createDarc(cl, allManagersDarc, genesisMsg.BlockInterval, projectDarcRules,
			project.Name, managerSigners...)
		if err != nil {
			panic(err)
		}

		// Register the sample project DARC with the value contract
		myvalue := []byte(projectDarc.GetIdentityString())
		ctx := service.ClientTransaction{
			Instructions: []service.Instruction{{
				InstanceID: service.NewInstanceID(allManagersDarc.GetBaseID()),
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
		err = ctx.Instructions[0].SignBy(allManagersDarc.GetBaseID(), managerSigners...)
		if err != nil {
			panic(err)
		}

		_, err = cl.AddTransaction(ctx)
		if err != nil {
			panic(err)
		}

		allProjectsListInstanceID = service.NewInstanceID(ctx.Instructions[0].Hash())
		pr, err := cl.WaitProof(allProjectsListInstanceID, genesisMsg.BlockInterval, nil)
		if pr.InclusionProof.Match() != true {
			panic(err)
		}
	}
}
