package main

import (
	"errors"
	"fmt"
	"time"

	"github.com/DPPH/MedChain/medChainServer/conf"
	"github.com/DPPH/MedChain/medChainServer/metadata"
	"github.com/DPPH/MedChain/medChainUtils"
	"github.com/dedis/cothority/omniledger/darc"
	"github.com/dedis/cothority/omniledger/darc/expression"
	"github.com/dedis/cothority/omniledger/service"
)

/**
This file takes care of the bootstraping process for the hospital and the user darcs
**/

/**
Helper function to keep the created darcs in the metadata
**/
func addDarcToMaps(NewDarc *darc.Darc, metaData *metadata.Metadata) string {
	IDHash := medChainUtils.IDToB64String(NewDarc.GetBaseID())
	metaData.BaseIdToDarcMap[IDHash] = NewDarc
	metaData.DarcIdToBaseIdMap[NewDarc.GetIdentityString()] = IDHash
	return IDHash
}

/**
Read the keys from the configuration files.
Initialize the metadata for each users and each hospitals.
**/
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

/**
Creates the genesis darc.
Stores it in the metadata.
**/
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

	genesisMsg.BlockInterval = time.Second
	genesisBlock, err := cl.CreateGenesisBlock(genesisMsg)
	if err != nil {
		panic(err)
	}
	metaData.GenesisBlock = genesisBlock
	metaData.GenesisMsg = genesisMsg
	metaData.GenesisDarcBaseId = addDarcToMaps(&genesisMsg.GenesisDarc, metaData)
}

/**
Creates an individual darc for each super administrator (one per hospital).
**/
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
		tempDarc, err := medChainUtils.CreateDarc(cl, genesisDarc, metaData.GenesisMsg.BlockInterval, rules, "Single Super Admin darc", signers...)
		if err != nil {
			panic(err)
		}
		addDarcToMaps(tempDarc, metaData)
		hospital.SuperAdmin.DarcBaseId = addDarcToMaps(tempDarc, metaData)
		fmt.Println("add super admin darc", IdString)
	}
}

/**
Creates the allSuperAdminsDarc that has all the super admins as signers
**/
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
	allSuperAdminsDarc, err := medChainUtils.CreateDarc(cl, genesisDarc, metaData.GenesisMsg.BlockInterval, rules,
		"All Super Admins darc", signers...)
	if err != nil {
		panic(err)
	}
	metaData.AllSuperAdminsDarcBaseId = addDarcToMaps(allSuperAdminsDarc, metaData)
}

/**
An helper function that creates a darc for a single generic user
user_metadata contains the initialized metadata of the generc user,
owner_darc is the darc that will have the "invoke:evolve" rule for the darc,
user_type can be either "Admin", "Manager" or "User"
metaData is the common metadata object,
signers is the list of super_admin *darc.Signer objects, used to spawn the darc with the genesis darc
**/
func createGenericUserDarc(user_metadata *metadata.GenericUser, owner_darc *darc.Darc, user_type string, metaData *metadata.Metadata, signers []darc.Signer) *darc.Darc {
	genesisDarc, ok := metaData.BaseIdToDarcMap[metaData.GenesisDarcBaseId]
	if !ok {
		panic(errors.New("Could not load the genesisDarc"))
	}
	darc_owners := []darc.Identity{darc.NewIdentityDarc(owner_darc.GetID())}
	darc_signers := []darc.Identity{user_metadata.Id}
	rules := darc.InitRulesWith(darc_owners, darc_signers, "invoke:evolve")
	tempDarc, err := medChainUtils.CreateDarc(cl, genesisDarc, metaData.GenesisMsg.BlockInterval, rules, "Darc for a single "+user_type, signers...)
	if err != nil {
		panic(err)
	}
	fmt.Println("add "+user_type+" darc", user_metadata.Id.String())
	user_metadata.DarcBaseId = addDarcToMaps(tempDarc, metaData)
	user_metadata.IsCreated = true
	return tempDarc
}

