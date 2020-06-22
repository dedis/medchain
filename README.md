# MedChain

MedChain is a distributed authorization and access management service for MedCo. It is written in Go and is based on [Cothority](https://github.com/dedis/cothority/blob/master/README.md) and [ByzCoin](https://github.com/dedis/cothority/blob/master/byzcoin/README.md). MedChain uses [Darcs](https://github.com/dedis/cothority/blob/master/darc/README.md) to authorize queries.  


## MedChain Node (Server)
MedChain server code can be found in [cmd/medchain-server]()cmd/medchain-server) where you can also find instructions on how to run one or more MedChain nodes.

## MedChain Client

MedChain supports 2 ways for clients to connect to MedChain service: [Go API](service/api.go) and [CLI](cmd/medchain-cli-client). 
Before the client can use the API, a ByzCoin object should be created with a Darc that authorizes the client for "spawn:medchain" and "invoke:medchain.update". More instructions on clients can be found in [services](services) or [cmd/medchain-cli-client](cmd/medchain-cli-client)

* ### MedChain Go API Client

Detailed API can be found in the directory [services](https://github.com/ldsec/medchain/tree/dev/services). Examples of Go API usage can also be found in [services/api_test.go](services/api_test.go).

* ### MedChain CLI Client

The client is able to use command-line interface to use MedChain service. Please refer to `medchain-cli-client` documentation [here](cmd/medchain-cli-client/README.md) for further details.


## Source code organization

- *build*: packaging and Continuous Integration.
    - *cmd/ci*: CI (travis) configurations and scripts
    - *cmd/package*: Docker package configurations and scripts
- *cmd*: main applications for this project
    - *cmd/medchain-server*: server application
    - *cmd/medchain-cli-client*: client application
    - *cmd/medadmint*: MedChain admin client application
- *deployment*: docker image definition and docker-compose deployment of MedChain
- *protocol*: MedChain to MedChain communication implementation
- *services*: server-side API and service definitions
- *simulation*: service and protocol simulations
- *util*: utility code (configuration)
    - *client*: client-related utility code
    - *server*: server-related utility code