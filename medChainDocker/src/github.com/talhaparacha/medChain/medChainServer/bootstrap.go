package main

import (
	"errors"
	"fmt"
	"time"

	"github.com/dedis/cothority/omniledger/contracts"
	"github.com/dedis/cothority/omniledger/darc"
	"github.com/dedis/cothority/omniledger/darc/expression"
	"github.com/dedis/cothority/omniledger/service"
	"github.com/talhaparacha/medChain/medChainServer/conf"
	"github.com/talhaparacha/medChain/medChainUtils"
)

// Admins, Managers and Users as per the context defined in system diagram
var super_admins []darc.Signer
var admins map[string][]darc.Signer
var managers map[string][]darc.Signer
var users map[string][]darc.Identity

func findUser(userCoordinates conf.UserCoordinates) *darc.Identity {
	super_admin_signer := super_admins[userCoordinates.I]
	admin_signer := admins[super_admin_signer.Identity().String()][userCoordinates.J]
	manager_signer := managers[admin_signer.Identity().String()][userCoordinates.K]
	user_identity := users[manager_signer.Identity().String()][userCoordinates.L]
	return &user_identity
}

func findManager(userCoordinates conf.ManagerCoordinates) *darc.Signer {
	super_admin_signer := super_admins[userCoordinates.I]
	admin_signer := admins[super_admin_signer.Identity().String()][userCoordinates.J]
	manager_signer := managers[admin_signer.Identity().String()][userCoordinates.K]
	return &manager_signer
}

func addDarcToMaps(NewDarc *darc.Darc, id string, mapId map[string]string) {
	IDHash := medChainUtils.IDToHexString(NewDarc.GetBaseID())
	baseIdToDarcMap[IDHash] = NewDarc
	darcIdToBaseIdMap[NewDarc.GetIdentityString()] = IDHash
	mapId[id] = IDHash
}

func loadKeys(configuration *conf.Configuration) {

	super_admins = []darc.Signer{}
	admins = make(map[string][]darc.Signer)
	managers = make(map[string][]darc.Signer)
	users = make(map[string][]darc.Identity)

	for _, hospital := range configuration.Hospitals {

		super_admin_signer := medChainUtils.LoadSignerEd25519(configuration.KeyDirectory+hospital.PublicKey, configuration.KeyDirectory+hospital.PrivateKey)
		super_admins = append(super_admins, super_admin_signer)
		//fmt.Println(super_admin_signer.Identity().String())
		super_admin_IDString := super_admin_signer.Identity().String()
		admins[super_admin_IDString] = []darc.Signer{}

		for _, admin := range hospital.Admins {

			admin_signer := medChainUtils.LoadSignerEd25519(configuration.KeyDirectory+admin.PublicKey, configuration.KeyDirectory+admin.PrivateKey)
			admins[super_admin_IDString] = append(admins[super_admin_IDString], admin_signer)
			adminIdString := admin_signer.Identity().String()
			managers[adminIdString] = []darc.Signer{}

			for _, manager := range admin.Managers {

				manager_signer := medChainUtils.LoadSignerEd25519(configuration.KeyDirectory+manager.PublicKey, configuration.KeyDirectory+manager.PrivateKey)
				managers[adminIdString] = append(managers[adminIdString], manager_signer)
				managerIdString := manager_signer.Identity().String()
				users[managerIdString] = []darc.Identity{}

				for _, user := range manager.Users {

					user_signer := medChainUtils.LoadSignerEd25519(configuration.KeyDirectory+user.PublicKey, configuration.KeyDirectory+user.PrivateKey)
					users[managerIdString] = append(users[managerIdString], user_signer.Identity())

				}
			}
		}
	}
}

