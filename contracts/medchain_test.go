package contracts

import (
	"testing"
	"time"

	"encoding/hex"
	"github.com/dedis/cothority/omniledger/darc"
	"github.com/dedis/cothority/omniledger/darc/expression"
	"github.com/dedis/cothority/omniledger/service"
	"github.com/stretchr/testify/require"
	"strings"
	"math/rand"
	"fmt"
	"github.com/montanaflynn/stats"
	"math"
	"github.com/dedis/onet"
	"github.com/dedis/cothority"
	"github.com/pkg/errors"
)

var rosterPath = ""
var blockInterval = 500 * time.Millisecond

// Need to adjust based on the complexity of test
var waitLoopFactor = 1000

func Test_System_Rigorous(t *testing.T) {
	demo1 := []float64{}
	transCr := []float64{}
	wait := []float64{}
	verify := []float64{}
	demo2 := []float64{}

	for pp := 0; pp < 1; pp++ {
		fmt.Printf("***** EXPERIMENT RUN %d *****\n", pp)
		// Complexity
		numberOfHospitals := 5
		adminsPerHospital := 1
		managersPerHospital := 5
		usersPerHospital := 10
		//serversPerHospital := 1
		numberOfProjects := numberOfHospitals * 3

		// Virtual servers
		local := onet.NewTCPTest(cothority.Suite)
		defer local.CloseAll()
		_, roster, _ := local.GenTree(3, true)

		// Servers from a roster
		//in, err := os.Open(rosterPath)
		//require.Nil(t, err)
		//g, err := app.ReadGroupDescToml(in)
		//require.Nil(t, err)
		//roster := g.Roster

		cl := service.NewClient()

		// Admins
		admins := []darc.Signer{}
		for i := 0; i < numberOfHospitals*adminsPerHospital; i++ {
			admins = append(admins, darc.NewSignerEd25519(nil, nil))
		}

		// Managers
		managers := []darc.Signer{}
		for i := 0; i < numberOfHospitals*managersPerHospital; i++ {
			managers = append(managers, darc.NewSignerEd25519(nil, nil))
		}

		// Users
		users := []darc.Signer{}
		for i := 0; i < numberOfHospitals*usersPerHospital; i++ {
			users = append(users, darc.NewSignerEd25519(nil, nil))
		}

		println("Finished creating keys...")

		// Create genesis block
		allAdminIdentities := ExtractIdentities(numberOfHospitals*adminsPerHospital, admins)
		allAdminStrings := ExtractIdentityStrings(numberOfHospitals*adminsPerHospital, admins)

		genesisMsg, err := service.DefaultGenesisMsg(service.CurrentVersion, roster,
			[]string{}, allAdminIdentities...)
		require.Nil(t, err)

		gDarc := &genesisMsg.GenesisDarc
		gDarc.Rules.UpdateSign(expression.InitAndExpr(allAdminStrings...))
		gDarc.Rules.AddRule("spawn:darc", gDarc.Rules.GetSignExpr())

		genesisMsg.BlockInterval = blockInterval
		genesisBlock, err := cl.CreateGenesisBlock(genesisMsg)
		require.Nil(t, err)

		// Create a DARC for managers of each hospital
		managersDarcs := []*darc.Darc{}
		for i := 0; i < numberOfHospitals; i++ {
			managersOfHospital := managers[i*managersPerHospital : (i+1)*managersPerHospital]
			adminsOfHospital := admins[i*adminsPerHospital : (i+1)*adminsPerHospital]

			rules := darc.InitRules(ExtractIdentities(adminsPerHospital, adminsOfHospital),
				ExtractIdentities(managersPerHospital, managersOfHospital))
			tempDarc, err := createDarc(cl, genesisBlock, gDarc, genesisMsg.BlockInterval, rules,
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
		allManagersDarc, err := createDarc(cl, genesisBlock, gDarc, genesisMsg.BlockInterval, rules,
			"AllManagers darc", admins...)
		require.Nil(t, err)

		println("Finished creating DARCs for managers...")

		// Create a DARC for users of each hospital
		usersDarcs := []*darc.Darc{}
		for i := 0; i < numberOfHospitals; i++ {
			usersOfHospital := users[i*usersPerHospital : (i+1)*usersPerHospital]

			rules := darc.InitRules([]darc.Identity{darc.NewIdentityDarc(managersDarcs[i].GetID())},
				ExtractIdentities(usersPerHospital, usersOfHospital))
			tempDarc, err := createDarc(cl, genesisBlock, allManagersDarc, genesisMsg.BlockInterval, rules,
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
		allUsersDarc, err := createDarc(cl, genesisBlock, allManagersDarc, genesisMsg.BlockInterval, rules,
			"AllUsers darc", managers...)
		require.Nil(t, err)

		println("Finished creating DARCs for users...")

		// Create projects
		projects := []*darc.Darc{}
		projectsIdentityString := ""
		tmp3 := []darc.Identity{}
		for i := 0; i < len(managersDarcs); i++ {
			tmp3 = append(tmp3, darc.NewIdentityDarc(managersDarcs[i].GetID()))
		}
		for q := 0; q < numberOfProjects; q++ {
			owners := []darc.Identity{}
			signers := []darc.Identity{}
			// Each project has ~20% of hospitals, chosen randomly, collaborating
			list := rand.Perm(len(managersDarcs))
			for i, v := range list {
				if i >= int(math.Floor(float64(len(managersDarcs)) * (0.20))) {
					break
				}
				owners = append(owners, darc.NewIdentityDarc(managersDarcs[v].GetID()))
				signers = append(signers, darc.NewIdentityDarc(usersDarcs[v].GetID()))
			}

			projectXDarcRules := darc.InitRules(owners, signers)
			// Define access control rules for the project DARC
			projectXDarcRules.AddRule("spawn:AuthGrant", projectXDarcRules.GetSignExpr())
			projectXDarcRules.AddRule("spawn:CreateQuery", projectXDarcRules.GetSignExpr())
			projectXDarcRules.AddRule(darc.Action("spawn:"+QueryTypes[0]), projectXDarcRules.GetSignExpr())
			projectXDarcRules.AddRule(darc.Action("spawn:"+QueryTypes[1]), projectXDarcRules.GetSignExpr())
			projectXDarc, err := createDarc(cl, genesisBlock, allManagersDarc, genesisMsg.BlockInterval, projectXDarcRules,
				"Project"+string(q), managers...)
			require.Nil(t, err)

			projectsIdentityString += projectXDarc.GetIdentityString() + ";"
			projects = append(projects, projectXDarc)
		}

		// Register all project DARCs with the value contract
		ctx := service.ClientTransaction{
			Instructions: []service.Instruction{{
				InstanceID: service.NewInstanceID(allManagersDarc.GetBaseID()),
				Nonce:      service.Nonce{},
				Index:      0,
				Length:     1,
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
		pr, err := cl.WaitProof(allProjectsListInstanceID, genesisMsg.BlockInterval, nil)
		for x := 0; x < waitLoopFactor; x++ {
			if err != nil {
				pr, err = cl.WaitProof(allProjectsListInstanceID, genesisMsg.BlockInterval, nil)
			} else {
				break
			}
		}
		require.True(t, pr.InclusionProof.Match())

		println("Finished creating DARCs for projects...")

		// Demo1/rigorous ///////////////////////////////////////////////
		// Get the list of all projects/actions a user is associated with
		startDemo1 := time.Now()
		startTransCr := time.Now()
		ctx = service.ClientTransaction{
			Instructions: []service.Instruction{{
				InstanceID: service.NewInstanceID(allUsersDarc.GetBaseID()),
				Nonce:      service.Nonce{},
				Index:      0,
				Length:     1,
				Spawn: &service.Spawn{
					ContractID: ContractProjectListIDSlow,
					Args: []service.Argument{{
						Name:  "allProjectsListInstanceID",
						Value: []byte(allProjectsListInstanceID.Slice()),
					}},
				},
			}},
		}
		require.Nil(t, ctx.Instructions[0].SignBy(allUsersDarc.GetBaseID(), users[0]))
		_, err = cl.AddTransaction(ctx)
		elapsedTransCr := time.Since(startTransCr)
		require.Nil(t, err)
		startWait := time.Now()
		instID := service.NewInstanceID(ctx.Instructions[0].Hash())
		pr, err = cl.WaitProof(instID, genesisMsg.BlockInterval, nil)
		for x := 0; x < waitLoopFactor; x++ {
			if err != nil {
				pr, err = cl.WaitProof(instID, genesisMsg.BlockInterval, nil)
			} else {
				break
			}
		}
		elapsedWait := time.Since(startWait)
		startVerify := time.Now()
		require.Nil(t, pr.Verify(genesisBlock.Skipblock.Hash))
		elapsedVerify := time.Since(startVerify)
		elapsedDemo1 := time.Since(startDemo1)
		values, err := pr.InclusionProof.RawValues()
		require.Nil(t, err)
		println("Demo1 took " + elapsedDemo1.String())
		println("TransCr took " + elapsedTransCr.String())
		println("Wait took " + elapsedWait.String())
		println("Verify took " + elapsedVerify.String())

		demo1 = append(demo1, elapsedDemo1.Seconds())
		transCr = append(transCr, elapsedTransCr.Seconds())
		wait = append(wait, elapsedWait.Seconds())
		verify = append(verify, elapsedVerify.Seconds())

		println(string(values[0][:]))
		authorizedProjectDarcID := strings.Split(string(values[0][:]), "......")[1]
		authorizedProjectDarcID = strings.Split(authorizedProjectDarcID, "...")[1]
		authorizedProjectDarcIDx, _ := hex.DecodeString(authorizedProjectDarcID[5:])

		//Demo2/rigorous /////////////////////////////////////////////
		//Create a query with a particular type for a particular user
		startDemo2 := time.Now()
		ctx = service.ClientTransaction{
			Instructions: []service.Instruction{{
				InstanceID: service.NewInstanceID(authorizedProjectDarcIDx),
				Nonce:      service.Nonce{},
				Index:      0,
				Length:     1,
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
		for x := 0; x < waitLoopFactor; x++ {
			if err != nil {
				pr, err = cl.WaitProof(instID, genesisMsg.BlockInterval, nil)
			} else {
				break
			}
		}
		require.Nil(t, pr.Verify(genesisBlock.Skipblock.Hash))
		elapsedDemo2 := time.Since(startDemo2)
		values, err = pr.InclusionProof.RawValues()
		require.Nil(t, err)
		println("Demo2 took " + elapsedDemo2.String())
		demo2 = append(demo2, elapsedDemo2.Seconds())
		println(string(values[0][:]))
		cl.Close()
	}

	d1MD, _ := stats.Median(demo1)
	d2MD, _ := stats.Median(demo2)
	crMD, _ := stats.Median(transCr)
	waitMD, _ := stats.Median(wait)
	verifyMD, _ := stats.Median(verify)

	fmt.Printf("Demo1 took %fs \n", d1MD)
	fmt.Printf("Demo2 took %fs \n", d2MD)
	fmt.Printf("TransCr took %fs \n", crMD)
	fmt.Printf("Wait took %fs \n", waitMD)
	fmt.Printf("Verify took %fs \n", verifyMD)
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

func Test_System_Simple(t *testing.T) {
	// Virtual servers
	local := onet.NewTCPTest(cothority.Suite)
	defer local.CloseAll()
	_, roster, _ := local.GenTree(3, true)

	// Servers from a roster
	//in, err := os.Open("/Users/talhaparacha/Desktop/public.toml")
	//require.Nil(t, err)
	//g, err := app.ReadGroupDescToml(in)
	//require.Nil(t, err)
	//roster := g.Roster

	cl := service.NewClient()
	defer cl.Close()

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

	genesisMsg.BlockInterval = blockInterval
	genesisBlock, err := cl.CreateGenesisBlock(genesisMsg)
	require.Nil(t, err)

	// Create a DARC for managers of each hospital
	managersDarcs := []*darc.Darc{}
	for i := 0; i < len(managers); i++ {
		rules := darc.InitRules([]darc.Identity{admins[i].Identity()},
			[]darc.Identity{managers[i].Identity()})
		tempDarc, err := createDarc(cl, genesisBlock, gDarc, genesisMsg.BlockInterval, rules,
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
	allManagersDarc, err := createDarc(cl, genesisBlock, gDarc, genesisMsg.BlockInterval, rules,
		"AllManagers darc", admins...)
	require.Nil(t, err)

	// Create a DARC for users of each hospital
	usersDarcs := []*darc.Darc{}
	for i := 0; i < len(users); i++ {
		rules := darc.InitRules([]darc.Identity{darc.NewIdentityDarc(managersDarcs[i].GetID())},
			[]darc.Identity{users[i].Identity()})
		tempDarc, err := createDarc(cl, genesisBlock, allManagersDarc, genesisMsg.BlockInterval, rules,
			"Users darc", managers...)
		require.Nil(t, err)
		usersDarcs = append(usersDarcs, tempDarc)
	}

	// Create a collective users DARC
	rules = darc.InitRules([]darc.Identity{darc.NewIdentityDarc(allManagersDarc.GetID())},
		[]darc.Identity{darc.NewIdentityDarc(usersDarcs[0].GetID()), darc.NewIdentityDarc(usersDarcs[1].GetID()),
			darc.NewIdentityDarc(usersDarcs[2].GetID())})
	rules.AddRule("spawn:ProjectList", rules.GetSignExpr())
	allUsersDarc, err := createDarc(cl, genesisBlock, allManagersDarc, genesisMsg.BlockInterval, rules,
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
	projectXDarc, err := createDarc(cl, genesisBlock, allManagersDarc, genesisMsg.BlockInterval, projectXDarcRules,
		"ProjectX", managers...)
	require.Nil(t, err)

	// Register the sample project DARC with the value contract
	myvalue := []byte(projectXDarc.GetIdentityString())
	ctx := service.ClientTransaction{
		Instructions: []service.Instruction{{
			InstanceID: service.NewInstanceID(allManagersDarc.GetBaseID()),
			Nonce:      service.Nonce{},
			Index:      0,
			Length:     1,
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
	pr, err := cl.WaitProof(allProjectsListInstanceID, genesisMsg.BlockInterval*5, nil)
	require.True(t, pr.InclusionProof.Match())

	// Create a users-projects map contract instance
	usersByte := []byte(users[2].Identity().String() + ";" + users[0].Identity().String())
	ctx = service.ClientTransaction{
		Instructions: []service.Instruction{{
			InstanceID: service.NewInstanceID(allManagersDarc.GetBaseID()),
			Nonce:      service.Nonce{},
			Index:      0,
			Length:     1,
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
			Nonce:      service.Nonce{},
			Index:      0,
			Length:     1,
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
	require.Nil(t, err)
	require.Nil(t, pr.Verify(genesisBlock.Skipblock.Hash))
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
			Nonce:      service.Nonce{},
			Index:      0,
			Length:     1,
			Spawn: &service.Spawn{
				ContractID: ContractAuthGrantID,
				Args:       []service.Argument{},
			},
		}},
	}
	require.Nil(t, ctx.Instructions[0].SignBy(authorizedProjectDarcIDx, users[0]))

	_, err = cl.AddTransaction(ctx)
	require.Nil(t, err)
	instID = service.NewInstanceID(ctx.Instructions[0].Hash())
	pr, err = cl.WaitProof(instID, genesisMsg.BlockInterval, nil)
	require.Nil(t, err)
	require.Nil(t, pr.Verify(genesisBlock.Skipblock.Hash))
	require.True(t, pr.InclusionProof.Match())
	values, err = pr.InclusionProof.RawValues()
	require.Nil(t, err)
	println(string(values[0][:]))

	//Demo3 /////////////////////////////////////////////////////
	//Create a query with a particular type for a particular user
	ctx = service.ClientTransaction{
		Instructions: []service.Instruction{{
			InstanceID: service.NewInstanceID(authorizedProjectDarcIDx),
			Nonce:      service.Nonce{},
			Index:      0,
			Length:     1,
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
	require.Nil(t, err)
	require.Nil(t, pr.Verify(genesisBlock.Skipblock.Hash))
	require.True(t, pr.InclusionProof.Match())
	values, err = pr.InclusionProof.RawValues()
	require.Nil(t, err)
	println(string(values[0][:]))
}

func createDarc(client *service.Client, genesisBlock *service.CreateGenesisBlockResponse, baseDarc *darc.Darc, interval time.Duration, rules darc.Rules, description string, signers ...darc.Signer) (*darc.Darc, error) {
	// Create a transaction to spawn a DARC
	tempDarc := darc.NewDarc(rules, []byte(description))
	tempDarcBuff, err := tempDarc.ToProto()
	if err != nil {
		return nil, err
	}
	ctx := service.ClientTransaction{
		Instructions: []service.Instruction{{
			InstanceID: service.NewInstanceID(baseDarc.GetBaseID()),
			Nonce:      service.Nonce{},
			Index:      0,
			Length:     1,
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
	pr, err := client.WaitProof(instID, interval, nil)
	for x := 0; x < waitLoopFactor; x++ {
		if err != nil {
			pr, err = client.WaitProof(instID, interval, nil)
		} else {
			break
		}
	}
	if err != nil || pr.Verify(genesisBlock.Skipblock.Hash) != nil {
		return nil, errors.New("Error during proof validation while creating a DARC")
	}
	return tempDarc, nil
}
