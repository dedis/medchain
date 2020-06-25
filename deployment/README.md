# MedChain Docker-based Deployment 

In order to facilitate the deployment of (multi-node) MedChain network, docker-based deployment of it is enabled that uses docker images built for both MedChain server node and MedChain CLI client.

Docker-based deployment of MedChain is located in this directory. Below, is the description of files found here:

- `Dockerfile`: Contains the commands needed to assemble MedChain node docker image
- `client.Dockerfile`: Dockerfile to build MedChain-cli-client docker image
- `docker-compose.yaml`: Multi-container definition of MedChain node and MedChain CLI client
- `docker-entrypoint.sh`: Docker entrypoint script for both MedChain node and MedChain CLI client containers
- `docker-compose-demo.yaml`: Multi-container definition of a network of 3 MedChain nodes and 1 MedChain CLI client used in demo.

## Use Docker to Run MedChain

Docker can be used to run a MedChain node and its CLI client or a network of multiple MedChain nodes and CLI clients. 

To setup and run a single MedChain node and a single MedChain CLI client, and/or to build their docker images, one can run the following commands in this directory:

```bash
bash$ docker-compose -f docker-compose.yaml up --build
```

This command will build MedChain server and CLI client images, setup and run the server in a container, and create a MedChain CLI client container that can interact with running server. Please note that flag `--build` is only necessary if the user needs to build docker images for MedChain node and its CLI client.


Please note that before we can setup and run MedChain nodes using `docker-compose.yaml` file, we need to have the `private.toml` file of every MedChain node (server) in `deployment/medchain-config/mcX` directory where X corresponds to node index. We can use the script `run_nodes.sh` provided in [medchain-server](../cmd/medchain-server) folder to setup as many MedChain nodes as we want. For example, to setup 3  MedChain nodes, we can use the command below:

```bash
bash$ mkdir medchain-config
bash$ ../cmd/medchain-server/run_nodes.sh -v 5 -n 3 -d ./medchain-config/
```
Once all the servers are up and running, we need to use MedChain CLI client container and run the commands in it, to this end we can use:

```bash
bash$ docker exec -it <name_of_cli_client_container> bash
```

We can, then, use the commands described in MedChain CLI client [README](../cmd/medchain-cli-client/README.md) file to interact with MedChain node through the CLI.  

To setup and run a multi-node network, one can define their own network in a docker-compose file and run it using:

```bash
bash$ docker-compose -f <docker-compose file path> up  
```

**Important point**: Once the network is up and running, we need to update the `private.toml` file of each node as well as `group.toml` file with the IP address of corresponding docker containers. To get the IP address of each container, we can use:

```bash
bash$ docker network inspect <name of MedChain network>
```
 