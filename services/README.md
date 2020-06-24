# MedChain Services 

In this directory you will find code implementation of MedChsin services. Services are run by MedChain nodes. They keep a persistent state and need to be restarted if the node crashes.
Services
- can start protocols
- communicate with other service-instances
- communicate with the clients through an API
- have state that is kept over restart of the server

In Cothority, every app (e.g.  Command-Line-Interface app) or client (at the front-end) communicates with services to interact with Conodes.  In other words, services are responsible forhandling Client-to-Conode communications.  Services are based on [Onet](https://github.com/dedis/onet) library that provides the overlay network and enables definition of communication protocols in Cothority.  A service is created with Conode.  It serves client requests,  creates protocols for Overlay network, and handles communication of the information to other services on other Conodes. 

In MedChain node implementation, in order to define client-MedChain node communications and enable message sharing among them, we defined our own service.  MedChain service code can be found in `service.go`.  In MedChain service, that is based on Onet, we define services that use Protocols which can send and receive messages.This means that MedChain node uses Onet to send/receive messages to/from the client over the service-API, which is implemented using protobuf over WebSockets.  Messages defined in MedChain are explained in next section.

## MedChain Messages and API Calls

In Medchain, we mainly used two services: [ByzCoin](https://github.com/dedis/cothority/tree/master/byzcoin) and [Onet](https://github.com/dedis/onet) services that enable client-server communications. We define Medchain API using these services and later implement the CLI-program on top of this API (see [MedChain CLI client code](../cmd/medchain-cli-client)). Messages defined and used in MedChain can be found in `struct.go` in this directory. Table below shows some of the most important messages defined in MedChain as well as resources they take and their responses.

|Name of Method | Description| Resources| Response|
| ------ | ------ | ------ | ------ | 
|`AddQuery` | Spawn a query transaction | User ID, <br> Query definition, <br> Darc ID | Instance ID, <br> OK? |
| `AddDeferredQuery`| Spawn a deferred query transaction  | User ID, <br> Query definition, <br> Instance ID, <br> Darc ID | Instance ID, <br> OK? |
| `SignDeferredTx`| Sign a deferred query transaction  | User ID,<br> User keys, <br> Instance ID | Instance ID, <br> OK? |
|`ExecDeferredTx`| Execute a deferred query transaction  | User ID,<br> Instance ID, | Instance ID, <br> OK? |
|`AuthorizeQuery`| Authorize a query  | Query definition,<br> Instance ID, <br> Darc ID| Instance ID, <br> Query Status,<br> OK? |
| `GetSharedData`| Get instance IDs shared with node  | - | Instance IDs | 
|`PropageteID`| Broadcast instance IDs to roster | Instance ID,<br> Roster| OK? | 

## MedChain API (Application Programming Interface)

MedChain API is implemented in `api.go`. Also, examples of MedChain Go API usage can be found in `api_test.go`.

The most important building blocks of MedChain API are:

- Smart contract
- Deferred Transactions
- Darcs (Distributed Access Right Controls)

Each of the above building blocks is explained in the following sections.

### Smart Contract

In Byzcoin, smart contract defines the data that is added to the ledger, i.e., data written to the ledger is, in fact, instances of smart contract. Also, smart contract gives us the API to manipulate the instances, for example, in order to create a new instance, we can `Spawn` an instance of smart contract and to update the contract instance, we can apply an `Invoke()` to it. 

In MedChain, we define our own smart contract that implements the suitable data structure: **the query**. We developed the smart contract in Go (MedChain smart contract is implemented in `contract.go`. This smart contract is a simple key-value contract, meaning that every instance of the contract that is recorded in the ledger, i.e., the query, is a key-value store data structure. Medchain smart contract is identified by its name `MedChainContract`. This contract is distinguishable from any other contracts used in Medchain deployment such as Darc contracts (see following sections) or Value contracts by its ID `MedChainContractID`. The client uses this ID in order to define which contract instance he/she wants to manipulate. The queries submitted to MedChain are recorded in the ledger as instances of `MedChainContract`. The data structure the contract implements (i.e., the structure of query) is defined in `proto.go`. 

In ByzCoin, it is mandatory that the smart contract implement three API methods: `Spawn`, `Invoke`, and `Delete`. The very first instance of a contract is created by creating a transaction using `Spawn` as the instruction. By this transaction, the user sets the query ID as the key of the instance and its status as the value. Later, the user can manipulate this instance of the contract by calling an `Invoke` on it. The `Invoke` method in `MedChainContract` implements two methods itself: `update` and `verifystatus`. Using update, the user is able to retrieve a specific contract instance (i.e., query) from the Skipchain by its key (i.e., query ID) and update its value (i.e., the status of the query) if it already exists in the skipchain, however, if that is not the case, a new instance of the contract will be spawned using the provided key-value pair. `verifystatus` method is used by Medchain server itself to retrieve a query from the ledger and verify its status in a similar manner to `update` method. In Medchain, we decided not to implement the `Delete` function in contract as we do not want the user to be able to remove a query (i.e., an instance of the contract) from the global state. 

Whenever a contract instance is created (spawned) it is allocated an `InstanceID` that is determined based on the ID of the Darc contract ruling it. Later, this instance of the contract is retrievable and authorized by the Darc controlling it using this `InstanceID`.

### Deferred Transactions

In order to enable multi-signature rules in MedChain, we use ByzCoin _deferred transactions_. Deferred transactions allow a transaction to remain **proposed** (i.e., not written to the ledger) until it receives the threshold number of signatures defined by the Darc governing it. Once it receives enough number of signatures, it can be **executed** and written to the ledger. 

In order to enable deferred transactions in ByzCoin server, the developer should define a special method in the smart contract, namely, `VerifyDeferredInstruction`, which is not implemented in a `BasicContract` (i.e., the basic data structure that all contracts implement by default). In other words, types which embed `BasicContract` must override this method if they want to support deferred transactions (using the `Deferred contract`). 

To enable deferred execution of a `MedChainContract` instance, the following steps are taken:

1. User spawns a `MedChainContract` instance (This is the proposed transaction).
2. User spawns a `deferred_contract` instance with query instance ID as its arguments. In other words, the proposed transaction is the what the deferred instance holds.
3. Signers sign the proposed transaction by invoking an `addProof` on it.
4. User invokes an `execProposedTx` on proposed transaction to execute it.

### Darcs (Distributed Access Right Controls)

[Darcs](https://github.com/dedis/cothority/tree/master/darc) are used to enable authorization in MedChain. In ByzCoin, Darcs are responsible for handling authorization and access management for various resources, such as smart contract instances, and they use action/expression pairs to define rules.

In MedChain, projects define various databases and in order to control access to the database every project is associated with a Darc. 

The new instance of `MedChainContract` spawned will have an instance ID equal to the hash of the Spawn instruction. This instance ID is the hook to this instance and the client needs to remember this it in order to manipulate this instance later, for example, invoke methods on it to update its status. In Medchain, we also use `contract_name` of ByzCoin. This contract is a singleton contract that is always created in the genesis block. One can only invoke the `naming contract` to relate a Darc ID and name tuple to another instance ID. Once an instance is named, the client can the name given to the instance ID to retrieve it from the ledger.

After this step, a `MedChainContract` instance spawned by the user (which is in fact the query submitted to Medchain from MedCo) is bound to Project A Darc and is governed by it; thus, the Darc can check for the authorizations of the action the client is trying to take. Also, the instance ID of the  

Since the very first transaction, i.e., the one created right after MedChain receives a query from MedCo, is always immediately written to the ledger (due to auditability purposes) using a `Spawn` instruction of `Value` contract, all project Darcs need to have `spawn:value` and `invoke:value` rules enabled for all Darc users (i.e., using an _at least one_ expression).


## Directory overview

Below is the description of code and files avaliable in this directory:

- `api.go`: MedChain API implementation
- `api_test.go`: Go tests for MedChain API 
- `contract.go`: MedChain smart contract implementation
- `contract_test.go`: Go test for contract
- `proto.go`: Query structure definition 
- `service.go`: MedChain service implementation. It defines what to do for every API-call
- `service_test.go`: MedChain service Go test
- `struct.go`: The messages that will be sent around in MedChain