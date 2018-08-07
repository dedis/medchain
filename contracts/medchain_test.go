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

func Test_System(t *testing.T) {
	local := onet.NewTCPTest(cothority.Suite)
	defer local.CloseAll()
	servers, roster, _ := local.GenTree(3, true)
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
				InstanceID: service.InstanceID{
					DarcID: allManagersDarc.GetBaseID(),
					SubID:  service.SubID{},
				},
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
		require.Nil(t, ctx.Instructions[0].SignBy(managers[0], managers[1], managers[2]))
		_, err = cl.AddTransaction(ctx)
		require.Nil(t, err)
		allProjectsListInstanceID := service.InstanceID{
			DarcID: ctx.Instructions[0].InstanceID.DarcID,
			SubID:  service.NewSubID(ctx.Instructions[0].Hash()),
		}
		pr, err := cl.WaitProof(allProjectsListInstanceID, genesisMsg.BlockInterval, nil)
		require.True(t, pr.InclusionProof.Match())

		// Create a users-projects map contract instance
		usersByte := []byte(users[2].Identity().String() + ";" + users[0].Identity().String())
		ctx = service.ClientTransaction{
			Instructions: []service.Instruction{{
				InstanceID: service.InstanceID{
					DarcID: allManagersDarc.GetBaseID(),
					SubID:  service.SubID{},
				},
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
		require.Nil(t, ctx.Instructions[0].SignBy(managers[0], managers[2]))
		_, err = cl.AddTransaction(ctx)
		require.Nil(t, err)
		userProjectsMapInstanceID := service.InstanceID{
			DarcID: ctx.Instructions[0].InstanceID.DarcID,
			SubID:  service.NewSubID(ctx.Instructions[0].Hash()),
		}
		pr, err = cl.WaitProof(userProjectsMapInstanceID, genesisMsg.BlockInterval, nil)
		require.True(t, pr.InclusionProof.Match())

		//// Try updating users-projects map contract instance
		//usersByte = []byte(users[0].Identity().String())
		//ctx = service.ClientTransaction{
		//	Instructions: []service.Instruction{{
		//		InstanceID: service.InstanceID{
		//			DarcID: userProjectsMapInstanceID.DarcID,
		//			SubID:  userProjectsMapInstanceID.SubID,
		//		},
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
		//require.Nil(t, ctx.Instructions[0].SignBy(managers[0]))
		//_, err = cl.AddTransaction(ctx)
		//require.Nil(t, err)

	// Demo1 ////////////////////////////////////////////////////////
	// Get the list of all projects/actions a user is associated with
	ctx = service.ClientTransaction{
		Instructions: []service.Instruction{{
			InstanceID: service.InstanceID{
				DarcID: allUsersDarc.GetBaseID(),
				SubID:  service.SubID{},
			},
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
	require.Nil(t, ctx.Instructions[0].SignBy(users[0]))
/*
	data, err := network.Marshal(&ctx)
	if err != nil {
		fmt.Println("error:", err)
	}
	var testTransactionRetrieved *service.ClientTransaction
	_, tmp, err := network.Unmarshal(data, cothority.Suite)
	if err != nil {
		fmt.Println("error:", err)
	}
	if err != nil {
		fmt.Println("error:", err)
	}
	testTransactionRetrieved, ok := tmp.(*service.ClientTransaction)
	if !ok {
		fmt.Println("Data of wrong time:", err)
	}
	_, err = cl.AddTransaction(*testTransactionRetrieved)
*/


	_, err = cl.AddTransaction(ctx)
	require.Nil(t, err)
	instID := service.InstanceID{
		DarcID: ctx.Instructions[0].InstanceID.DarcID,
		SubID:  service.NewSubID(ctx.Instructions[0].Hash()),
	}
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
			InstanceID: service.InstanceID{
				DarcID: []byte(authorizedProjectDarcIDx),
				SubID:  service.SubID{},
			},
			Nonce:  service.Nonce{},
			Index:  0,
			Length: 1,
			Spawn: &service.Spawn{
				ContractID: ContractAuthGrantID,
				Args: []service.Argument{},
			},
		}},
	}
	require.Nil(t, ctx.Instructions[0].SignBy(users[0]))

	_, err = cl.AddTransaction(ctx)
	require.Nil(t, err)
	instID = service.InstanceID{
		DarcID: ctx.Instructions[0].InstanceID.DarcID,
		SubID:  service.NewSubID(ctx.Instructions[0].Hash()),
	}
	pr, err = cl.WaitProof(instID, genesisMsg.BlockInterval, nil)
	require.True(t, pr.InclusionProof.Match())
	values, err = pr.InclusionProof.RawValues()
	require.Nil(t, err)
	println(string(values[0][:]))

	//Demo3 /////////////////////////////////////////////////////
	//Create a query with a particular type for a particular user
	ctx = service.ClientTransaction{
		Instructions: []service.Instruction{{
			InstanceID: service.InstanceID{
				DarcID: []byte(authorizedProjectDarcIDx),
				SubID:  service.SubID{},
			},
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
	require.Nil(t, ctx.Instructions[0].SignBy(users[0]))

	_, err = cl.AddTransaction(ctx)
	require.Nil(t, err)
	instID = service.InstanceID{
		DarcID: ctx.Instructions[0].InstanceID.DarcID,
		SubID:  service.NewSubID(ctx.Instructions[0].Hash()),
	}
	pr, err = cl.WaitProof(instID, genesisMsg.BlockInterval, nil)
	require.True(t, pr.InclusionProof.Match())
	values, err = pr.InclusionProof.RawValues()
	require.Nil(t, err)
	println(string(values[0][:]))

	// Wrap things up
	services := local.GetServices(servers, service.OmniledgerID)
	for _, s := range services {
		s.(*service.Service).ClosePolling()
	}
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
			InstanceID: service.InstanceID{
				DarcID: baseDarc.GetBaseID(),
				SubID:  service.SubID{},
			},
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
	if err := ctx.Instructions[0].SignBy(signers...); err != nil {
		return nil, err
	}

	// Commit transaction
	if _, err := client.AddTransaction(ctx); err != nil {
		return nil, err
	}

	// Verify DARC creation before returning its reference
	instID := service.InstanceID{
		DarcID: tempDarc.GetBaseID(),
		SubID:  service.SubID{},
	}
	pr, err := client.WaitProof(instID, interval, nil)
	if err != nil || pr.InclusionProof.Match() == false {
		return nil, err
	}

	return tempDarc, nil
}