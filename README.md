# MedChain

The MedChain is an authorization and access management service is for medical queries. It is written in GO and is based on [Cothority](https://github.com/dedis/cothority/blob/master/README.md) and [ByzCoin](https://github.com/dedis/cothority/blob/master/byzcoin/README.md). 
MedChain uses [Darcs](https://github.com/dedis/cothority/blob/master/darc/README.md) to authorize the queries.  



## Running the service
The MedChain service is based on conodes. To find out more about conodes and how to run them,
please see [conode documentation](https://github.com/dedis/cothority/blob/master/conode/README.md).

## Client API

MedChain supports 2 ways for clients to connect to MedChain service: Go API and CLI. 
Before the client can use the API, a ByzCoin object should be created with a Darc that authorizes the client for "spawn:medchain" and "invoke:medchain.update". 

### Go API

The detailed API can be found in the directory 
[./client](https://github.com/ldsec/medchain/tree/dev/client). Examples and demos can also be found in `./client/api_test.go`.


### CLI 

The client is able to use command-line interface to use MedChain service. 
Please refer to `mc` documentation [here](mc/README.md) for further details.
