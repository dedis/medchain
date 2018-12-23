package main

import (
	"errors"
	"fmt"
	"time"

	"github.com/dedis/cothority/omniledger/darc"
	"github.com/dedis/cothority/omniledger/darc/expression"
	"github.com/dedis/cothority/omniledger/service"
	"github.com/talhaparacha/medChain/medChainServer/conf"
	"github.com/talhaparacha/medChain/medChainServer/metadata"
	"github.com/talhaparacha/medChain/medChainUtils"
)

// func findUser(userCoordinates conf.Coordinates) *darc.Identity {
// 	super_admin_signer := super_admins[userCoordinates.I]
// 	user_identity := users[super_admin_signer.Identity().String()][userCoordinates.J]
// 	return &user_identity
// }
//
// func findManager(userCoordinates conf.Coordinates) *darc.Identity {
// 	super_admin_signer := super_admins[userCoordinates.I]
// 	manager_identity := managers[super_admin_signer.Identity().String()][userCoordinates.J]
// 	return &manager_identity
// }

func addDarcToMaps(NewDarc *darc.Darc, metaData *metadata.Metadata) string {
	IDHash := medChainUtils.IDToB64String(NewDarc.GetBaseID())
	metaData.BaseIdToDarcMap[IDHash] = NewDarc
	metaData.DarcIdToBaseIdMap[NewDarc.GetIdentityString()] = IDHash
	return IDHash
}

func loadKeys(configuration *conf.Configuration, metaData *metadata.Metadata) []darc.Signer {

	super_admins := []darc.Signer{}

	for _, hospital := range configuration.Hospitals {
		super_admin_signer := medChainUtils.LoadSignerEd25519(configuration.KeyDirectory+hospital.SuperAdmin.PublicKey, configuration.KeyDirectory+hospital.SuperAdmin.PrivateKey)
		hospital_metadata, super_admin_metadata := metadata.NewHospital(super_admin_signer.Identity(), hospital.Name, hospital.SuperAdmin.Name)
		metaData.Hospitals[super_admin_metadata.Id.String()] = hospital_metadata
		metaData.GenericUsers[super_admin_metadata.Id.String()] = super_admin_metadata
		super_admins = append(super_admins, super_admin_signer)

		if len(hospital.Admins) < 2 {
			panic(errors.New("All hospitals should have at least 2 admins to avoid a single point of failure"))
		}

		for _, admin := range hospital.Admins {
			admin_identity := medChainUtils.LoadIdentityEd25519(configuration.KeyDirectory + admin.PublicKey)
			admin_metadata := metadata.NewAdmin(admin_identity, admin.Name, hospital_metadata)
			metaData.GenericUsers[admin_metadata.Id.String()] = admin_metadata
		}

		for _, manager := range hospital.Managers {
			manager_identity := medChainUtils.LoadIdentityEd25519(configuration.KeyDirectory + manager.PublicKey)
			manager_metadata := metadata.NewManager(manager_identity, manager.Name, hospital_metadata)
			metaData.GenericUsers[manager_metadata.Id.String()] = manager_metadata
		}

		for _, user := range hospital.Users {
			user_identity := medChainUtils.LoadIdentityEd25519(configuration.KeyDirectory + user.PublicKey)
			user_metadata := metadata.NewUser(user_identity, user.Name, hospital_metadata)
			metaData.GenericUsers[user_metadata.Id.String()] = user_metadata
		}
	}
	return super_admins
}

func createGenesis(metaData *metadata.Metadata) {
	super_adminsIds := []darc.Identity{}
	super_adminsIDStrings := []string{}

	for IdString, hospital := range metaData.Hospitals {
		super_adminsIds = append(super_adminsIds, hospital.SuperAdmin.Id)
		super_adminsIDStrings = append(super_adminsIDStrings, IdString)
	}

	fmt.Println("Super admins", len(super_adminsIds))

	// Create Genesis block
	genesisMsg, err := service.DefaultGenesisMsg(service.CurrentVersion, roster,
		[]string{}, super_adminsIds...)
	if err != nil {
		panic(err)
	}

	gDarc := &genesisMsg.GenesisDarc
	gDarc.Rules.UpdateSign(expression.InitAndExpr(super_adminsIDStrings...))
	gDarc.Rules.AddRule("spawn:darc", gDarc.Rules.GetSignExpr())
	gDarc.Rules.AddRule("spawn:value", gDarc.Rules.GetSignExpr())

	genesisMsg.BlockInterval = time.Second
	genesisBlock, err := cl.CreateGenesisBlock(genesisMsg)
	if err != nil {
		panic(err)
	}
	metaData.GenesisBlock = genesisBlock
	metaData.GenesisMsg = genesisMsg
	metaData.GenesisDarcBaseId = addDarcToMaps(&genesisMsg.GenesisDarc, metaData)

}