func createGenesis() {
	super_adminsIds := []darc.Identity{}
	super_adminsIDStrings := []string{}

	for _, super_admin := range super_admins {
		super_adminsIds = append(super_adminsIds, super_admin.Identity())
		super_adminsIDStrings = append(super_adminsIDStrings, super_admin.Identity().String())
	}

	// Create Genesis block
	genesisMsg, err = service.DefaultGenesisMsg(service.CurrentVersion, roster,
		[]string{}, super_adminsIds...)
	if err != nil {
		panic(err)
	}
	gDarc := &genesisMsg.GenesisDarc
	gDarc.Rules.UpdateSign(expression.InitAndExpr(super_adminsIDStrings...))
	gDarc.Rules.AddRule("spawn:darc", gDarc.Rules.GetSignExpr())

	genesisMsg.BlockInterval = time.Second
	genesisBlock, err = cl.CreateGenesisBlock(genesisMsg)
	if err != nil {
		panic(err)
	}
}

func createSuperAdminsDarcs() {
	// Create a DARC for admins of each hospital
	gDarc := &genesisMsg.GenesisDarc
	for _, super_admin_signer := range super_admins {
		owners := []darc.Identity{darc.NewIdentityDarc(gDarc.GetID())}
		signers := []darc.Identity{super_admin_signer.Identity()}
		rules := darc.InitRulesWith(owners, signers, "invoke:evolve")
		rules.AddRule("spawn:darc", rules.GetSignExpr())
		tempDarc, err := createDarc(cl, gDarc, genesisMsg.BlockInterval, rules, "Single Super Admin darc", super_admins...)
		if err != nil {
			panic(err)
		}
		addDarcToMaps(tempDarc, super_admin_signer.Identity().String(), superAdminsDarcsMap)
		fmt.Println("add super admin darc", super_admin_signer.Identity().String())
	}
}

func createAllSuperAdminsDarc() {
	darcIdList := []string{}
	gDarc := &genesisMsg.GenesisDarc
	owners := []darc.Identity{darc.NewIdentityDarc(gDarc.GetID())}
	for _, super_admin_signer := range super_admins {
		super_admin_IDString := super_admin_signer.Identity().String()
		super_admin_darc, ok := getDarcFromId(super_admin_IDString, baseIdToDarcMap, superAdminsDarcsMap)
		if !ok {
			fmt.Println("failed super admin darc", super_admin_IDString)
			panic(errors.New("Could not load super admin darc"))
		}
		darcIdList = append(darcIdList, super_admin_darc.GetIdentityString())
	}
	rules := darc.InitRulesWith(owners, []darc.Identity{}, "invoke:evolve")
	rules.UpdateSign(expression.InitAndExpr(darcIdList...)) // OR or AND ?
	rules.AddRule("spawn:darc", rules.GetSignExpr())
	allSuperAdminsDarc, err = createDarc(cl, gDarc, genesisMsg.BlockInterval, rules,
		"All Super Admins darc", super_admins...)
	if err != nil {
		panic(err)
	}
}

func createAdminsDarcs() {
	// Create a DARC for admins of each hospital
	for _, super_admin_signer := range super_admins {
		super_adminIDString := super_admin_signer.Identity().String()
		super_admin_darc, ok := getDarcFromId(super_adminIDString, baseIdToDarcMap, superAdminsDarcsMap)
		if !ok {
			fmt.Println("failed super admin darc", super_adminIDString)
			panic(errors.New("Could not load super admin darc"))
		}
		for _, admin_signer := range admins[super_adminIDString] {
			owners := []darc.Identity{darc.NewIdentityDarc(super_admin_darc.GetID())}
			signers := []darc.Identity{admin_signer.Identity()}
			rules := darc.InitRulesWith(owners, signers, "invoke:evolve")
			rules.AddRule("spawn:darc", rules.GetSignExpr())
			tempDarc, err := createDarc(cl, super_admin_darc, genesisMsg.BlockInterval, rules, "Single Admin darc", super_admin_signer)
			if err != nil {
				panic(err)
			}
			fmt.Println("add admin darc", admin_signer.Identity().String())
			addDarcToMaps(tempDarc, admin_signer.Identity().String(), adminsDarcsMap)
		}
	}
}