/**
Creates a darc for each administrator
Creates a AdminListDarc for each hospital,
	that has all of the single admins darcs of the hospital as signers,
	can be used to spawn darcs
	and can be used to update the project list, via the ProjectCreatorDarc
Creates an AllAdminsDarc that has all of the AdminListDarc as signers
metaData is the common metadata object,
signers is the list of super_admin *darc.Signer objects, used to spawn the darc with the genesis darc
**/
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
		adminsListDarc, err := medChainUtils.CreateDarc(cl, genesisDarc, metaData.GenesisMsg.BlockInterval, rules, "List of Admin of Hospital: "+hospital.Name, signers...)
		if err != nil {
			panic(err)
		}
		hospital.AdminListDarcBaseId = addDarcToMaps(adminsListDarc, metaData)
		admins_list_darc_ids = append(admins_list_darc_ids, darc.NewIdentityDarc(adminsListDarc.GetID()))
	}
	owner_id := []darc.Identity{darc.NewIdentityDarc(genesisDarc.GetID())}
	rules := darc.InitRulesWith(owner_id, admins_list_darc_ids, "invoke:evolve")
	rules.AddRule("spawn:value", rules.GetSignExpr())
	rules.AddRule("invoke:update", rules.GetSignExpr())
	allAdminsDarc, err := medChainUtils.CreateDarc(cl, genesisDarc, metaData.GenesisMsg.BlockInterval, rules, "All Admins darc", signers...)
	if err != nil {
		panic(err)
	}
	metaData.AllAdminsDarcBaseId = addDarcToMaps(allAdminsDarc, metaData)
}

/**
Creates a darc for each manager
Creates a ManagerListDarc for each hospital,
	that has all of the single managers darcs of the hospital as signers,
	and can be used to update the project list, via the ProjectCreatorDarc
Creates an AllManagerssDarc that has all of the ManagerListDarc as signers
metaData is the common metadata object,
signers is the list of super_admin *darc.Signer objects, used to spawn the darc with the genesis darc
**/
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
		managersListDarc, err := medChainUtils.CreateDarc(cl, genesisDarc, metaData.GenesisMsg.BlockInterval, rules, "List of Managers of Hospital: "+hospital.Name, signers...)
		if err != nil {
			panic(err)
		}
		hospital.ManagerListDarcBaseId = addDarcToMaps(managersListDarc, metaData)
		managers_list_darc_ids = append(managers_list_darc_ids, darc.NewIdentityDarc(managersListDarc.GetID()))
	}
	owner_id := []darc.Identity{darc.NewIdentityDarc(genesisDarc.GetID())}
	rules := darc.InitRulesWith(owner_id, managers_list_darc_ids, "invoke:evolve")
	rules.AddRule("spawn:UserProjectsMap", rules.GetSignExpr())
	rules.AddRule("invoke:update", rules.GetSignExpr())
	allManagersDarc, err := medChainUtils.CreateDarc(cl, genesisDarc, metaData.GenesisMsg.BlockInterval, rules, "All Managers darc", signers...)
	if err != nil {
		panic(err)
	}
	metaData.AllManagersDarcBaseId = addDarcToMaps(allManagersDarc, metaData)
}

/**
Creates a darc for each user
Creates a UserListDarc for each hospital, that has all of the single users darcs of the hospital as signers,
Creates an AllUsersDarc that has all of the UserListDarcs as signers
 	and can be used to spawn the ProjectList contract
metaData is the common metadata object,
signers is the list of super_admin *darc.Signer objects, used to spawn the darc with the genesis darc
**/
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
		rules.AddRule("spawn:ProjectList", rules.GetSignExpr())
		usersListDarc, err := medChainUtils.CreateDarc(cl, genesisDarc, metaData.GenesisMsg.BlockInterval, rules, "List of Users of Hospital: "+hospital.Name, signers...)
		if err != nil {
			panic(err)
		}
		hospital.UserListDarcBaseId = addDarcToMaps(usersListDarc, metaData)
		users_list_darc_ids = append(users_list_darc_ids, darc.NewIdentityDarc(usersListDarc.GetID()))
		hospital.SuperAdmin.IsCreated = true
	}
	owner_id := []darc.Identity{darc.NewIdentityDarc(genesisDarc.GetID())}
	rules := darc.InitRulesWith(owner_id, users_list_darc_ids, "invoke:evolve")
	allUsersDarc, err := medChainUtils.CreateDarc(cl, genesisDarc, metaData.GenesisMsg.BlockInterval, rules, "All Users darc", signers...)
	if err != nil {
		panic(err)
	}
	metaData.AllUsersDarcBaseId = addDarcToMaps(allUsersDarc, metaData)

}

/**
This function loads the configuration from the configuration file
and starts the bootstrapping process
**/
func startSystem(metaData *metadata.Metadata) {
	fmt.Println("####### Starting Bootstraping #######")
	configuration, err := conf.ReadConf(configFileName)
	if err != nil {
		panic(err)
	}

	signers := loadKeys(configuration, metaData)

	createGenesis(metaData)

	createSuperAdminsDarcs(metaData, signers)

	createAllSuperAdminsDarc(metaData, signers)

	createAdminsDarcs(metaData, signers)

	createManagersDarcs(metaData, signers)

	createUsersDarcs(metaData, signers)

	createProjectCreatorDarc(metaData, signers)

	createProjectDarcs(configuration, metaData, signers)

	fmt.Println("####### Finished Bootstraping #######")

}
