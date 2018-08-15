package main

import (
	"github.com/talhaparacha/medChain/medChainUtils"
	"io/ioutil"
)

func main() {
	// Load signer from keys
	signer := medChainUtils.LoadSignerEd25519("/Users/talhaparacha/Desktop/keys/users/0_public", "/Users/talhaparacha/Desktop/keys/users/0_private")

	// Transaction for logging the user
	tmp := medChainUtils.CreateLoginTransaction("dYSkNjIWYe3At5cDQAT957IHJ1WXkNaVlFP64vRB9Xk=",
		"LN3sD8dm2YJsIMFQsQFz47N0N1hp/VAINmuJgiEU6msrF7UujuBh6E1wkW2pSjfG7k4KuZZiRhyy5+zYmVk5zQ==", signer)
	println(tmp)

	// Transaction for logging the query on the MedChain
	query, err := ioutil.ReadFile("/Users/talhaparacha/Desktop/query.json")
	medChainUtils.Check(err)
	tmp = medChainUtils.CreateQueryTransaction("darc:6bfdbb6f2b467c6ea858c0f57198aac9faed8d3d441aec17b1f53cec82b5f1d2",
		"AggregatedQuery",
		string(query[:]), signer)
	println(tmp)
}