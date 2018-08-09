package main

import (
	"github.com/talhaparacha/medChain/medChainUtils"
	"github.com/dedis/cothority/omniledger/service"
	"github.com/dedis/cothority/omniledger/contracts"
	"encoding/base64"
	"github.com/dedis/onet/network"
	"time"
	"io/ioutil"
	"encoding/hex"
)

func createQueryTransaction(projectDarc string, queryType string, query string, pathToPublic string, pathToPrivate string) string {
	// We don't need the "darc:" part from the ID, and a
	projectDarcDecoded, err := hex.DecodeString(projectDarc[5:])
	medChainUtils.Check(err)

	ctx := service.ClientTransaction{
		Instructions: []service.Instruction{{
			InstanceID: service.InstanceID{
				DarcID: projectDarcDecoded,
				SubID:  service.SubID{},
			},
			Nonce:  service.Nonce{},
			Index:  0,
			Length: 1,
			Spawn: &service.Spawn{
				ContractID: contracts.ContractCreateQueryID,
				Args: []service.Argument{{
					Name:  "queryType",
					Value: []byte(queryType),
				}, {
					Name:  "query",
					Value: []byte(query),
				}, {
					Name:  "currentTime",
					Value: []byte(time.Now().String()),
				}},
			},
		}},
	}

	err = ctx.Instructions[0].SignBy(medChainUtils.LoadSignerEd25519(pathToPublic, pathToPrivate))
	medChainUtils.Check(err)
	data, err := network.Marshal(&ctx)
	medChainUtils.Check(err)
	return base64.StdEncoding.EncodeToString(data)
}

func createLoginTransaction(allUsersDarc string, userProjectsMap string, pathToPublic string, pathToPrivate string) string {
	allUsersDarcBytes, err := base64.StdEncoding.DecodeString(allUsersDarc)
	medChainUtils.Check(err)
	userProjectsMapBytes, err := base64.StdEncoding.DecodeString(userProjectsMap)
	medChainUtils.Check(err)

	ctx := service.ClientTransaction{
		Instructions: []service.Instruction{{
			InstanceID: service.InstanceID{
				DarcID: allUsersDarcBytes,
				SubID:  service.SubID{},
			},
			Nonce:  service.Nonce{},
			Index:  0,
			Length: 1,
			Spawn: &service.Spawn{
				ContractID: contracts.ContractProjectListID,
				Args: []service.Argument{{
					Name:  "userProjectsMapInstanceID",
					Value: userProjectsMapBytes,
				}, {
					Name:  "currentTime",
					Value: []byte(time.Now().String()),
				}},
			},
		}},
	}

	err = ctx.Instructions[0].SignBy(medChainUtils.LoadSignerEd25519(pathToPublic, pathToPrivate))
	medChainUtils.Check(err)
	data, err := network.Marshal(&ctx)
	medChainUtils.Check(err)
	return base64.StdEncoding.EncodeToString(data)
}

func main() {
	tmp := createLoginTransaction("dYSkNjIWYe3At5cDQAT957IHJ1WXkNaVlFP64vRB9Xk=",
		"LN3sD8dm2YJsIMFQsQFz47N0N1hp/VAINmuJgiEU6msrF7UujuBh6E1wkW2pSjfG7k4KuZZiRhyy5+zYmVk5zQ==",
		"/Users/talhaparacha/Desktop/keys/users/0_public",
		"/Users/talhaparacha/Desktop/keys/users/0_private")
	println(tmp)

	query, err := ioutil.ReadFile("/Users/talhaparacha/Desktop/query.json")
	medChainUtils.Check(err)
	tmp = createQueryTransaction("darc:6bfdbb6f2b467c6ea858c0f57198aac9faed8d3d441aec17b1f53cec82b5f1d2",
		"AggregatedQuery",
		string(query[:]),
		"/Users/talhaparacha/Desktop/keys/users/0_public",
		"/Users/talhaparacha/Desktop/keys/users/0_private")
	println(tmp)
}