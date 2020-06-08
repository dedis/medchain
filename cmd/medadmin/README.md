# Medadmin : Medchain administration CLI

Medadmin is the Medchain administration CLI used to manage the admin darc, manage deferred transactions and manage projects.

## CLI calls

### Admin

     $ medadmin admin subcommand [options] arguments

The admin command manage the admin darc  

| Subcommands | Arguments                                                                                                                                                                                                                                                   | Description                                                                                                        |
|-------------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|--------------------------------------------------------------------------------------------------------------------|
| `get`       | `keys` : the ed25519 private key that will sign the create query transaction, `bc` : the ByzCoin config (default is $BC), `adid` : the admin darc id (default is $adid)                                                                                     | Get the list of all admins in admin darc                                                                           |
| `attach`    | `bc` : the ByzCoin config (default is $BC), `keys` : the ed25519 private key that will sign the create query transaction, `id` :the instance id of the value contract                                                                                       | Attach the admins list to admin darc *(need to be run only once at setup)*                                         |
| `add`       | `bc` : the ByzCoin config (default is $BC), `keys` : the ed25519 private key that will sign the create query transaction, `identity` : the new admin identity string, `adid` : the admin darc id (default is $adid)                                         | Add a new admin to the admin darc and admins list. Returns the instance id of the deferred transaction.            |
| `remove`    | `bc` : the ByzCoin config (default is $BC), `keys` : the ed25519 private key that will sign the create query transaction, `identity` : the new admin identity string, `adid` : the admin darc id (default is $adid)                                         | Remove an admin from the admin darc and from the admins list. Returns the instance id of the deferred transaction. |
| `modify`    | `bc` : the ByzCoin config (default is $BC), `keys` : the ed25519 private key that will sign the create query transaction, `oldkey` : the old admin identity string, `newkey` : the new admin identity string, `adid` : the admin darc id (default is $adid) | Modify the admin identity in the admin darc and admins list. Returns the instance id of the deferred transaction.  |

`create` :	Create a new admin, admin darc, admin list

| Subcommands | Arguments                                                                                                                                                               | Description                                                                                                      |
|-------------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------------|------------------------------------------------------------------------------------------------------------------|
| `darc`      | `keys` : the ed25519 private key that will sign the create query transaction, `bc` : the ByzCoin config (default is $BC)                                                | Spawn a new admin darc *(need to be run only once at setup)*                                                     |
| `admin`     | `bc` : the ByzCoin config (default is $BC)                                                                                                                              | Create a new admin identity                                                                                      |
| `list`      | `bc` : the ByzCoin config (default is $BC), `adid` : the admin darc id (default is $adid), `keys` : the ed25519 private key that will sign the create query transaction | Create the adminsList, the list that contains all admins public identities *(need to be run only once at setup)* |

### Deferred

     $ medadmin deffered subcommand [options] arguments

The defferred command manages the deffered transaction registered in the global state of Medchain.  

| Subcommands | Arguments                                                                                                                                                                                                                      | Description                                                                                                                                                                                                                                              |
|-------------|--------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| `sync`      | `bc` : the ByzCoin config (default is $BC), `keys` : the ed25519 private key that will sign the create query transaction                                                                                                       | Get the latest deferred transactions instance ids                                                                                                                                                                                                        |
| `sign`      | `bc` : the ByzCoin config (default is $BC), `keys` : the ed25519 private key that will sign the create query transaction, `id` : the instance id of the deffered transaction, `instidx` : the index of the instruction to sign | Sign a deferred transaction                                                                                                                                                                                                                              |
| `get`       | `bc` : the ByzCoin config (default is $BC), `keys` : the ed25519 private key that will sign the create query transaction, `id` : the instance id of the deffered transaction                                                   | Get the content of a deferred transaction                                                                                                                                                                                                                |
| `exec`      | `bc` : the ByzCoin config (default is $BC), `keys` : the ed25519 private key that will sign the create query transaction, `id` : the instance id of the deffered transaction                                                   | Execute the deferred transaction                                                                                                                                                                                                                         |
| `getexecid` | `bc` : the ByzCoin config (default is $BC), `keys` : the ed25519 private key that will sign the create query transaction, `id` : the instance id of the deffered transaction                                                   | Get the instance id of the executed deferred transaction. (Each time a new identity sign the transaction, the signature is included in the transaction and the final instance-id change, hence we need a method to get the result id from the execution) |

### project

     $ medadmin project subcommand [options] arguments

The project command manages the project access rights.  

