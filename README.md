# Medchain

Medchain is a distributed Identity Provider and Access Control service designed for medical databases.

It is based on a fork of omniledger, a blockchain developped by the dedis lab (now called Byzcoin)

## Set Up Omniledger

To set up omniledger, you have to get the forked version of cothority that was used for mechain

First clone it :

`cd $GOPATH/src/github.com/DPPH`

`git clone git@github.com:DPPH/cothority.git`  

Then copy the medchain contracts in the omniledger :

`cp $GOPATH/src/github.com/DPPH/MedChain/contracts/medchain.go $GOPATH/src/github.com/DPPH/cothority/omniledger/contracts/medchain.go`

Then copy these lines after line 30 of `$GOPATH/src/github.com/DPPH/cothority/omniledger/contracts/service.go` :

```
service.RegisterContract(c, ContractProjectListID, ContractProjectList)
service.RegisterContract(c, ContractProjectListIDSlow, ContractProjectListSlow)
service.RegisterContract(c, ContractAuthGrantID, ContractAuthGrant)
service.RegisterContract(c, ContractCreateQueryID, ContractCreateQuery)
service.RegisterContract(c, ContractUserProjectsMapID, ContractUserProjectsMap)
```

Then copy the whole cothority folder in the dedis folder:

`cp -rf $GOPATH/src/github.com/DPPH/cothority/ $GOPATH/src/github.com/dedis/cothority/`


(Warning : if you use cothority for a different project, this might be a problem)

## medChainServer :

The server interacts with the omniledger chain to grant access to users, perform log-ins and queries authorizations by creating tokens.

run `cd medChainServer; go build; ./medChainServer -h` to see the flags and their use

## medChainUtils :

Helper functions used by the Service

## medChainDocker :

Some scripts and Docker files to set up the Service

## medChainClient :

A client for users to get login tokens, and register queries in the chain. To be later used by the querying service to verify that the query was Authorized

## medChainAdmin :

This parts is a local client that is mainly used to perform signatures, in order to avoid sharing the private key with the server. The reason we use it is because we were unable to translate the signature library in javascript, that could enable us to do the signature directly in the browser.

run `cd medChainAdmin; go build; ./medChainAdmin -h` to see the flags and their use

## signingService :

Our solution to collect the multiple signatures was to have a centralized service to keep the transactions that needed to be signed. This is done by this signing service. The actions are registered in the service and then updated every time someone approves and signs the transaction.

It uses sqlite3 for the database.

run `cd signingService; go build; ./signingService -h` to see the flags and their use

## Demo :

run `cd medChainDocker; ./launch_demo.sh` to run the Demo

wait for the end of the bootstrapping (it should output "Success", it takes around 20s)

then go to http://localhost:8989/gui

The keys for the demo are in `medChainServer/keys/` and the bootstrapped configuration can be found in `medChainServer/conf/test_conf.json`