func createAdminsListDarcs() {
	// Create a DARC for admins of each hospital
	adminsListDarcsIds := []string{}
	super_admin_signers := []darc.Signer{}
	for _, super_admin_signer := range super_admins {
		super_adminIDString := super_admin_signer.Identity().String()
		super_admin_darc, ok := getDarcFromId(super_adminIDString, baseIdToDarcMap, superAdminsDarcsMap)
		if !ok {
			fmt.Println("failed super admin darc", super_adminIDString)
			panic(errors.New("Could not load super admin darc"))
		}
		overall_owners := []darc.Identity{super_admin_signer.Identity()}
		overall_signers := []string{}
		for _, admin_signer := range admins[super_adminIDString] {
			admin_darc, ok := getDarcFromId(admin_signer.Identity().String(), baseIdToDarcMap, adminsDarcsMap)
			if !ok {
				fmt.Println("failed admin darc", admin_signer.Identity().String())
				panic(errors.New("Could not load admin darc"))
			}
			overall_signers = append(overall_signers, admin_darc.GetIdentityString())
		}
		rules := darc.InitRulesWith(overall_owners, []darc.Identity{}, "invoke:evolve")
		rules.UpdateSign(expression.InitOrExpr(overall_signers...))
		adminsListDarc, err := createDarc(cl, super_admin_darc, genesisMsg.BlockInterval, rules, "Admins List, Super Admin :"+super_adminIDString, super_admin_signer)
		if err != nil {
			panic(err)
		}
		addDarcToMaps(adminsListDarc, super_admin_signer.Identity().String(), adminsListDarcsMap)
		adminsListDarcsIds = append(adminsListDarcsIds, adminsListDarc.GetIdentityString())
		super_admin_signers = append(super_admin_signers, super_admin_signer)
	}
	createAllAdminsDarc(adminsListDarcsIds, super_admin_signers)
}

func createAllAdminsDarc(adminsListDarcsIds []string, super_admin_signers []darc.Signer) {
	// Create a collective users DARC
	allAdminsDarcOwner := []darc.Identity{darc.NewIdentityDarc(allSuperAdminsDarc.GetID())}
	rules := darc.InitRulesWith(allAdminsDarcOwner, []darc.Identity{}, "invoke:evolve")
	rules.UpdateSign(expression.InitAndExpr(adminsListDarcsIds...)) // OR or AND ?
	rules.AddRule("spawn:darc", rules.GetSignExpr())
	allAdminsDarc, err = createDarc(cl, allSuperAdminsDarc, genesisMsg.BlockInterval, rules,
		"AllAdmins darc", super_admin_signers...)
	if err != nil {
		panic(err)
	}
	allAdminsBaseID = medChainUtils.IDToHexString(allAdminsDarc.GetBaseID())
	baseIdToDarcMap[allAdminsBaseID] = allAdminsDarc
	darcIdToBaseIdMap[allAdminsDarc.GetIdentityString()] = allAdminsBaseID
}

func createManagersDarcs() {
	// Create a DARC for managers of each hospital
	for _, super_admin_signer := range super_admins {
		super_adminIDString := super_admin_signer.Identity().String()
		for _, admin_signer := range admins[super_adminIDString] {
			admin_darc, ok := getDarcFromId(admin_signer.Identity().String(), baseIdToDarcMap, adminsDarcsMap)
			if !ok {
				fmt.Println("failed admin darc", admin_signer.Identity().String())
				panic(errors.New("Could not load admin darc"))
			}
			for _, manager_signer := range managers[admin_signer.Identity().String()] {
				owners := []darc.Identity{admin_signer.Identity()}
				signers := []darc.Identity{manager_signer.Identity()}
				rules := darc.InitRulesWith(owners, signers, "invoke:evolve")
				rules.AddRule("spawn:darc", rules.GetSignExpr())
				tempDarc, err := createDarc(cl, admin_darc, genesisMsg.BlockInterval, rules,
					"Single Manager darc", admin_signer)
				if err != nil {
					panic(err)
				}
				addDarcToMaps(tempDarc, manager_signer.Identity().String(), managersDarcsMap)
				fmt.Println("add manager darc", manager_signer.Identity().String())
			}
		}
	}
}

