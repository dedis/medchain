package contracts

import (
	"testing"
	"time"

	"github.com/dedis/cothority"
	"github.com/dedis/cothority/omniledger/darc"
	"github.com/dedis/cothority/omniledger/darc/expression"
	"github.com/dedis/cothority/omniledger/service"
	"github.com/dedis/onet"
	"github.com/stretchr/testify/require"
	"strings"
	"encoding/hex"
)

// Need to adjust based on the complexity of contracts
var waitFactor = 1

func Test_System_Rigorous(t *testing.T) {
	// Complexity
	numberOfHospitals := 3
	adminsPerHospital := 1
	managersPerHospital := 1
	usersPerHospital := 5
	serversPerHospital := 1
	numberOfProjects := 2

	// Virtual servers
	local := onet.NewTCPTest(cothority.Suite)
	defer local.CloseAll()
	_, roster, _ := local.GenTree(numberOfHospitals * serversPerHospital, true)
	cl := service.NewClient()

	// Admins
	admins := []darc.Signer{}
	for i := 0; i < numberOfHospitals * adminsPerHospital; i++ {
		admins = append(admins, darc.NewSignerEd25519(nil, nil))
	}

	// Managers
	managers := []darc.Signer{}
	for i := 0; i < numberOfHospitals * managersPerHospital; i++ {
		managers = append(managers, darc.NewSignerEd25519(nil, nil))
	}

	// Users
	users := []darc.Signer{}
	for i := 0; i < numberOfHospitals * usersPerHospital; i++ {
		users = append(users, darc.NewSignerEd25519(nil, nil))
	}

	// Create genesis block
	allAdminIdentities := ExtractIdentities(numberOfHospitals * adminsPerHospital, admins)
	allAdminStrings := ExtractIdentityStrings(numberOfHospitals * adminsPerHospital, admins)

	genesisMsg, err := service.DefaultGenesisMsg(service.CurrentVersion, roster,
		[]string{}, allAdminIdentities...)
	require.Nil(t, err)

	gDarc := &genesisMsg.GenesisDarc
	gDarc.Rules.UpdateSign(expression.InitAndExpr(allAdminStrings...))
	gDarc.Rules.AddRule("spawn:darc", gDarc.Rules.GetSignExpr())

	genesisMsg.BlockInterval = time.Second
	_, err = cl.CreateGenesisBlock(genesisMsg)
	require.Nil(t, err)

	// Create a DARC for managers of each hospital
	managersDarcs := []*darc.Darc{}
	for i := 0; i < numberOfHospitals; i++ {
		managersOfHospital := managers[i*managersPerHospital:(i+1)*managersPerHospital]
		adminsOfHospital := admins[i*adminsPerHospital:(i+1)*adminsPerHospital]

		rules := darc.InitRules(ExtractIdentities(adminsPerHospital, adminsOfHospital),
			ExtractIdentities(managersPerHospital, managersOfHospital))
		tempDarc, err := createDarc(cl, gDarc, genesisMsg.BlockInterval, rules,
			"Managers darc", admins...)
		require.Nil(t, err)
		managersDarcs = append(managersDarcs, tempDarc)
	}

	// Create a collective managers DARC
	// allManagerIdentities := extractIdentities(numberOfHospitals * managersPerHospital, managers)
	// allManagerStrings := extractIdentityStrings(numberOfHospitals * managersPerHospital, managers)
	tmp := []string{}
	for i := 0; i < len(managersDarcs); i++ {
		tmp = append(tmp, managersDarcs[i].GetIdentityString())
	}
	rules := darc.InitRules(allAdminIdentities, []darc.Identity{})
	rules.UpdateSign(expression.InitAndExpr(tmp...))
	rules.AddRule("spawn:darc", rules.GetSignExpr())
	rules.AddRule("spawn:value", rules.GetSignExpr())
	rules.AddRule("spawn:UserProjectsMap", expression.InitOrExpr(tmp...))
	rules.AddRule("invoke:update", rules["spawn:UserProjectsMap"])
	allManagersDarc, err := createDarc(cl, gDarc, genesisMsg.BlockInterval, rules,
		"AllManagers darc", admins...)
	require.Nil(t, err)


	// Create a DARC for users of each hospital
	usersDarcs := []*darc.Darc{}
	for i := 0; i < numberOfHospitals; i++ {
		usersOfHospital := users[i*usersPerHospital:(i+1)*usersPerHospital]

		rules := darc.InitRules([]darc.Identity{darc.NewIdentityDarc(managersDarcs[i].GetID())},
			ExtractIdentities(usersPerHospital, usersOfHospital))
		tempDarc, err := createDarc(cl, allManagersDarc, genesisMsg.BlockInterval, rules,
			"Users darc", managers...)
		require.Nil(t, err)
		usersDarcs = append(usersDarcs, tempDarc)
	}

	// Create a collective users DARC
	// allUserIdentities := extractIdentities(numberOfHospitals * usersPerHospital, users)
	// allUserStrings := extractIdentityStrings(numberOfHospitals * usersPerHospital, users)
	tmp2 := []darc.Identity{}
	for i := 0; i < len(usersDarcs); i++ {
		tmp2 = append(tmp2, darc.NewIdentityDarc(usersDarcs[i].GetID()))
	}
	rules = darc.InitRules([]darc.Identity{darc.NewIdentityDarc(allManagersDarc.GetID())},
		tmp2)
	rules.AddRule("spawn:ProjectListSlow", rules.GetSignExpr())
	rules.AddRule("spawn:ProjectList", rules.GetSignExpr())
	allUsersDarc, err := createDarc(cl, allManagersDarc, genesisMsg.BlockInterval, rules,
		"AllUsers darc", managers...)
	require.Nil(t, err)

	// Create projects with all hospitals collaborating in each one
	projects := []*darc.Darc{}
	projectsIdentityString := ""
	tmp3 := []darc.Identity{}
	for i := 0; i < len(managersDarcs); i++ {
		tmp3 = append(tmp3, darc.NewIdentityDarc(managersDarcs[i].GetID()))
	}
	for q := 0; q < numberOfProjects; q++ {
		projectXDarcRules := darc.InitRules(tmp3, tmp2)
		// Define access control rules for the project DARC
		projectXDarcRules.AddRule("spawn:AuthGrant", projectXDarcRules.GetSignExpr())
		projectXDarcRules.AddRule("spawn:CreateQuery", projectXDarcRules.GetSignExpr())
		projectXDarcRules.AddRule(darc.Action("spawn:"+QueryTypes[0]), projectXDarcRules.GetSignExpr())
		projectXDarcRules.AddRule(darc.Action("spawn:"+QueryTypes[1]), projectXDarcRules.GetSignExpr())
		projectXDarc, err := createDarc(cl, allManagersDarc, genesisMsg.BlockInterval, projectXDarcRules,
			"Project" + string(q), managers...)
		require.Nil(t, err)

		projectsIdentityString += projectXDarc.GetIdentityString() + ";"
		projects = append(projects, projectXDarc)
	}

	// Register all project DARCs with the value contract
	ctx := service.ClientTransaction{
		Instructions: []service.Instruction{{
			InstanceID: service.NewInstanceID(allManagersDarc.GetBaseID()),
			Nonce:  service.Nonce{},
			Index:  0,
			Length: 1,
			Spawn: &service.Spawn{
				ContractID: ContractValueID,
				Args: []service.Argument{{
					Name:  "value",
					Value: []byte(projectsIdentityString),
				}},
			},
		}},
	}
	require.Nil(t, ctx.Instructions[0].SignBy(allManagersDarc.GetBaseID(), managers...))
	_, err = cl.AddTransaction(ctx)
	require.Nil(t, err)
	allProjectsListInstanceID := service.NewInstanceID(ctx.Instructions[0].Hash())
	pr, err := cl.WaitProof(allProjectsListInstanceID, genesisMsg.BlockInterval * time.Duration(waitFactor), nil)
	require.True(t, pr.InclusionProof.Match())

		// Create a users-projects map contract instance
		ctx = service.ClientTransaction{
			Instructions: []service.Instruction{{
				InstanceID: service.NewInstanceID(allManagersDarc.GetBaseID()),
				Nonce:  service.Nonce{},
				Index:  0,
				Length: 1,
				Spawn: &service.Spawn{
					ContractID: ContractUserProjectsMapID,
					Args: []service.Argument{{
						Name:  "allProjectsListInstanceID",
						Value: []byte(allProjectsListInstanceID.Slice()),
					}, {
						Name:  "users",
						Value: []byte(""),
					}},
				},
			}},
		}
		require.Nil(t, ctx.Instructions[0].SignBy(allManagersDarc.GetBaseID(), managers...))
		_, err = cl.AddTransaction(ctx)
		require.Nil(t, err)
	    userProjectsMapInstanceID := service.NewInstanceID(ctx.Instructions[0].Hash())
		pr, err = cl.WaitProof(userProjectsMapInstanceID, genesisMsg.BlockInterval * time.Duration(waitFactor) , nil)
		require.True(t, pr.InclusionProof.Match())
		values, err := pr.InclusionProof.RawValues()

		// Update the users-projects map contract instance for users of interest
		usersToBeUpdatedInMap := ""
		for i := 0; i < 1; i++ {
			usersToBeUpdatedInMap += users[i].Identity().String() + ";"
			ctx = service.ClientTransaction{
				Instructions: []service.Instruction{{
					InstanceID: userProjectsMapInstanceID,
					Nonce:  service.Nonce{},
					Index:  0,
					Length: 1,
					Invoke: &service.Invoke{
						Command: "update",
						Args: []service.Argument{{
							Name:  "allProjectsListInstanceID",
							Value: []byte(allProjectsListInstanceID.Slice()),
						}, {
							Name:  "users",
							Value: []byte(usersToBeUpdatedInMap),
						}},
					},
				}},
			}

			// Any manager can authorize an update of the user-projects map
			require.Nil(t, ctx.Instructions[0].SignBy(allManagersDarc.GetBaseID(), managers[0]))
			_, err = cl.AddTransaction(ctx)
			require.Nil(t, err)

			// Hack to proceed only when the update goes through
			for {
				pr, err = cl.WaitProof(userProjectsMapInstanceID, genesisMsg.BlockInterval * time.Duration(waitFactor) , nil)
				require.True(t, pr.InclusionProof.Match())
				tmp, _ := pr.InclusionProof.RawValues()
				if string(values[0][:]) == string(tmp[0][:]) {
					time.Sleep(1 * time.Second)
				} else {
					println(string(tmp[0][:]))
					break
				}
			}
		}

	// Demo1/rigorous ///////////////////////////////////////////////
	// Get the list of all projects/actions a user is associated with
		start := time.Now()
	ctx = service.ClientTransaction{
		Instructions: []service.Instruction{{
			InstanceID: service.NewInstanceID(allUsersDarc.GetBaseID()),
			Nonce:  service.Nonce{},
			Index:  0,
			Length: 1,
			Spawn: &service.Spawn{
				ContractID: ContractProjectListID,
				Args: []service.Argument{{
					Name:  "userProjectsMapInstanceID",
					Value: []byte(userProjectsMapInstanceID.Slice()),
				}},
			},
		}},
	}
	require.Nil(t, ctx.Instructions[0].SignBy(allUsersDarc.GetBaseID(), users[0]))
	_, err = cl.AddTransaction(ctx)
	require.Nil(t, err)
	instID := service.NewInstanceID(ctx.Instructions[0].Hash())
	pr, err = cl.WaitProof(instID, genesisMsg.BlockInterval * time.Duration(waitFactor), nil)
	for x := 0; x < 1000; x++ {
		if err != nil || pr.InclusionProof.Match() != true {
			pr, err = cl.WaitProof(instID, genesisMsg.BlockInterval * time.Duration(waitFactor), nil)
		} else {
			break
		}
	}
	require.True(t, pr.InclusionProof.Match())
	values, err = pr.InclusionProof.RawValues()
	require.Nil(t, err)
		elapsed := time.Since(start)
		println("Demo1 took " + elapsed.String())
		println(string(values[0][:]))
	authorizedProjectDarcID := strings.Split(string(values[0][:]), "......")[1]
	authorizedProjectDarcID = strings.Split(authorizedProjectDarcID, "...")[1]
	authorizedProjectDarcIDx, _ := hex.DecodeString(authorizedProjectDarcID[5:])

	//Demo2/rigorous /////////////////////////////////////////////
	//Create a query with a particular type for a particular user
		start = time.Now()
	ctx = service.ClientTransaction{
		Instructions: []service.Instruction{{
			InstanceID: service.NewInstanceID(authorizedProjectDarcIDx),
			Nonce:  service.Nonce{},
			Index:  0,
			Length: 1,
			Spawn: &service.Spawn{
				ContractID: ContractCreateQueryID,
				Args: []service.Argument{{
					Name:  "queryType",
					Value: []byte("AggregatedQuery"),
				}, {
					Name:  "query",
					Value: []byte("<bla bla>"),
				}},
			},
		}},
	}
	require.Nil(t, ctx.Instructions[0].SignBy(authorizedProjectDarcIDx, users[0]))

	_, err = cl.AddTransaction(ctx)
	require.Nil(t, err)
	instID = service.NewInstanceID(ctx.Instructions[0].Hash())
	pr, err = cl.WaitProof(instID, genesisMsg.BlockInterval, nil)
	for x := 0; x < 1000; x++ {
		if err != nil || pr.InclusionProof.Match() != true {
			pr, err = cl.WaitProof(instID, genesisMsg.BlockInterval * time.Duration(waitFactor), nil)
		} else {
			break
		}
	}
	require.True(t, pr.InclusionProof.Match())
	values, err = pr.InclusionProof.RawValues()
	require.Nil(t, err)
		elapsed = time.Since(start)
		println("Demo2 took " + elapsed.String())
		println(string(values[0][:]))

	// Wrap things up
	local.WaitDone(genesisMsg.BlockInterval)
}