func createSuperAdminsDarcs(metaData *metadata.Metadata, signers []darc.Signer) {
	// Create a DARC for admins of each hospital
	genesisDarc, ok := metaData.BaseIdToDarcMap[metaData.GenesisDarcBaseId]
	if !ok {
		panic(errors.New("Could not load the genesisDarc"))
	}
	for IdString, hospital := range metaData.Hospitals {
		darc_owners := []darc.Identity{darc.NewIdentityDarc(genesisDarc.GetID())}
		darc_signers := []darc.Identity{hospital.SuperAdmin.Id}
		rules := darc.InitRulesWith(darc_owners, darc_signers, "invoke:evolve")
		rules.AddRule("spawn:darc", rules.GetSignExpr()) // that's allright for super admins
		tempDarc, err := createDarc(cl, genesisDarc, metaData.GenesisMsg.BlockInterval, rules, "Single Super Admin darc", signers...)
		if err != nil {
			panic(err)
		}
		addDarcToMaps(tempDarc, metaData)
		hospital.SuperAdmin.DarcBaseId = addDarcToMaps(tempDarc, metaData)
		fmt.Println("add super admin darc", IdString)
	}
}

func createAllSuperAdminsDarc(metaData *metadata.Metadata, signers []darc.Signer) {
	darcIdList := []string{}

	genesisDarc, ok := metaData.BaseIdToDarcMap[metaData.GenesisDarcBaseId]
	if !ok {
		panic(errors.New("Could not load the genesis Darc"))
	}

	darc_owners := []darc.Identity{darc.NewIdentityDarc(genesisDarc.GetID())}
	for IdString, hospital := range metaData.Hospitals {

		super_admin_darc, ok := metaData.BaseIdToDarcMap[hospital.SuperAdmin.DarcBaseId]
		if !ok {
			fmt.Println("failed super admin darc", IdString)
			panic(errors.New("Could not load super admin darc"))
		}
		darcIdList = append(darcIdList, super_admin_darc.GetIdentityString())
	}
	rules := darc.InitRulesWith(darc_owners, []darc.Identity{}, "invoke:evolve")
	rules.UpdateSign(expression.InitOrExpr(darcIdList...)) // OR or AND ?
	allSuperAdminsDarc, err := createDarc(cl, genesisDarc, metaData.GenesisMsg.BlockInterval, rules,
		"All Super Admins darc", signers...)
	if err != nil {
		panic(err)
	}
	metaData.AllSuperAdminsDarcBaseId = addDarcToMaps(allSuperAdminsDarc, metaData)
}

// func createGenericUserDarcs(generic_user_list map[string][]darc.Identity, darcMap, ownerMap map[string]string, user_type string) {
// 	gDarc := &genesisMsg.GenesisDarc
// 	// Create a DARC for admins of each hospital
// 	for _, super_admin_signer := range super_admins {
//
// 		super_adminIDString := super_admin_signer.Identity().String()
// 		owner_darc, ok := getDarcFromId(super_adminIDString, baseIdToDarcMap, ownerMap)
//
// 		if !ok {
// 			fmt.Println("failed super admin darc", super_adminIDString)
// 			panic(errors.New("Could not load super admin darc"))
// 		}
//
// 		for _, user_identity := range generic_user_list[super_adminIDString] {
// 			owners := []darc.Identity{darc.NewIdentityDarc(owner_darc.GetID())}
// 			signers := []darc.Identity{user_identity}
// 			rules := darc.InitRulesWith(owners, signers, "invoke:evolve")
// 			tempDarc, err := createDarc(cl, gDarc, genesisMsg.BlockInterval, rules, "Darc for a single "+user_type, super_admins...)
// 			if err != nil {
// 				panic(err)
// 			}
// 			fmt.Println("add darc", user_identity.String())
// 			addDarcToMaps(tempDarc, user_identity.String(), darcMap)
// 		}
//
// 	}
// }