func createManagersListDarcs() {
	// Create a DARC for admins of each hospital
	listsOfLevel1Ids := []string{}
	super_admin_signers := []darc.Signer{}
	for _, super_admin_signer := range super_admins {
		super_adminIDString := super_admin_signer.Identity().String()
		listsOfLevel0Ids := []string{}
		for _, admin_signer := range admins[super_adminIDString] {
			listLevel0 := createManagersListLevel0(admin_signer)
			listsOfLevel0Ids = append(listsOfLevel0Ids, listLevel0.GetIdentityString())
		}
		listLevel1 := createManagersListLevel1(super_admin_signer, listsOfLevel0Ids)
		listsOfLevel1Ids = append(listsOfLevel1Ids, listLevel1.GetIdentityString())
		super_admin_signers = append(super_admin_signers, super_admin_signer)
	}
	createAllManagersDarc(listsOfLevel1Ids, super_admin_signers)
}

func createManagersListLevel0(admin_signer darc.Signer) *darc.Darc {
	adminIDString := admin_signer.Identity().String()
	admin_darc, ok := getDarcFromId(adminIDString, baseIdToDarcMap, adminsDarcsMap)
	if !ok {
		fmt.Println("failed admin darc", adminIDString)
		panic(errors.New("Could not load admin darc"))
	}
	overall_owners := []darc.Identity{admin_signer.Identity()}
	overall_signers := []string{}
	for _, manager_signer := range managers[adminIDString] {
		manager_darc, ok := getDarcFromId(manager_signer.Identity().String(), baseIdToDarcMap, managersDarcsMap)
		if !ok {
			fmt.Println("failed manager darc", manager_signer.Identity().String())
			panic(errors.New("Could not load manager darc"))
		}
		overall_signers = append(overall_signers, manager_darc.GetIdentityString())
	}
	rules := darc.InitRulesWith(overall_owners, []darc.Identity{}, "invoke:evolve")
	rules.UpdateSign(expression.InitOrExpr(overall_signers...))
	managersListDarc, err := createDarc(cl, admin_darc, genesisMsg.BlockInterval, rules, "Managers List Level 0, Admin :"+adminIDString, admin_signer)
	if err != nil {
		panic(err)
	}
	addDarcToMaps(managersListDarc, admin_signer.Identity().String(), managersListLevel0DarcsMap)
	return managersListDarc
}

func createManagersListLevel1(super_admin_signer darc.Signer, listsOfLevel0Ids []string) *darc.Darc {
	superAdminIDString := super_admin_signer.Identity().String()
	super_admin_darc, ok := getDarcFromId(superAdminIDString, baseIdToDarcMap, superAdminsDarcsMap)
	if !ok {
		fmt.Println("failed super admin darc", superAdminIDString)
		panic(errors.New("Could not load super admin darc"))
	}
	overall_owners := []darc.Identity{super_admin_signer.Identity()}
	rules := darc.InitRulesWith(overall_owners, []darc.Identity{}, "invoke:evolve")
	rules.UpdateSign(expression.InitOrExpr(listsOfLevel0Ids...)) // OR or AND ?
	managersListDarc, err := createDarc(cl, super_admin_darc, genesisMsg.BlockInterval, rules, "Managers List Level1 , Super Admin :"+superAdminIDString, super_admin_signer)
	if err != nil {
		panic(err)
	}
	addDarcToMaps(managersListDarc, super_admin_signer.Identity().String(), managersListLevel1DarcsMap)
	return managersListDarc
}

func createAllManagersDarc(managersListDarcsIds []string, super_admin_signers []darc.Signer) {
	// Create a collective users DARC
	allManagersDarcOwner := []darc.Identity{darc.NewIdentityDarc(allSuperAdminsDarc.GetID())}
	rules := darc.InitRulesWith(allManagersDarcOwner, []darc.Identity{}, "invoke:evolve")
	rules.UpdateSign(expression.InitAndExpr(managersListDarcsIds...)) // OR or AND ?
	rules.AddRule("spawn:darc", rules.GetSignExpr())
	rules.AddRule("spawn:value", rules.GetSignExpr())
	rules.AddRule("spawn:UserProjectsMap", expression.InitOrExpr(managersListDarcsIds...))
	rules.AddRule("invoke:update", rules["spawn:UserProjectsMap"])
	allManagersDarc, err = createDarc(cl, allSuperAdminsDarc, genesisMsg.BlockInterval, rules,
		"AllManagers darc", super_admin_signers...)
	if err != nil {
		panic(err)
	}
	allManagersBaseID = medChainUtils.IDToHexString(allManagersDarc.GetBaseID())
	baseIdToDarcMap[allManagersBaseID] = allManagersDarc
	darcIdToBaseIdMap[allManagersDarc.GetIdentityString()] = allManagersBaseID
}

