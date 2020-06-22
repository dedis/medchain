# Administration API


The admin directory contains the implementation of the medchain administration service. This code includes the administration API to maintain the Medchain ecosystem, interacting with the Byzcoin ledger, the accessright contract code and the shareID service used to share the ID of the deferred transaction spawned by the different methods of the API.

## Overview

Here is a high overview of the file structure of the administration service:

- `accessright.go`, `accessright\_test.go`: the implementation of the accessright contract
- `admin\_api.go`, `admin\_api\_test.go`: the implementation of the administration API
- `struct.go`: defines the different structures used for the administration of Medchain. The `shareID` service messages and the accessright contract stored data.  The `init` method of this file, register the ShareID messages in onet network.
- `service.go`: the logic of the `shareID` service. The `init` method of this file, deploy the `shareID` service and the accessright contract on-chain.
- `utils.go`, `utils\_test.go`: define helper methods to manage lists as sets in go. It also defines some methods with a logic that is independent of the admin API (e.g the spawnTransaction method that only calls methods of Byzcoin)


The administration  Application Programming Interface (API) contains several methods to manage the Medchain system. The methods defined in the API interact with Byzcoin by spawning transactions to update the global state.

## API

The administration Application Programming Interface (API) contains several methods to manage the Medchain system. The methods defined in the API interact with Byzcoin by spawning transactions to update the global state.


## ShareID service

The ShareID service is used to share the instance id of the deferred transaction spawned on the chain and the id of the administration darc to every conode in the roster. It uses the [propagate.go](https://github.com/ldsec/medco-unlynx/tree/master/protocols) protocol used in the Unlynx project, to broadcast the ID to every node and store it in their local databases.

## Access Right contract

The access right contract is the contract spawned by project darcs to hold the access rights of the different users. This contract holds the AccessRight structure that contains two lists:


- Ids that hold the list of all queriers identities
- Access that holds the list of all access


There is a one to one mapping between the indexes of the two lists.

The reason why two lists are used instead of a map that map IDs to access rights is that Byzcoin doesn't support maps. Hashmap operations in Golang are not deterministic and this operation is not compatible with the consensus process of Byzcoin.

This contract implement three invoke methods:

- **add:** add a user in the access rights
- **remove:** remove from the access rights
- **update:** modify the access rights of a user

---------

## Important implementation details

### Multisignature and Administration darc evolution

For security and resilience reasons, every sensitive administrative action is performed under multisignature rules defined as darc expressions. 

Each time we add/remove or modify an administrator identity in the administration darc, we need to recompute all rules to satisfy the multisignature scheme chosen:

1. We get the administrator list stored in a value contract instance that has a name resolver to `adminList`.
2. We compute the new list by adding, removing or modifying the right identity
3. We give this list as an argument to the `createMultisigRuleExpression` method. This method takes as argument the identity list and the minimum `min` number of signature required for the expression to be valid. It returns the expression (all the combination of the `min` out of N expression)
4. We then update the darc rules with this new expression
5. We spawn a deferred transaction that contains one instruction to update the value contract with the new list and another instruction to evolve the administration darc

> During the development of the project we required every admin to sign every sensitive operation. Meaning the createMultisigRuleExpression takes the length of the list as the minimum number of signature required

To modify the multisignature behavior, we only need to choose the minimum number of signature required, for a sensitive transaction to be executed by the darc.

### Name resolution

The API uses a lot the name resolution service of Byzcoin. 

Having name resolution is useful for not having to remember all instance id of every contract or having to give the instance id for each call to the API. The API instead performs name resolution to get the instance id and the content of the contract.

The naming contract is used to have name resolution of instance id. We can bind the instance id of any contract to a specific name to avoid storing all the instance ids. 

> The naming contract needs to be spawned once when setting up the Byzcoin blockchain, and then we can invoke it to create name resolvers.

The instance id is retrieved both based on the name defined and the id of the darc that guard the contract. We can, therefore, have several contracts instance ids that resolve to the same name if they are managed by different darcs. 

> For example We use this feature to have all our access right contracts that resolve to the same name ’AR’ but are spawned by different project darcs.


## Run tests 


To test the API, the accessright contract and the ShareID service run the following:

    path/to/admin$ go test

This will lunch every test in the `accessright_test.go`, `admin_api_test.go` and `utils_test.go`