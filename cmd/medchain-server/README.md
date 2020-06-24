# MedChain Node (MedChain Server)

MedChain node is a [Conode](https://github.com/dedis/cothority/tree/master/conode) that has specific Medchain services and protocols available in it.The  main  MedChain  node  (i.e.,  server)  code  can  be  found in `medchain.go`. Table below shows the commands (subcommands of server) that are supported in MedChain node.


| Command | Description | Arguments|
| ------ | ------ | ------ |
|`server`| Run MedChain server | `--c`: Server configuration file <br> `--d`: debug-level|
|`server setup`| Setup MedChain server | -
|`server setupNonInteractive` | Setup server non-interactively  |`--sb`: server binding <br> `--desc`: node description <br> `--priv`: Private toml file path <br> `--pub`: Public toml file path <br> `--privKey`: Provided private key <br> `--pubKey`: Provided public key |
|`server config` | Check servers in roster  | `--g`: Group definition file <br> `--t`: Set a different timeout|


## MedChain Node: Setup and Run

The easiest way to setup and run multiple MedChain nodes (locally and as a shell process) is to use the provided shell script `run_nodes.sh` provided in this directory. For example, the following code can be executed in order to run 3 MedChain nodes :

```bash
path/to/medchain/cmd/medchain-server$ go build
path/to/medchain/cmd/medchain-server$ run_nodes.sh -n 3 -d medchain-config
```    
The above code, will setup and start 3 MedChain nodes and will put all configuration files in `./medchain-config/` and the file containing all the public configurations will be in `./medchain-config/group.toml`. The folders created in `./medchain-config/` (i.e., `mc1/`, `mc2/`, ...) each hold private keys of a server in `private.toml`. The servers will then be listening to ports. It is important to note that each MedChain node uses two ports. MedChain-MedChain communication is automatically secured via TLS when you use the unchanged configuration from MedChain server setup. However, MedChain-client communication happens on the next port up from MedChain-MedChain port and it defaults to WebSockets inside of HTTP.

Additionally, it is possible to run MedChain nodes one at a time and without the mentioned script. For this end, the command below can be used to run a single MedChain node:
```bash
path/to/medchain/cmd/medchain-server$ go build
path/to/medchain/cmd/medchain-server$ ./medchain-server server setup
path/to/medchain/cmd/medchain-server$ ./medchain-server server setup
```

After the above commands are run, MedChain node is setup. Now, to run the server, the following can be used:
```bash
path/to/medchain/cmd/medchain-server$ ./medchain-server server setup
path/to/medchain/cmd/medchain-server$ ./medchain-server server
```

Last but not least, one can use Docker to run a single MedChain server or a network of them. This is implemenmted and explained in details in [deployment](../../deployment)

## MedChain Node: Test

To test the server code using Go tests, you can use the following command:

```bash
path/to/medchain/cmd/medchain-server$ go test 
```

## Directory overview

Below is the description of code and files avaliable in this directory

- `libtest.sh`: Bash script that contains code that can be used to test MedChain nodes
- `medchain.go`: main MedChain server code
- `medchain_test.go`: Server Go tests
- `medchain-server.go`: Definition of server functions (i.e., actions) used in `medchain.go`
- `non_interactive_setup.go`": Code used to setup the server non-interactively.
- `run_nodes.sh`: Bash script that can be used to setup and run multiple MedChain nodes locally.
- `server_test.go`: Server Go test (with logging)