func createUsersDarcs() {
	for _, super_admin_signer := range super_admins {
		super_adminIDString := super_admin_signer.Identity().String()
		for _, admin_signer := range admins[super_adminIDString] {
			adminIDString := admin_signer.Identity().String()
			for _, manager_signer := range managers[adminIDString] {
				managerIDString := manager_signer.Identity().String()
				manager_darc, ok := getDarcFromId(managerIDString, baseIdToDarcMap, managersDarcsMap)
				if !ok {
					fmt.Println("failed manager darc", managerIDString)
					panic(errors.New("Could not load manager darc"))
				}
				for _, user_identity := range users[managerIDString] {
					owners := []darc.Identity{darc.NewIdentityDarc(manager_darc.GetID())}
					signers := []darc.Identity{user_identity}
					rules := darc.InitRulesWith(owners, signers, "invoke:evolve")
					tempDarc, err := createDarc(cl, manager_darc, genesisMsg.BlockInterval, rules,
						"Single User darc", manager_signer)
					if err != nil {
						panic(err)
					}
					addDarcToMaps(tempDarc, user_identity.String(), usersDarcsMap)
					fmt.Println("add user darc", user_identity.String())
				}
			}
		}
	}
}

func createUsersListDarcs() {
	// Create a DARC for admins of each hospital
	listsOfLevel2Ids := []string{}
	super_admin_signers := []darc.Signer{}
	for _, super_admin_signer := range super_admins {
		super_adminIDString := super_admin_signer.Identity().String()
		listsOfLevel1Ids := []string{}
		for _, admin_signer := range admins[super_adminIDString] {
			adminIDString := admin_signer.Identity().String()
			listsOfLevel0Ids := []string{}
			for _, manager_signer := range managers[adminIDString] {
				listLevel0 := createUsersListLevel0(manager_signer)
				listsOfLevel0Ids = append(listsOfLevel0Ids, listLevel0.GetIdentityString())
			}
			listLevel1 := createUsersListLevel1(admin_signer, listsOfLevel0Ids)
			listsOfLevel1Ids = append(listsOfLevel1Ids, listLevel1.GetIdentityString())
		}
		listLevel2 := createUsersListLevel2(super_admin_signer, listsOfLevel1Ids)
		listsOfLevel2Ids = append(listsOfLevel2Ids, listLevel2.GetIdentityString())
		super_admin_signers = append(super_admin_signers, super_admin_signer)
	}
	createAllUsersDarc(listsOfLevel2Ids, super_admin_signers)
}

func createUsersListLevel0(manager_signer darc.Signer) *darc.Darc {
	managerIDString := manager_signer.Identity().String()
	manager_darc, ok := getDarcFromId(managerIDString, baseIdToDarcMap, managersDarcsMap)
	if !ok {
		fmt.Println("failed manager darc", managerIDString)
		panic(errors.New("Could not load manager darc"))
	}
	overall_owners := []darc.Identity{manager_signer.Identity()}
	overall_signers := []string{}
	for _, user_identity := range users[managerIDString] {
		user_darc, ok := getDarcFromId(user_identity.String(), baseIdToDarcMap, usersDarcsMap)
		if !ok {
			fmt.Println("failed user darc", user_identity.String())
			panic(errors.New("Could not load user darc"))
		}
		overall_signers = append(overall_signers, user_darc.GetIdentityString())
	}
	rules := darc.InitRulesWith(overall_owners, []darc.Identity{}, "invoke:evolve")
	rules.UpdateSign(expression.InitOrExpr(overall_signers...))
	usersListDarc, err := createDarc(cl, manager_darc, genesisMsg.BlockInterval, rules, "Users List Level 0, Manager :"+managerIDString, manager_signer)
	if err != nil {
		panic(err)
	}
	addDarcToMaps(usersListDarc, manager_signer.Identity().String(), usersListLevel0DarcsMap)
	return usersListDarc
}