func createGenericUserDarc(user_metadata *metadata.GenericUser, owner_darc *darc.Darc, user_type string, metaData *metadata.Metadata, signers []darc.Signer) *darc.Darc {
	genesisDarc, ok := metaData.BaseIdToDarcMap[metaData.GenesisDarcBaseId]
	if !ok {
		panic(errors.New("Could not load the genesisDarc"))
	}
	darc_owners := []darc.Identity{darc.NewIdentityDarc(owner_darc.GetID())}
	darc_signers := []darc.Identity{user_metadata.Id}
	rules := darc.InitRulesWith(darc_owners, darc_signers, "invoke:evolve")
	tempDarc, err := createDarc(cl, genesisDarc, metaData.GenesisMsg.BlockInterval, rules, "Darc for a single "+user_type, signers...)
	if err != nil {
		panic(err)
	}
	fmt.Println("add "+user_type+" darc", user_metadata.Id.String())
	user_metadata.DarcBaseId = addDarcToMaps(tempDarc, metaData)
	user_metadata.IsCreated = true
	return tempDarc
}

func createAdminsDarcs(metaData *metadata.Metadata, signers []darc.Signer) {

	admins_list_darc_ids := []darc.Identity{}

	genesisDarc, ok := metaData.BaseIdToDarcMap[metaData.GenesisDarcBaseId]
	if !ok {
		panic(errors.New("Could not load the genesisDarc"))
	}

	for IdString, hospital := range metaData.Hospitals {

		owner_darc, ok := metaData.BaseIdToDarcMap[hospital.SuperAdmin.DarcBaseId]
		if !ok {
			fmt.Println("failed super admin darc", IdString)
			panic(errors.New("Could not load super admin darc"))
		}

		owner_id := []darc.Identity{darc.NewIdentityDarc(owner_darc.GetID())}

		admin_darcs_ids := []darc.Identity{}
		admin_darcs_ids_strings := []string{}
		for _, admin_metadata := range hospital.Admins {
			admin_darc := createGenericUserDarc(admin_metadata, owner_darc, "Admin", metaData, signers)
			admin_darcs_ids = append(admin_darcs_ids, darc.NewIdentityDarc(admin_darc.GetID()))
			admin_darcs_ids_strings = append(admin_darcs_ids_strings, admin_darc.GetIdentityString())
		}

		rules := darc.InitRulesWith(owner_id, admin_darcs_ids, "invoke:evolve")
		rules.AddRule("spawn:darc", medChainUtils.InitAtLeastTwoExpr(admin_darcs_ids_strings))
		adminsListDarc, err := createDarc(cl, genesisDarc, metaData.GenesisMsg.BlockInterval, rules, "List of Admin of Hospital: "+hospital.Name, signers...)
		if err != nil {
			panic(err)
		}
		hospital.AdminListDarcBaseId = addDarcToMaps(adminsListDarc, metaData)
		admins_list_darc_ids = append(admins_list_darc_ids, darc.NewIdentityDarc(adminsListDarc.GetID()))
	}
	owner_id := []darc.Identity{darc.NewIdentityDarc(genesisDarc.GetID())}
	rules := darc.InitRulesWith(owner_id, admins_list_darc_ids, "invoke:evolve")
	allAdminsDarc, err := createDarc(cl, genesisDarc, metaData.GenesisMsg.BlockInterval, rules, "All Admins darc", signers...)
	if err != nil {
		panic(err)
	}
	metaData.AllAdminsDarcBaseId = addDarcToMaps(allAdminsDarc, metaData)
}

