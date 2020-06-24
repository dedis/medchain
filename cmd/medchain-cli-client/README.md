# MedChain Command-line-interface (CLI) Client 

MedChain supports two ways for clients to connect to the MedChain service:  
  - Go API Client
  - CLI
  
In this folder you will find the implementation of MedChain CLI client.

An [app](https://github.com/dedis/onet/tree/master/app), in the context of [Onet](https://github.com/dedis/onet), is a CLI-program that interacts with one or more Conodes through the use of the API defined by one or more services. 
We implemented the app for MedChain which is a client that talks to the service available in server (MedChain node). The code for this CLI-app is found in [cmd/medchain-cli-client](cmd/medchain-cli-client) directory of MedChain repository. CLI client can be used to interact with one or more MedChain nodes.

## MedChain Command-line-interface (CLI) Commands  

Table below summarizes the commands that are supported in MedChain CLI Client. Please note that in the table, `client_flags` refers to:

- `--bc: ByzCoin config file`
- `--file: MedChain group definition file`
- `--cid: ID of the client interacting with MedChain server`
- `--address: Address of server to contact`
- `--key:The ed25519 private key that will sign the transactions`

| Command | Description | Arguments|
| ------ | ------ | ------ |
| `create` | Create a MedChain CLI Client | `--client_flags` <br> `--qid`: The ID of query <br> `--did`: The ID of project darc <br>`--idfile`: File to save instance IDs |
| `query` | Submit a query for authorization | `--client_flags`  <br> `--instid`: Instance ID of query to sign |
|  `sign` | Add signature to a proposed query | `--client_flags` <br> `--instid`: Instance ID of query to verify |
| `verify` | Verify the status of a query  | `--client_flags` <br> `--instid`: Instance ID of query to execute|
| `exec` | Execute a proposed query | `--client_flags`  <br> `--instid`: Instance ID of query to execute |
| `get` | Get deferred data  | `--client_flags` <br> `--instid`: Instance ID of deferred data |
| `key` | Generate a new keypair | `--client_flags`  <br> `--save`: File to save key <br> `--print`: Print the private and public key |
| `darc` | Tool for managing Darcs (see Table for Darc subcommands | - |
| `fetch` | Fetch deferred query instance IDs  | `--client_flags` |

Table below summarizes the subcommands of `darc` command:

| Command | Description | Arguments|
| ------ | ------ | ------ |
| `show` | Show a DARC | `--client_flags` <br> `--darc`: ID of darc to show|
| `update` | Update the genesis Darc | `--client_flags`  <br> `--identity`: The identity of the signer |
|  `add` | Add a new project DARC| `--client_flags` <br> `--save`: Output file for Darc ID  <br> `--name`: Name of new DARC  |
| `rule` | Add signer to a rule or delete the rule  | `--client_flags` <br> `--darc`: ID of the DARC to update <br> `--name`: Name of DARC to update <br> `--rule`: Rule to which signer is added  <br> `--identity`: Identity of signer <br> `--type`: Type of rule to use  <br> `--delete`: Delete the rule|


## How to Use MedChain CLI Client

In order to use MedChain CLI client a network of at least 3 MedChain nodes should already be up and running. Please refer to [cmd/medchain-server](cmd/medchain-server) to learn more about MedChain nodes and how to deploy a network of them.
Once a network of MedChain nodes (i.e., MedChain servers) is up and running, the following commands can be used to use MedChain CLI client to interact with MedChain servers, submit a query to the network for authorization and demo query rejection and authorization scenarios.

Please note that all the below commands must be run in `cmd/medchain-cli-client`. 

In order to setup and start ByzCoin ledger, we run the following commands:

```bash
bash$ go build
bash$ mkdir medchain-config
bash$ ../medchain-server/run_nodes.sh -v 5 -n 3 -d ./medchain-config/
bash$ bcadmin --config medchain-config create medchain-config/group.toml | tail -n 1
bash$ bcadmin -c config info
```

We can export below environment variables so that we can more easily use them later:

```bash
bash$ export BC=medchain-config/bc-xxx.cfg
bash$ export admin1= ed25519:xxx
bash$ export adminDarc=....
bash$ export adrs1=tls://localhost:7770
bash$ export adrs2=tls://localhost:7772
bash$ export adrs3=tls://localhost:7774
bash$ export MEDCHAIN_GROUP_FILE_PATH=medchain-config/group.toml
```

Next, we  create keys for users 2 and 3:

```bash
bash$ ./medchain-cli-client key --save admin2.txt
bash$ export admin2=$(cat admin2.txt) 
bash$ ./medchain-cli-client key --save admin3.txt
bash$ export admin3=$(cat admin3.txt)
```

We can run the command below to check the genesis Darc:

```bash
bash$ ./medchain-cli-client darc show --bc $BC --file $MEDCHAIN_GROUP_FILE_PATH  --cid 1 --address $adrs1 --key $admin1 --darc $adminDarc
```

In order to create the first client we run:

```bash
bash$ ./medchain-cli-client create --bc $BC --file $MEDCHAIN_GROUP_FILE_PATH --cid 1 --address $adrs1 --key $admin1
```

Next, we add a default project Darc and call it "Project A Darc". Please note that this functionality is only enabled for test purposes. [medadmin](cmd/medadmin) is the main CLI tool to generate and manage MedChain admin and project Darcs. Please refer to [cmd/medadmin/README.md](cmd/medadmin/README.md) for further details. 

```bash
bash$ ./medchain-cli-client darc add --bc $BC --file $MEDCHAIN_GROUP_FILE_PATH  --cid 1 --address $adrs1 --key $admin1 --save darc_ids.txt --name A
bash$ export projectA=...
```

To create and start clients 2 and 3 we run:

```bash
bash$ ./medchain-cli-client create --bc $BC --file $MEDCHAIN_GROUP_FILE_PATH  --cid 2 --address $adrs2 --key $admin2
bash$ ./medchain-cli-client create --bc $BC --file $MEDCHAIN_GROUP_FILE_PATH  --cid 3 --address $adrs3 --key $admin3
```

Next, we need to add clients 2 and 3 as signers of project A Darc (for demo/test purposes):

```bash
bash$ ./medchain-cli-client  darc rule --bc $BC --file $MEDCHAIN_GROUP_FILE_PATH  
    --cid 1 --address $adrs1 --key $admin1 --id $projectA --name A 
    --rule spawn:medchain 
    --rule invoke:medchain.patient_list 
    --rule invoke:medchain.count_per_site
    --rule invoke:medchain.count_per_site_obfuscated
    --rule invoke:medchain.count_per_site_shuffled 
    --rule invoke:medchain.count_per_site_shuffled_obfuscated 
    --rule invoke:medchain.count_global 
    --rule invoke:medchain.count_global_obfuscated 
    --identity $admin2 --type AND

bash$ ./medchain-cli-client  darc rule --bc $BC --file$MEDCHAIN_GROUP_FILE_PATH  
    --cid 1 --address $adrs1 --key $admin1 --id $projectA 
    --name A --rule spawn:deferred 
    --rule invoke:medchain.update 
    --rule invoke:deferred.addProof 
    --rule invoke:deferred.execProposedTx 
    --rule spawn:darc --rule invoke:darc.evolve 
    --rule _name:deferred 
    --rule spawn:naming
    --rule _name:medchain 
    --rule spawn:value 
    --rule invoke:value.update --rule _name:value 
    --identity $admin2 --type OR
bash$ ./medchain-cli-client darc show --bc $BC --file $MEDCHAIN_GROUP_FILE_PATH--cid 3 --address $adr2 --key $admin2 --darc $projectA
```

We run the above commands also once for client 3, i.e., using `--identity \$admin3`. 

In order to submit query, we can use:

```bash
bash$ ./medchain-cli-client query --bc $BC --file $MEDCHAIN_GROUP_FILE_PATH --cid 1 --address $adrs1 --key $admin1 --qid  test:A:patient_list --darc $projectA --idfile InstIDs1.txt
export inst1=$(cat deferred_InstIDs1.txt)
```

Please note that a deferred instance ID is returned by server if the query is authorized.

To get the deferred query, one can run

```bash
bash$ ./medchain-cli-client get --bc $BC --file $MEDCHAIN_GROUP_FILE_PATH  --cid 1 --address $adrs1 --key $admin1 --instid $inst1
bash$ ./medchain-cli-client get --bc $BC --file $MEDCHAIN_GROUP_FILE_PATH  --cid 2 --address $adrs2 --key $admin2 --instid $inst1
bash$ ./medchain-cli-client get --bc $BC --file $MEDCHAIN_GROUP_FILE_PATH  --cid 3 --address $adrs3 --key $admin3 --instid $inst1
```

Furthermore, deferred query instances can be fetched by:

```bash
bash$ ./medchain-cli-client fetch --bc $BC --file $MEDCHAIN_GROUP_FILE_PATH  --cid 1 --address $adrs1 --key $admin1 
bash$ ./medchain-cli-client fetch --bc $BC --file $MEDCHAIN_GROUP_FILE_PATH  --cid 2 --address $adrs2 --key $admin2
bash$ ./medchain-cli-client fetch --bc $BC --file $MEDCHAIN_GROUP_FILE_PATH  --cid 3 --address $adrs3 --key $admin3 
```

Now, to sign the deferred query we use:

```bash
bash$ ./medchain-cli-client sign --bc $BC --file $MEDCHAIN_GROUP_FILE_PATH  --cid 2 --address $adrs2 --key $admin2 --instid $inst1
bash$ ./medchain-cli-client sign --bc $BC --file $MEDCHAIN_GROUP_FILE_PATH  --cid 3 --address $adrs3 --key $admin3 --instid $inst1
bash$ ./medchain-cli-client sign --bc $BC --file $MEDCHAIN_GROUP_FILE_PATH  --cid 1 --address $adrs1 --key $admin1 --instid $inst1
```

 Finally, to execute the transaction, we use:

```bash
    bash$ ./medchain-cli-client exec --bc $BC --file $MEDCHAIN_GROUP_FILE_PATH --cid 2 --address $adrs2 --key $admin2 --instid $inst1
 ```

Last but not least, one can use Docker to run a network of MedChain nodes as well as MedChain CLI client using docker-based deployment of MedChain. This is explained in details in [deployment](deployment/README.md).  

## Directory overview

Below is the description of code and files avaliable in this directory:

- `commands.go`: Definition of MedChain CLI client commands 
- `main.go`: Definition of MedChain CLI client functions (i.e., actions) used in `commands.go`  