func createUsersListLevel1(admin_signer darc.Signer, listsOfLevel0Ids []string) *darc.Darc {
	adminIDString := admin_signer.Identity().String()
	admin_darc, ok := getDarcFromId(adminIDString, baseIdToDarcMap, adminsDarcsMap)
	if !ok {
		fmt.Println("failed admin darc", adminIDString)
		panic(errors.New("Could not load admin darc"))
	}
	overall_owners := []darc.Identity{admin_signer.Identity()}
	rules := darc.InitRulesWith(overall_owners, []darc.Identity{}, "invoke:evolve")
	rules.UpdateSign(expression.InitOrExpr(listsOfLevel0Ids...)) // OR or AND ?
	usersListDarc, err := createDarc(cl, admin_darc, genesisMsg.BlockInterval, rules, "Users List Level1 , Admin :"+adminIDString, admin_signer)
	if err != nil {
		panic(err)
	}
	addDarcToMaps(usersListDarc, admin_signer.Identity().String(), usersListLevel1DarcsMap)
	return usersListDarc
}

func createUsersListLevel2(super_admin_signer darc.Signer, listsOfLevel0Ids []string) *darc.Darc {
	superAdminIDString := super_admin_signer.Identity().String()
	super_admin_darc, ok := getDarcFromId(superAdminIDString, baseIdToDarcMap, superAdminsDarcsMap)
	if !ok {
		fmt.Println("failed super admin darc", superAdminIDString)
		panic(errors.New("Could not load super admin darc"))
	}
	overall_owners := []darc.Identity{super_admin_signer.Identity()}
	rules := darc.InitRulesWith(overall_owners, []darc.Identity{}, "invoke:evolve")
	rules.UpdateSign(expression.InitOrExpr(listsOfLevel0Ids...)) // OR or AND ?
	usersListDarc, err := createDarc(cl, super_admin_darc, genesisMsg.BlockInterval, rules, "Users List Level2 , Super Admin :"+superAdminIDString, super_admin_signer)
	if err != nil {
		panic(err)
	}
	addDarcToMaps(usersListDarc, super_admin_signer.Identity().String(), usersListLevel2DarcsMap)
	return usersListDarc
}

// func createUsersListDarcs() {
// 	// Create a DARC for admins of each hospital
// 	usersListDarcsIds := []string{}
// 	manager_signers := []darc.Signer{}
// 	for _, super_admin_signer := range super_admins {
// 		super_adminIDString := super_admin_signer.Identity().String()
// 		for _, admin_signer := range admins[super_adminIDString] {
// 			adminIDString := admin_signer.Identity().String()
// 			for _, manager_signer := range managers[adminIDString] {
// 				managerIDString := manager_signer.Identity().String()
// 				manager_darc, ok := getDarcFromId(managerIDString, baseIdToDarcMap, managersDarcsMap)
// 				overall_owners := []darc.Identity{manager_signer.Identity()}
// 				overall_signers := []string{}
// 				if ok {
// 					for _, user_identity := range users[managerIDString] {
// 						user_darc, ok := getDarcFromId(user_identity.String(), baseIdToDarcMap, usersDarcsMap)
// 						if ok {
// 							overall_signers = append(overall_signers, user_darc.GetIdentityString())
// 						}
// 					}
// 				}
// 				if len(overall_signers) > 0 {
// 					rules := darc.InitRulesWith(overall_owners, []darc.Identity{}, "invoke:evolve")
// 					rules.UpdateSign(expression.InitOrExpr(overall_signers...))
// 					usersListDarc, err := createDarc(cl, manager_darc, genesisMsg.BlockInterval, rules, "Users List, Manager :"+managerIDString, manager_signer)
// 					if err != nil {
// 						panic(err)
// 					}
// 					addDarcToMaps(usersListDarc, baseIdToDarcMap, manager_signer.Identity().String(), usersListDarcsMap, usersListDarcsMapWithDarcId, darcIdToBaseIdMap)
// 					usersListDarcsIds = append(usersListDarcsIds, usersListDarc.GetIdentityString())
// 					manager_signers = append(manager_signers, manager_signer)
// 				}
// 			}
// 		}
// 	}
// 	if len(usersListDarcsIds) > 0 {
// 		createAllUsersDarc(usersListDarcsIds, manager_signers)
// 	}
// }
//