func ExtractIdentities(x int, signerSlice []darc.Signer) []darc.Identity {
	allIdentities := []darc.Identity{}
	for i := 0; i < x; i++ {
		allIdentities = append(allIdentities, signerSlice[i].Identity())
	}
	return allIdentities
}

func ExtractIdentityStrings(x int, signerSlice []darc.Signer) []string {
	allStrings := []string{}
	for i := 0; i < x; i++ {
		allStrings = append(allStrings, signerSlice[i].Identity().String())
	}
	return allStrings
}

func Test_System(t *testing.T) {
	local := onet.NewTCPTest(cothority.Suite)
	defer local.CloseAll()
	_, roster, _ := local.GenTree(3, true)
	cl := service.NewClient()

	// Admins, Managers and Users as per our context
	admins := []darc.Signer{darc.NewSignerEd25519(nil, nil), darc.NewSignerEd25519(nil, nil),
		darc.NewSignerEd25519(nil, nil)}
	managers := []darc.Signer{darc.NewSignerEd25519(nil, nil), darc.NewSignerEd25519(nil, nil),
		darc.NewSignerEd25519(nil, nil)}
	users := []darc.Signer{darc.NewSignerEd25519(nil, nil), darc.NewSignerEd25519(nil, nil),
		darc.NewSignerEd25519(nil, nil)}

	// Create Genesis block
	genesisMsg, err := service.DefaultGenesisMsg(service.CurrentVersion, roster,
		[]string{}, admins[0].Identity(), admins[1].Identity(), admins[2].Identity())
	require.Nil(t, err)
	gDarc := &genesisMsg.GenesisDarc
	gDarc.Rules.UpdateSign(expression.InitAndExpr(admins[0].Identity().String(),
		admins[1].Identity().String(), admins[2].Identity().String()))
	gDarc.Rules.AddRule("spawn:darc", gDarc.Rules.GetSignExpr())

	genesisMsg.BlockInterval = time.Second
	_, err = cl.CreateGenesisBlock(genesisMsg)
	require.Nil(t, err)

	// Create a DARC for managers of each hospital
	managersDarcs := []*darc.Darc{}
	for i := 0; i < len(managers); i++ {
		rules := darc.InitRules([]darc.Identity{admins[i].Identity()},
			[]darc.Identity{managers[i].Identity()})
		tempDarc, err := createDarc(cl, gDarc, genesisMsg.BlockInterval, rules,
			"Managers darc", admins...)
		require.Nil(t, err)
		managersDarcs = append(managersDarcs, tempDarc)
	}

	// Create a collective managers DARC
	rules := darc.InitRules([]darc.Identity{admins[0].Identity(), admins[1].Identity(),
		admins[2].Identity()}, []darc.Identity{})
	rules.UpdateSign(expression.InitAndExpr(managersDarcs[0].GetIdentityString(),
		managersDarcs[1].GetIdentityString(), managersDarcs[2].GetIdentityString()))
	rules.AddRule("spawn:darc", rules.GetSignExpr())
	rules.AddRule("spawn:value", rules.GetSignExpr())
	rules.AddRule("spawn:UserProjectsMap", expression.InitOrExpr(managersDarcs[0].GetIdentityString(),
		managersDarcs[1].GetIdentityString(), managersDarcs[2].GetIdentityString()))
	rules.AddRule("invoke:update", rules["spawn:UserProjectsMap"])
	allManagersDarc, err := createDarc(cl, gDarc, genesisMsg.BlockInterval, rules,
		"AllManagers darc", admins...)
	require.Nil(t, err)

	// Create a DARC for users of each hospital
	usersDarcs := []*darc.Darc{}
	for i := 0; i < len(users); i++ {
		rules := darc.InitRules([]darc.Identity{darc.NewIdentityDarc(managersDarcs[i].GetID())},
			[]darc.Identity{users[i].Identity()})
		tempDarc, err := createDarc(cl, allManagersDarc, genesisMsg.BlockInterval, rules,
			"Users darc", managers...)
		require.Nil(t, err)
		usersDarcs = append(usersDarcs, tempDarc)
	}

	// Create a collective users DARC
	rules = darc.InitRules([]darc.Identity{darc.NewIdentityDarc(allManagersDarc.GetID())},
		[]darc.Identity{darc.NewIdentityDarc(usersDarcs[0].GetID()), darc.NewIdentityDarc(usersDarcs[1].GetID()),
			darc.NewIdentityDarc(usersDarcs[2].GetID())})
	rules.AddRule("spawn:ProjectList", rules.GetSignExpr())
	allUsersDarc, err := createDarc(cl, allManagersDarc, genesisMsg.BlockInterval, rules,
		"AllUsers darc", managers...)
	require.Nil(t, err)

	// Create a sample project DARC
	projectXDarcRules := darc.InitRules([]darc.Identity{darc.NewIdentityDarc(managersDarcs[0].GetID()),
		darc.NewIdentityDarc(managersDarcs[2].GetID())}, []darc.Identity{darc.NewIdentityDarc(usersDarcs[0].GetID()),
		darc.NewIdentityDarc(usersDarcs[2].GetID())})
	// Define access control rules for the project DARC
	projectXDarcRules.AddRule("spawn:AuthGrant", projectXDarcRules.GetSignExpr())
	projectXDarcRules.AddRule("spawn:CreateQuery", projectXDarcRules.GetSignExpr())
	projectXDarcRules.AddRule(darc.Action("spawn:"+QueryTypes[0]), projectXDarcRules.GetSignExpr())
	projectXDarcRules.AddRule(darc.Action("spawn:"+QueryTypes[1]), expression.InitOrExpr(usersDarcs[0].GetIdentityString()))
	projectXDarc, err := createDarc(cl, allManagersDarc, genesisMsg.BlockInterval, projectXDarcRules,
		"ProjectX", managers...)
	require.Nil(t, err)

		// Register the sample project DARC with the value contract
		myvalue := []byte(projectXDarc.GetIdentityString())
		ctx := service.ClientTransaction{
			Instructions: []service.Instruction{{
				InstanceID: service.NewInstanceID(allManagersDarc.GetBaseID()),
				Nonce:  service.Nonce{},
				Index:  0,
				Length: 1,
				Spawn: &service.Spawn{
					ContractID: ContractValueID,
					Args: []service.Argument{{
						Name:  "value",
						Value: myvalue,
					}},
				},
			}},
		}
		require.Nil(t, ctx.Instructions[0].SignBy(allManagersDarc.GetBaseID(), managers[0], managers[1], managers[2]))
		_, err = cl.AddTransaction(ctx)
		require.Nil(t, err)
		allProjectsListInstanceID := service.NewInstanceID(ctx.Instructions[0].Hash())
		pr, err := cl.WaitProof(allProjectsListInstanceID, genesisMsg.BlockInterval, nil)
		require.True(t, pr.InclusionProof.Match())

		// Create a users-projects map contract instance
		usersByte := []byte(users[2].Identity().String() + ";" + users[0].Identity().String())
		ctx = service.ClientTransaction{
			Instructions: []service.Instruction{{
				InstanceID: service.NewInstanceID(allManagersDarc.GetBaseID()),
				Nonce:  service.Nonce{},
				Index:  0,
				Length: 1,
				Spawn: &service.Spawn{
					ContractID: ContractUserProjectsMapID,
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
		require.Nil(t, ctx.Instructions[0].SignBy(allManagersDarc.GetBaseID(), managers[0], managers[2]))
		_, err = cl.AddTransaction(ctx)
		require.Nil(t, err)
		userProjectsMapInstanceID := service.NewInstanceID(ctx.Instructions[0].Hash())

		pr, err = cl.WaitProof(userProjectsMapInstanceID, genesisMsg.BlockInterval, nil)
		require.True(t, pr.InclusionProof.Match())

		// Try updating users-projects map contract instance
		//usersByte = []byte(users[0].Identity().String())
		//ctx = service.ClientTransaction{
		//	Instructions: []service.Instruction{{
		//		InstanceID: userProjectsMapInstanceID,
		//		Nonce:  service.Nonce{},
		//		Index:  0,
		//		Length: 1,
		//		Invoke: &service.Invoke{
		//			Command: "update",
		//			Args: []service.Argument{{
		//				Name:  "allProjectsListInstanceID",
		//				Value: []byte(allProjectsListInstanceID.Slice()),
		//			}, {
		//				Name:  "users",
		//				Value: usersByte,
		//			}},
		//		},
		//	}},
		//}
		//require.Nil(t, ctx.Instructions[0].SignBy(allManagersDarc.GetBaseID(), managers[0]))
		//_, err = cl.AddTransaction(ctx)
		//require.Nil(t, err)

	// Demo1 ////////////////////////////////////////////////////////
	// Get the list of all projects/actions a user is associated with
	ctx = service.ClientTransaction{
		Instructions: []service.Instruction{{
			InstanceID: service.NewInstanceID(allUsersDarc.GetBaseID()),
			Nonce:  service.Nonce{},
			Index:  0,
			Length: 1,
			Spawn: &service.Spawn{
				ContractID: ContractProjectListID,
				Args: []service.Argument{{
					Name:  "userProjectsMapInstanceID",
					Value: []byte(userProjectsMapInstanceID.Slice()),
				}},
			},
		}},
	}
	require.Nil(t, ctx.Instructions[0].SignBy(allUsersDarc.GetBaseID(), users[0]))

	// To see if the transactions can be sent/retrieved over the network
	//data, err := network.Marshal(&ctx)
	//if err != nil {
	//	fmt.Println("error:", err)
	//}
	//var testTransactionRetrieved *service.ClientTransaction
	//_, tmp, err := network.Unmarshal(data, cothority.Suite)
	//if err != nil {
	//	fmt.Println("error:", err)
	//}
	//if err != nil {
	//	fmt.Println("error:", err)
	//}
	//testTransactionRetrieved, ok := tmp.(*service.ClientTransaction)
	//if !ok {
	//	fmt.Println("Data of wrong time:", err)
	//}
	//_, err = cl.AddTransaction(*testTransactionRetrieved)

	_, err = cl.AddTransaction(ctx)
	require.Nil(t, err)
	instID := service.NewInstanceID(ctx.Instructions[0].Hash())

	pr, err = cl.WaitProof(instID, genesisMsg.BlockInterval, nil)
	require.True(t, pr.InclusionProof.Match())
	values, err := pr.InclusionProof.RawValues()
	require.Nil(t, err)
	println(string(values[0][:]))

	authorizedProjectDarcID := strings.Split(string(values[0][:]), "......")[1]
	authorizedProjectDarcID = strings.Split(authorizedProjectDarcID, "...")[1]
	authorizedProjectDarcIDx, _ := hex.DecodeString(authorizedProjectDarcID[5:])

	// Demo2 //////////////////////////////////////////////////////////
	// Get authorization for a particular project for a particular user
	ctx = service.ClientTransaction{
		Instructions: []service.Instruction{{
			InstanceID: service.NewInstanceID(authorizedProjectDarcIDx),
			Nonce:  service.Nonce{},
			Index:  0,
			Length: 1,
			Spawn: &service.Spawn{
				ContractID: ContractAuthGrantID,
				Args: []service.Argument{},
			},
		}},
	}
	require.Nil(t, ctx.Instructions[0].SignBy(authorizedProjectDarcIDx, users[0]))

	_, err = cl.AddTransaction(ctx)
	require.Nil(t, err)
	instID = service.NewInstanceID(ctx.Instructions[0].Hash())
	pr, err = cl.WaitProof(instID, genesisMsg.BlockInterval, nil)
	require.True(t, pr.InclusionProof.Match())
	values, err = pr.InclusionProof.RawValues()
	require.Nil(t, err)
	println(string(values[0][:]))

	//Demo3 /////////////////////////////////////////////////////
	//Create a query with a particular type for a particular user
	ctx = service.ClientTransaction{
		Instructions: []service.Instruction{{
			InstanceID: service.NewInstanceID(authorizedProjectDarcIDx),
			Nonce:  service.Nonce{},
			Index:  0,
			Length: 1,
			Spawn: &service.Spawn{
				ContractID: ContractCreateQueryID,
				Args: []service.Argument{{
					Name:  "queryType",
					Value: []byte("AggregatedQuery"),
				}, {
					Name:  "query",
					Value: []byte("<bla bla>"),
				}},
			},
		}},
	}
	require.Nil(t, ctx.Instructions[0].SignBy(authorizedProjectDarcIDx, users[0]))

	_, err = cl.AddTransaction(ctx)
	require.Nil(t, err)
	instID = service.NewInstanceID(ctx.Instructions[0].Hash())
	pr, err = cl.WaitProof(instID, genesisMsg.BlockInterval, nil)
	require.True(t, pr.InclusionProof.Match())
	values, err = pr.InclusionProof.RawValues()
	require.Nil(t, err)
	println(string(values[0][:]))

	// Wrap things up
	local.WaitDone(genesisMsg.BlockInterval)
}

func createDarc(client *service.Client, baseDarc *darc.Darc, interval time.Duration, rules darc.Rules, description string, signers ...darc.Signer) (*darc.Darc, error) {
	// Create a transaction to spawn a DARC
	tempDarc := darc.NewDarc(rules, []byte(description))
	tempDarcBuff, err := tempDarc.ToProto()
	if err != nil {
		return nil, err
	}
	ctx := service.ClientTransaction{
		Instructions: []service.Instruction{{
			InstanceID: service.NewInstanceID(baseDarc.GetBaseID()),
			Nonce:  service.Nonce{},
			Index:  0,
			Length: 1,
			Spawn: &service.Spawn{
				ContractID: service.ContractDarcID,
				Args: []service.Argument{{
					Name:  "darc",
					Value: tempDarcBuff,
				}},
			},
		}},
	}
	if err := ctx.Instructions[0].SignBy(baseDarc.GetBaseID(), signers...); err != nil {
		return nil, err
	}

	// Commit transaction
	if _, err := client.AddTransaction(ctx); err != nil {
		return nil, err
	}

	// Verify DARC creation before returning its reference
	instID := service.NewInstanceID(tempDarc.GetBaseID())
	pr, err := client.WaitProof(instID, interval * time.Duration(waitFactor), nil)
	if err != nil || pr.InclusionProof.Match() == false {
		return nil, err
	}

	return tempDarc, nil
}