| Subcommands | Arguments                                                                                                                                                                                                                                                                  | Description                                                                                                                            |
|-------------|----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|----------------------------------------------------------------------------------------------------------------------------------------|
| `attach`    | `bc` : the ByzCoin config (default is $BC), `keys` : the ed25519 private key that will sign the create query transaction, `id` :the instance id of the accessright contract                                                                                                | Attach the access right contract instance id to the project id with the naming contract *(need to be run only once per project setup)* |
| `add`       | `bc` : the ByzCoin config (default is $BC), `keys` : the ed25519 private key that will sign the create query transaction, `pdid` : the project darc id, `adid` : the admin darc id (default is $adid), `qid` : the querier id, `access` : the access rights of the querier | Add a new querier to the project. Returns the instance id of the deferred transaction.                                                 |
| `remove`    | `bc` : the ByzCoin config (default is $BC), `keys` : the ed25519 private key that will sign the create query transaction, `pdid` : the project darc id, `adid` : the admin darc id (default is $adid), `qid` : the querier id                                              | Removes the querier from the project. Returns the instance id of the deferred transaction.                                             |
| `modify`    | `bc` : the ByzCoin config (default is $BC), `keys` : the ed25519 private key that will sign the create query transaction, `pdid` : the project darc id, `adid` : the admin darc id (default is $adid), `qid` : the querier id, `access` : the access rights of the querier | Modify the querier access rights in the project. Returns the instance id of the deferred transaction.                                  |
| `verify`    | `bc` : the ByzCoin config (default is $BC), `keys` : the ed25519 private key that will sign the create query transaction, `pdid` : the project darc id, `adid` : the admin darc id (default is $adid), `qid` : the querier id, `access` : the access rights of the querier |                                                                                                                                        |
| `show`      | `bc` : the ByzCoin config (default is $BC), `keys` : the ed25519 private key that will sign the create query transaction, `pdid` : the project darc id, `adid` : the admin darc id (default is $adid), `qid` : the querier id                                              | Show the access rights of a user                                                                                                       |

`create` :	Create a new project structure (Create project darc, create access right)