func createManagersDarcs(metaData *metadata.Metadata, signers []darc.Signer) {

	genesisDarc, ok := metaData.BaseIdToDarcMap[metaData.GenesisDarcBaseId]
	if !ok {
		panic(errors.New("Could not load the genesisDarc"))
	}

	managers_list_darc_ids := []darc.Identity{}

	for IdString, hospital := range metaData.Hospitals {

		owner_darc, ok := metaData.BaseIdToDarcMap[hospital.AdminListDarcBaseId]

		if !ok {
			fmt.Println("failed admin list darc", IdString)
			panic(errors.New("Could not load admin list darc"))
		}

		owner_id := []darc.Identity{darc.NewIdentityDarc(owner_darc.GetID())}

		manager_darcs_ids := []darc.Identity{}
		for _, manager_metadata := range hospital.Managers {
			manager_darc := createGenericUserDarc(manager_metadata, owner_darc, "Manager", metaData, signers)
			manager_darcs_ids = append(manager_darcs_ids, darc.NewIdentityDarc(manager_darc.GetID()))
		}

		rules := darc.InitRulesWith(owner_id, manager_darcs_ids, "invoke:evolve")
		managersListDarc, err := createDarc(cl, genesisDarc, metaData.GenesisMsg.BlockInterval, rules, "List of Managers of Hospital: "+hospital.Name, signers...)
		if err != nil {
			panic(err)
		}
		hospital.ManagerListDarcBaseId = addDarcToMaps(managersListDarc, metaData)
		managers_list_darc_ids = append(managers_list_darc_ids, darc.NewIdentityDarc(managersListDarc.GetID()))
	}
	owner_id := []darc.Identity{darc.NewIdentityDarc(genesisDarc.GetID())}
	rules := darc.InitRulesWith(owner_id, managers_list_darc_ids, "invoke:evolve")
	allManagersDarc, err := createDarc(cl, genesisDarc, metaData.GenesisMsg.BlockInterval, rules, "All Managers darc", signers...)
	if err != nil {
		panic(err)
	}
	metaData.AllManagersDarcBaseId = addDarcToMaps(allManagersDarc, metaData)
}

func createUsersDarcs(metaData *metadata.Metadata, signers []darc.Signer) {

	genesisDarc, ok := metaData.BaseIdToDarcMap[metaData.GenesisDarcBaseId]
	if !ok {
		panic(errors.New("Could not load the genesisDarc"))
	}

	users_list_darc_ids := []darc.Identity{}

	for IdString, hospital := range metaData.Hospitals {

		owner_darc, ok := metaData.BaseIdToDarcMap[hospital.AdminListDarcBaseId]

		if !ok {
			fmt.Println("failed admin list darc", IdString)
			panic(errors.New("Could not load admin list darc"))
		}

		owner_id := []darc.Identity{darc.NewIdentityDarc(owner_darc.GetID())}

		user_darcs_ids := []darc.Identity{}
		for _, user_metadata := range hospital.Users {
			user_darc := createGenericUserDarc(user_metadata, owner_darc, "User", metaData, signers)
			user_darcs_ids = append(user_darcs_ids, darc.NewIdentityDarc(user_darc.GetID()))
		}

		rules := darc.InitRulesWith(owner_id, user_darcs_ids, "invoke:evolve")
		usersListDarc, err := createDarc(cl, genesisDarc, metaData.GenesisMsg.BlockInterval, rules, "List of Users of Hospital: "+hospital.Name, signers...)
		if err != nil {
			panic(err)
		}
		hospital.UserListDarcBaseId = addDarcToMaps(usersListDarc, metaData)
		users_list_darc_ids = append(users_list_darc_ids, darc.NewIdentityDarc(usersListDarc.GetID()))
		hospital.SuperAdmin.IsCreated = true
	}
	owner_id := []darc.Identity{darc.NewIdentityDarc(genesisDarc.GetID())}
	rules := darc.InitRulesWith(owner_id, users_list_darc_ids, "invoke:evolve")
	allUsersDarc, err := createDarc(cl, genesisDarc, metaData.GenesisMsg.BlockInterval, rules, "All Users darc", signers...)
	if err != nil {
		panic(err)
	}
	metaData.AllUsersDarcBaseId = addDarcToMaps(allUsersDarc, metaData)

}

