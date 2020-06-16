# MedChain

The MedChain is an authorization and access management service is for medical queries. It is written in GO and is based on [Cothority](https://github.com/dedis/cothority/blob/master/README.md) and [ByzCoin](https://github.com/dedis/cothority/blob/master/byzcoin/README.md). 
MedChain uses [Darcs](https://github.com/dedis/cothority/blob/master/darc/README.md) to authorize the queries.  



## Running the service
The MedChain service is based on conodes. Conode is the main binary to rune a Cothority service. To find out more about conodes and how to run them,
please see [conode documentation](https://github.com/dedis/cothority/blob/master/conode/README.md).

## Client API

MedChain supports 2 ways for clients to connect to MedChain service: Go API and CLI. 
Before the client can use the API, a ByzCoin object should be created with a Darc that authorizes the client for "spawn:medchain" and "invoke:medchain.update". 

* ### Go API

The detailed API can be found in the directory 
[./services](https://github.com/ldsec/medchain/tree/dev/services). Examples and demos can also be found in `./services/api_test.go`.

* ### CLI 

The client is able to use command-line interface to use MedChain service. 
Please refer to `mccli` documentation [here](mc/README.md) for further details.


## Source code organization

- *build*: packaging and Continuous Integration.
    - *cmd/ci*: CI (travis) configurations and scripts
    - *cmd/package*: Docker package configurations and scripts
- *cmd*: main applications for this project
    - *cmd/app*: server application
    - *cmd/mccli*: client application
- *conode*: main binary for running a [Cothority](https://github.com/dedis/cothority/blob/master/README.md) server
- *deployment*: docker image definition and docker-compose deployment
- *protocol*: conode to conode communication implementation
- *services*: server-side API and service definitions
- *simulation*: service and protocol simulations
- *util*: utility code (configuration)
    - *client*: client-related utility code
    - *server*: server-related utility code