| Subcommands   | Arguments                                                                                                                                                                                             | Description                                                                               |
|---------------|-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|-------------------------------------------------------------------------------------------|
| `darc`        | `bc` : the ByzCoin config (default is $BC), `keys` : the ed25519 private key that will sign the create query transaction, `adid` : the admin darc id (default is $adid), `pname` : the project name`  | Create a new project darc                                                                 |
| `accessright` | `bc` : the ByzCoin config (default is $BC), `keys` : the ed25519 private key that will sign the create query transaction, `pdid` : the project darc id, `adid` : the admin darc id (default is $adid) | Create a new accessright contract instance *(need to be run only once per project setup)* |


## Set up a Byzcoin chain

> All commands start with -c build. This is used to keep all configuration files generated in the build directory. We can then clean with make clean to remove all the generated files

### Deploy the chain 

First start a few cothority nodes:

```
path/to/medchain/cmd/medadmin$ make clean
path/to/medchain/cmd/medadmin$ cd build
path/to/medchain/cmd/medadmin/build$ ./run_nodes
```

Open a new shell.
	
Spawn a new byzcoin chain:

```
path/to/medchain/cmd/medadmin$ make spawn
go build
bcadmin --config build create build/public.toml | tail -n 1
export BC="build/bc-afef3830ae372be9d227a10b4b3c87a4661e2ba3a07f1e35002d07a0b5ad6b57.cfg"
```

For ease of use of the CLI store the configuration file path into the `$BC` environment variable:

    path/to/medchain/cmd/medadmin$ export BC="build/bc-afef3830ae372be9d227a10b4b3c87a4661e2ba3a07f1e35002d07a0b5ad6b57.cfg"

Get information about the deployed byzcoin chain using the bcadmin CLI:

```
path/to/medchain/cmd/medadmin$ bcadmin -c build info
- Config:
-- Roster:
--- tls://localhost:7774
--- tls://localhost:7772
--- tls://localhost:7770
-- ByzCoinID: afef3830ae372be9d227a10b4b3c87a4661e2ba3a07f1e35002d07a0b5ad6b57
-- AdminDarc: fa362de6ddc79c4bc1a636c557faf4e0be5685ca294f82b96992170f68aacc76
-- Identity: ed25519:d052769a6d7458b49559021a5a1d7ada609db08c470b45cce632040e535dcc99
- BC: build/bc-afef3830ae372be9d227a10b4b3c87a4661e2ba3a07f1e35002d07a0b5ad6b57.cfg
```

**ed25519:d052769a6d7458b49559021a5a1d7ada609db08c470b45cce632040e535dcc99** : is the id of the super admin (the very first admin spawn with the byzcoin chain). 

*For ease of use, you can store it in an environment variable*:

    path/to/medchain/cmd/medadmin$ export admin1=ed25519:d052769a6d7458b49559021a5a1d7ada609db08c470b45cce632040e535dcc99

Spawn an instance of a naming contract (used to do name resolution of instance id in Byzcoin): 

    path/to/medchain/cmd/medadmin$ bcadmin -c build contract name spawn 

------------

### Setup the Medchain administration darc


Create an administration darc 

```
path/to/medchain/cmd/medadmin$ ./medadmin -c build admin create darc --keys $admin1
New admininistration darc spawned :
- Darc:
-- Description: "Admin darc guards medchain project darcs"
-- BaseID: darc:3b4750793029dbfd493f943bb3729a5a54a4de9d3db25b4e446c04df090b6ca3
-- PrevID: darc:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
-- Version: 0
-- Rules:
--- _evolve - "ed25519:d052769a6d7458b49559021a5a1d7ada609db08c470b45cce632040e535dcc99"
--- _sign - "ed25519:d052769a6d7458b49559021a5a1d7ada609db08c470b45cce632040e535dcc99"
--- invoke:darc.evolve - "ed25519:d052769a6d7458b49559021a5a1d7ada609db08c470b45cce632040e535dcc99"
--- spawn:deferred - "ed25519:d052769a6d7458b49559021a5a1d7ada609db08c470b45cce632040e535dcc99"
--- invoke:deferred.addProof - "ed25519:d052769a6d7458b49559021a5a1d7ada609db08c470b45cce632040e535dcc99"
--- invoke:deferred.execProposedTx - "ed25519:d052769a6d7458b49559021a5a1d7ada609db08c470b45cce632040e535dcc99"
--- spawn:darc - "ed25519:d052769a6d7458b49559021a5a1d7ada609db08c470b45cce632040e535dcc99"
--- spawn:value - "ed25519:d052769a6d7458b49559021a5a1d7ada609db08c470b45cce632040e535dcc99"
--- _name:value - "ed25519:d052769a6d7458b49559021a5a1d7ada609db08c470b45cce632040e535dcc99"
--- invoke:value.update - "ed25519:d052769a6d7458b49559021a5a1d7ada609db08c470b45cce632040e535dcc99"
-- Signatures:
Admin darc base id: darc:3b4750793029dbfd493f943bb3729a5a54a4de9d3db25b4e446c04df090b6ca3
```

For ease of use of the CLI commands store the admin darc id into an environment variable `$adid`:

    path/to/medchain/cmd/medadmin$ export adid=darc:3b4750793029dbfd493f943bb3729a5a54a4de9d3db25b4e446c04df090b6ca3

Spawn the administrators' list.

*This list is used to keep a record of all known administrators currently registered inside the administration darc. This list is useful to create multi-signature rules*

```
path/to/medchain/cmd/medadmin$ ./medadmin -c build admin create list --keys $admin1 
Admins list spawned with id:
412b79a2c07811826b14ce42185f970cf5f7ea624ff7f170e564816fcbc151c6
``` 

Attach the instance id of the value contract that holds the admin's list to the admin darc:

*This name resolution is then used by the API to get values of the contract without providing the instance id.*

```
path/to/medchain/cmd/medadmin$ /medadmin -c build admin attach --id 412b79a2c07811826b14ce42185f970cf5f7ea624ff7f170e564816fcbc151c6 --keys $admin1
Successfully attached admins list to admin darc with name resolution : adminsList
``` 

**The Administration darc is setup**

-------

### How to manage admins in the admin darc

> Every admin operations needs to be signed to satisfy the multi-signature scheme defined (for now the rule state that every admin needs to sign)

#### Add/Remove/Modify an admin identity in the admin darc

Create a new admin identity:

```
path/to/medchain/cmd/medadmin$ ./medadmin -c build admin create admin
New admin identity key pair created :
ed25519:10a7f32004d03a252ddcc36d9bdcffe807cc5db911c0cef138b8d3f3b7beac87
```

*For ease of use, you can store it in an environment variable*:

    path/to/medchain/cmd/medadmin$ export admin2=ed25519:10a7f32004d03a252ddcc36d9bdcffe807cc5db911c0cef138b8d3f3b7beac87

Spawn a deferred transactionn to add the admin2 in the admin darc and the admin list:

```
path/to/medchain/cmd/medadmin$ ./medadmin -c build  admin add --keys $admin1 --identity $admin2
Deffered transaction (2 instructions) spawned with ID:
7c93c25c950c9627c64389186d3e986cda7018950f1016dcbcc4e40ef2b56c5a
```

Admin1 needs to sign two instructions in the transaction:

*One instruction change the admin darc to include the new admin, the other add the admin in the admin list store in a value contract bind to the admin darc*

```
path/to/medchain/cmd/medadmin$ ./medadmin -c build  deferred sign --keys $admin1 --id 7c93c25c950c9627c64389186d3e986cda7018950f1016dcbcc4e40ef2b56c5a --instidx 0
Succesfully added signature to deferred transaction
path/to/medchain/cmd/medadmin$ ./medadmin -c build  deferred sign --keys $admin1 --id 7c93c25c950c9627c64389186d3e986cda7018950f1016dcbcc4e40ef2b56c5a --instidx 1
Succesfully added signature to deferred transaction
```

Admin1 needs to execute the transaction:

```
path/to/medchain/cmd/medadmin$ ./medadmin -c build  deferred exec --keys $admin1 --id 7c93c25c950c9627c64389186d3e986cda7018950f1016dcbcc4e40ef2b56c5a
Succesfully executed the deferred transaction
```

To see the new admin darc, you can use the bcadmin CLI:

```
path/to/medchain/cmd/medadmin$ bcadmin -c build darc show --darc $adid
- Darc:
-- Description: "Admin darc guards medchain project darcs"
-- BaseID: darc:3b4750793029dbfd493f943bb3729a5a54a4de9d3db25b4e446c04df090b6ca3
-- PrevID: darc:3b4750793029dbfd493f943bb3729a5a54a4de9d3db25b4e446c04df090b6ca3
-- Version: 1
-- Rules:
--- _evolve - "ed25519:d052769a6d7458b49559021a5a1d7ada609db08c470b45cce632040e535dcc99 & ed25519:10a7f32004d03a252ddcc36d9bdcffe807cc5db911c0cef138b8d3f3b7beac87"
--- _sign - "ed25519:d052769a6d7458b49559021a5a1d7ada609db08c470b45cce632040e535dcc99 & ed25519:10a7f32004d03a252ddcc36d9bdcffe807cc5db911c0cef138b8d3f3b7beac87"
--- invoke:darc.evolve - "ed25519:d052769a6d7458b49559021a5a1d7ada609db08c470b45cce632040e535dcc99 & ed25519:10a7f32004d03a252ddcc36d9bdcffe807cc5db911c0cef138b8d3f3b7beac87"
--- spawn:deferred - "ed25519:d052769a6d7458b49559021a5a1d7ada609db08c470b45cce632040e535dcc99 | ed25519:10a7f32004d03a252ddcc36d9bdcffe807cc5db911c0cef138b8d3f3b7beac87"
--- invoke:deferred.addProof - "ed25519:d052769a6d7458b49559021a5a1d7ada609db08c470b45cce632040e535dcc99 | ed25519:10a7f32004d03a252ddcc36d9bdcffe807cc5db911c0cef138b8d3f3b7beac87"
--- invoke:deferred.execProposedTx - "ed25519:d052769a6d7458b49559021a5a1d7ada609db08c470b45cce632040e535dcc99 | ed25519:10a7f32004d03a252ddcc36d9bdcffe807cc5db911c0cef138b8d3f3b7beac87"
--- spawn:darc - "ed25519:d052769a6d7458b49559021a5a1d7ada609db08c470b45cce632040e535dcc99 & ed25519:10a7f32004d03a252ddcc36d9bdcffe807cc5db911c0cef138b8d3f3b7beac87"
--- spawn:value - "ed25519:d052769a6d7458b49559021a5a1d7ada609db08c470b45cce632040e535dcc99 & ed25519:10a7f32004d03a252ddcc36d9bdcffe807cc5db911c0cef138b8d3f3b7beac87"
--- _name:value - "ed25519:d052769a6d7458b49559021a5a1d7ada609db08c470b45cce632040e535dcc99 & ed25519:10a7f32004d03a252ddcc36d9bdcffe807cc5db911c0cef138b8d3f3b7beac87"
--- invoke:value.update - "ed25519:d052769a6d7458b49559021a5a1d7ada609db08c470b45cce632040e535dcc99 & ed25519:10a7f32004d03a252ddcc36d9bdcffe807cc5db911c0cef138b8d3f3b7beac87"
-- Signatures:
```

> All sensitive operations needs to comply with the multisignature rule chosen (for now every admin needs to sign) e.g : **_sign - "ed25519:d052769a6d7458b49559021a5a1d7ada609db08c470b45cce632040e535dcc99 & ed25519:10a7f32004d03a252ddcc36d9bdcffe807cc5db911c0cef138b8d3f3b7beac87"**, both admins needs to sign any operation delegated to the darc

To get the list of all admins identitites registered in the admin darc:

```
path/to/medchain/cmd/medadmin$ ./medadmin -c build admin get --keys $admin1
The list of admin identities in the admin darc:
[ed25519:d052769a6d7458b49559021a5a1d7ada609db08c470b45cce632040e535dcc99 ed25519:10a7f32004d03a252ddcc36d9bdcffe807cc5db911c0cef138b8d3f3b7beac87]
```

> `add`, `remove`, `modify` operations follow the same workflow
-------

### How to manage projects

create project 

add querier 
remove querier
modify 

verify 
show