// func createGenericUserListDarc(generic_user_list []*metadata.GenericUser, owner_darc *darc.Darc, rules []string, user_type string, metaData *metadata.Metadata, signers []*darc.Signer) string {
// 	// Create a DARC for admins of each hospital
// 	listOfUsersDarcsList := []string{}
// 	for _, super_admin_signer := range super_admins {
// 		listOfUsersDarc := createGenericUsersOfHospitalList(super_admin_signer, generic_user_list, darcMap, listDarcMap, ownerDarcMap, user_type)
// 		listOfUsersDarcsList = append(listOfUsersDarcsList, listOfUsersDarc.GetIdentityString())
// 	}
// 	return createGenericAllUsersDarc(listOfUsersDarcsList, rules, user_type)
// }
//
// func createGenericUserListDarc(generic_user_list []*metadata.GenericUser, owner_darc *darc.Darc, rules []string, user_type string, metaData *metadata.Metadata, signers []*darc.Signer) string {
//
// 	overall_owners := []darc.Identity{darc.NewIdentityDarc(owner_darc.GetID())}
// 	overall_signers := []string{}
// 	for _, user_metadata := range generic_user_list {
// 		user_darc, ok := metaData.BaseIdToDarcMap[user_metadata.DarcBaseId]
// 		if !ok {
// 			fmt.Println("failed "+user_type+" darc", user_metadata.Id.String())
// 			panic(errors.New("Could not load " + user_type + " darc"))
// 		}
// 		overall_signers = append(overall_signers, user_darc.GetIdentityString())
// 	}
// 	rules := darc.InitRulesWith(overall_owners, []darc.Identity{}, "invoke:evolve")
// 	rules.UpdateSign(expression.InitOrExpr(overall_signers...))
// 	usersListDarc, err := createDarc(cl, gDarc, genesisMsg.BlockInterval, rules, "List of "+user_type+" of Hospital:"+superAdminIDString, super_admins...)
// 	if err != nil {
// 		panic(err)
// 	}
// 	addDarcToMaps(usersListDarc, superAdminIDString, listDarcMap)
// 	return usersListDarc
// }
//
// func createGenericAllUsersDarc(usersListDarcsIds []string, rules_actions []string, usertype string) *darc.Darc {
// 	gDarc := &genesisMsg.GenesisDarc
// 	// Create a collective users DARC
// 	allUsersDarcOwner := []darc.Identity{darc.NewIdentityDarc(gDarc.GetID())}
// 	rules := darc.InitRulesWith(allUsersDarcOwner, []darc.Identity{}, "invoke:evolve")
// 	rules.UpdateSign(expression.InitOrExpr(usersListDarcsIds...)) // OR or AND ?
// 	for _, action := range rules_actions {
// 		rules.AddRule(darc.Action(action), rules.GetSignExpr())
// 	}
// 	allGenericUsersDarc, err := createDarc(cl, gDarc, genesisMsg.BlockInterval, rules,
// 		"All "+usertype+" darc", super_admins...)
// 	if err != nil {
// 		panic(err)
// 	}
//
// 	return allGenericUsersDarc
// }
//
// func createAdminsDarcs() {
// 	createGenericUserDarcs(admins, adminsDarcsMap, superAdminsDarcsMap, "Admin")
// }
//
// func createAdminsListDarcs() {
// 	allAdminsDarc = createGenericUserListDarcs(admins, adminsDarcsMap, adminsListDarcsMap, superAdminsDarcsMap, []string{}, "Admin")
// 	allAdminsBaseID = medChainUtils.IDToHexString(allAdminsDarc.GetBaseID())
// 	baseIdToDarcMap[allAdminsBaseID] = allAdminsDarc
// 	darcIdToBaseIdMap[allAdminsDarc.GetIdentityString()] = allAdminsBaseID
// }
//
// func createPowerfulAdminsDarcs() {
// 	gDarc := &genesisMsg.GenesisDarc
// 	// Create a DARC for admins of each hospital
// 	for _, super_admin_signer := range super_admins {
// 		super_adminIDString := super_admin_signer.Identity().String()
// 		super_admin_darc, ok := getDarcFromId(super_adminIDString, baseIdToDarcMap, superAdminsDarcsMap)
// 		if !ok {
// 			fmt.Println("failed super admin darc", super_adminIDString)
// 			panic(errors.New("Could not load super admin darc"))
// 		}
// 		overall_owners := []darc.Identity{darc.NewIdentityDarc(super_admin_darc.GetID())}
// 		overall_signers := []string{}
// 		for _, admin_identity := range admins[super_adminIDString] {
// 			admin_darc, ok := getDarcFromId(admin_identity.String(), baseIdToDarcMap, adminsDarcsMap)
// 			if !ok {
// 				fmt.Println("failed admin darc", admin_identity.String())
// 				panic(errors.New("Could not load admin darc"))
// 			}
// 			overall_signers = append(overall_signers, admin_darc.GetIdentityString())
// 		}
// 		rules := darc.InitRulesWith(overall_owners, []darc.Identity{}, "invoke:evolve")
// 		rules.UpdateSign(medChainUtils.InitAtLeastTwoExpr(overall_signers))
// 		rules.AddRule("spawn:darc", rules.GetSignExpr())
// 		rules.AddRule("spawn:value", rules.GetSignExpr())
// 		powerfulDarc, err := createDarc(cl, gDarc, genesisMsg.BlockInterval, rules, "Powerful Darc, Super Admin :"+super_adminIDString, super_admins...)
// 		if err != nil {
// 			panic(err)
// 		}
// 		addDarcToMaps(powerfulDarc, super_admin_signer.Identity().String(), powerfulDarcsMap)
// 	}
// }
//
// func createManagersDarcs() {
// 	createGenericUserDarcs(managers, managersDarcsMap, powerfulDarcsMap, "Manager")
// }
//
// func createManagersListDarcs() {
// 	allManagersDarc = createGenericUserListDarcs(managers, managersDarcsMap, managersListDarcsMap, powerfulDarcsMap, []string{}, "Manager")
// 	allManagersBaseID = medChainUtils.IDToHexString(allManagersDarc.GetBaseID())
// 	baseIdToDarcMap[allManagersBaseID] = allManagersDarc
// 	darcIdToBaseIdMap[allManagersDarc.GetIdentityString()] = allManagersBaseID
// }
//
// func createUsersDarcs() {
// 	createGenericUserDarcs(users, usersDarcsMap, powerfulDarcsMap, "User")
// }
//
// func createUsersListDarcs() {
// 	rules := []string{"spawn:ProjectList"}
// 	allUsersDarc = createGenericUserListDarcs(users, usersDarcsMap, usersListDarcsMap, powerfulDarcsMap, rules, "User")
// 	allUsersBaseID = medChainUtils.IDToHexString(allUsersDarc.GetBaseID())
// 	baseIdToDarcMap[allUsersBaseID] = allUsersDarc
// 	darcIdToBaseIdMap[allUsersDarc.GetIdentityString()] = allUsersBaseID
// }