func createAllUsersDarc(usersListDarcsIds []string, super_admin_signers []darc.Signer) {
	// Create a collective users DARC
	collectiveUserDarcOwner := []darc.Identity{darc.NewIdentityDarc(allSuperAdminsDarc.GetID())}
	rules := darc.InitRulesWith(collectiveUserDarcOwner, []darc.Identity{}, "invoke:evolve")
	rules.UpdateSign(expression.InitOrExpr(usersListDarcsIds...)) // OR or AND ?
	rules.AddRule("spawn:ProjectList", rules.GetSignExpr())
	allUsersDarc, err = createDarc(cl, allSuperAdminsDarc, genesisMsg.BlockInterval, rules,
		"AllUsers darc", super_admin_signers...)
	if err != nil {
		panic(err)
	}
	allUsersBaseID = medChainUtils.IDToHexString(allUsersDarc.GetBaseID())
	baseIdToDarcMap[allUsersBaseID] = allUsersDarc
	darcIdToBaseIdMap[allUsersDarc.GetIdentityString()] = allUsersBaseID
}

func createProjectDarcs(configuration *conf.Configuration) {

	var allProjectsListInstanceID service.InstanceID
	manager_signers := []darc.Signer{}
	for _, super_admin_signer := range super_admins {
		for _, admin_signer := range admins[super_admin_signer.Identity().String()] {
			for _, manager_signer := range managers[admin_signer.Identity().String()] {
				manager_signers = append(manager_signers, manager_signer)
			}
		}
	}

	for _, project := range configuration.Projects {

		owners := []darc.Identity{}
		for _, managerCoordinates := range project.ManagerOwners {
			manager_signer := findManager(managerCoordinates)
			manager_darc, ok := getDarcFromId(manager_signer.Identity().String(), baseIdToDarcMap, managersDarcsMap)
			if ok {
				id := darc.NewIdentityDarc(manager_darc.GetID())
				owners = append(owners, id)
			}
		}

		signers := []darc.Identity{}
		for _, userCoordinates := range project.SigningUsers {
			user_identity := findUser(userCoordinates)
			user_darc, ok := getDarcFromId(user_identity.String(), baseIdToDarcMap, usersDarcsMap)
			if ok {
				id := darc.NewIdentityDarc(user_darc.GetID())
				signers = append(owners, id)
			}
		}

		projectDarcRules := darc.InitRulesWith(owners, signers, "invoke:evolve")
		for _, rule := range project.Rules {
			usersIdString := []string{}
			for _, userCoordinates := range rule.Users {
				user_identity := findUser(userCoordinates)
				user_darc, _ := getDarcFromId(user_identity.String(), baseIdToDarcMap, usersDarcsMap)
				idString := user_darc.GetIdentityString()
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
			project.Name, manager_signers...)
		if err != nil {
			panic(err)
		}

		addDarcToMaps(projectDarc, project.Name, projectsDarcsMap)

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
		err = ctx.Instructions[0].SignBy(allManagersDarc.GetBaseID(), manager_signers...)
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

func startSystem() {
	configuration, err := conf.ReadConf(configFileName)
	if err != nil {
		panic(err)
	}
	// We need to load suitable keys to initialize the system DARCs as per our context

	loadKeys(configuration)

	createGenesis()

	createSuperAdminsDarcs()

	createAllSuperAdminsDarc()

	createAdminsDarcs()

	createAdminsListDarcs()

	createManagersDarcs()

	createManagersListDarcs()

	createUsersDarcs()

	createUsersListDarcs()

	createProjectDarcs(configuration)

}
