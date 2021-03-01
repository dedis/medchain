# MedChain

The MedChain is an authorization and access management service is for medical queries. It is written in GO and is based on [Cothority](https://github.com/dedis/cothority/blob/master/README.md) and [ByzCoin](https://github.com/dedis/cothority/blob/master/byzcoin/README.md). 
MedChain uses [Darcs](https://github.com/dedis/cothority/blob/master/darc/README.md) to authorize the queries.  

# Run a Cothority

You can run a set of nodes by running the following:

```sh
cd conode
go build -o conode && ./run_nodes.sh -v 3 -d tmp
```

It will setup 3 nodes and save their files in conode/tmp.