// func createProjectDarcs(configuration *conf.Configuration, metaData *metadata.Metadata) {
//
// 	gDarc := &genesisMsg.GenesisDarc
//
// 	var allProjectsListInstanceID service.InstanceID
//
// 	list_of_projects := []string{}
//
// 	// super_admin_index := 0
//
// 	for _, project := range configuration.Projects {
//
// 		owners := []darc.Identity{}
// 		for _, managerCoordinates := range project.ManagerOwners {
// 			// super_admin_index = managerCoordinates.I
// 			manager_identity := findManager(managerCoordinates)
// 			manager_darc, ok := getDarcFromId(manager_identity.String(), baseIdToDarcMap, managersDarcsMap)
// 			if !ok {
// 				fmt.Println("failed manager darc", manager_identity.String())
// 				panic(errors.New("Could not load manager darc"))
// 			}
// 			id := darc.NewIdentityDarc(manager_darc.GetID())
// 			owners = append(owners, id)
// 		}
//
// 		signers := []darc.Identity{}
// 		for _, userCoordinates := range project.SigningUsers {
// 			user_identity := findUser(userCoordinates)
// 			user_darc, ok := getDarcFromId(user_identity.String(), baseIdToDarcMap, usersDarcsMap)
// 			if !ok {
// 				fmt.Println("failed user darc", user_identity.String())
// 				panic(errors.New("Could not load user darc"))
// 			}
// 			id := darc.NewIdentityDarc(user_darc.GetID())
// 			signers = append(owners, id)
// 		}
//
// 		projectDarcRules := darc.InitRulesWith(owners, signers, "invoke:evolve")
// 		for _, rule := range project.Rules {
// 			usersIdString := []string{}
// 			for _, userCoordinates := range rule.Users {
// 				user_identity := findUser(userCoordinates)
// 				user_darc, ok := getDarcFromId(user_identity.String(), baseIdToDarcMap, usersDarcsMap)
// 				if !ok {
// 					fmt.Println("failed user darc", user_identity.String())
// 					panic(errors.New("Could not load user darc"))
// 				}
// 				idString := user_darc.GetIdentityString()
// 				usersIdString = append(usersIdString, idString)
// 			}
// 			var expr expression.Expr
// 			switch rule.ExprType {
// 			case "SIGNERS":
// 				expr = projectDarcRules.GetSignExpr()
// 			case "OR":
// 				expr = expression.InitOrExpr(usersIdString...)
// 			case "AND":
// 				expr = expression.InitAndExpr(usersIdString...)
// 			}
// 			projectDarcRules.AddRule(darc.Action(rule.Action), expr)
// 		}
//
// 		projectDarc, err := createDarc(cl, gDarc, genesisMsg.BlockInterval, projectDarcRules,
// 			project.Name, super_admins...)
// 		if err != nil {
// 			panic(err)
// 		}
//
// 		addDarcToMaps(projectDarc, project.Name, projectsDarcsMap)
//
// 		list_of_projects = append(list_of_projects, projectDarc.GetIdentityString())
// 	}
//
// 	// super_admin := super_admins[super_admin_index]
// 	// powerful_darc, ok := getDarcFromId(super_admin.Identity().String(), baseIdToDarcMap, powerfulDarcsMap)
//
// 	// Register the sample project DARC with the value contract
// 	myvalue := []byte(strings.Join(list_of_projects, ";"))
// 	ctx := service.ClientTransaction{
// 		Instructions: []service.Instruction{{
// 			InstanceID: service.NewInstanceID(allManagersDarc.GetBaseID()),
// 			Nonce:      service.Nonce{},
// 			Index:      0,
// 			Length:     1,
// 			Spawn: &service.Spawn{
// 				ContractID: contracts.ContractValueID,
// 				Args: []service.Argument{{
// 					Name:  "value",
// 					Value: myvalue,
// 				}},
// 			},
// 		}},
// 	}
// 	err = ctx.Instructions[0].SignBy(gDarc.GetBaseID(), super_admins...)
// 	if err != nil {
// 		panic(err)
// 	}
//
// 	_, err = cl.AddTransaction(ctx)
// 	if err != nil {
// 		panic(err)
// 	}
//
// 	allProjectsListInstanceID = service.NewInstanceID(ctx.Instructions[0].Hash())
// 	pr, err := cl.WaitProof(allProjectsListInstanceID, genesisMsg.BlockInterval, nil)
// 	if pr.InclusionProof.Match() != true {
// 		panic(err)
// 	}
// }

func startSystem(metaData *metadata.Metadata) {
	configuration, err := conf.ReadConf(configFileName)
	if err != nil {
		panic(err)
	}

	fmt.Println(len(configuration.Hospitals))
	for _, hosp := range configuration.Hospitals {
		fmt.Println(len(hosp.Admins))
		fmt.Println(len(hosp.Managers))
		fmt.Println(len(hosp.Users))
	}
	// We need to load suitable keys to initialize the system DARCs as per our context

	signers := loadKeys(configuration, metaData)

	createGenesis(metaData)

	createSuperAdminsDarcs(metaData, signers)

	createAllSuperAdminsDarc(metaData, signers)

	createAdminsDarcs(metaData, signers)

	// createAdminsListDarcs()

	// createPowerfulAdminsDarcs()

	createManagersDarcs(metaData, signers)

	// createManagersListDarcs()

	createUsersDarcs(metaData, signers)

	// createUsersListDarcs(metaData, signers)

	// createProjectDarcs(configuration)

}
