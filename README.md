# MedChain

MedChain is an authorization and access management service for medical queries.
It uses a collective authority to manage authorization of queries, which
provides full Auditability and distribution of power among participants. It is
based on [Cothority](https://github.com/dedis/cothority/blob/master/README.md),
[ByzCoin](https://github.com/dedis/cothority/blob/master/byzcoin/README.md), and
[Darcs](https://github.com/dedis/cothority/blob/master/darc/README.md).

[![Go test](https://github.com/ldsec/medchain/actions/workflows/go_test.yml/badge.svg)](https://github.com/ldsec/medchain/actions/workflows/go_test.yml)
[![Go lint](https://github.com/ldsec/medchain/actions/workflows/go_lint.yml/badge.svg)](https://github.com/ldsec/medchain/actions/workflows/go_lint.yml)

This repos contains the smart contracts (`/contracts`), deployment code
(`/conode`, `/bypros`), and a javascript example that implements a simple GUI
(`/gui`).

MedChain is used in combination with
[MedChain-front-end](https://github.com/dedis/medchain-frontend), a web-based
GUI for administrators.

Read the [End-of-project technical
report](https://github.com/dedis/medchain/releases/download/v1.0/MedChain.Technical.report.pdf).

# Smart contracts

Access on resources are expressed by instances of the **query** smart contract.
An instance of the query smart contract contains the request from a user to
perform some queries on a particular project. A project is the definition of 1
or more datasets.

Projects are defined with instances of the **project** smart contract. The
project smart contract defines the authorizations users have on its data. Each
instances of the project smart contract are responsible for spawning instances
of the **query** smart contract. This allows each project's instances to tell if
the query on its data is valid or not. Upon request to create an instance of a
query, the project instance will check if the user is allowed to perform that
query, based on the authorizations it contains. If allowed, the project instance
will create the query instance and set its status to **pending**. If not, it
still creates the query but sets the status of the query to **rejected**.

Instances of the project smart contract are controlled by the **DARC admin**,
which uses threshold rules to guard the actions on the project instances, ie.
creating new project instances, and updating authorizations on projects.

The DARC admin is itself managed by the genesis DARC, which is created at the
creation of the chain.

The following illustration summarizes the above explanations:

![contracts](contracts/smart_contracts.png)

# Run postgres for the Byzcoin proxy

The Byzcoin proxy (bypros) keeps a representation of the blockchain in an SQL
database. It allows for fast queries in situation where we would normally need
to browse all the blocks from the chain. That would be the case for example if
one wants to get all the pending deferred transactions.

From the `bypros/` folder run the docker-compose:

```sh
# use -d to launch as a daemon
docker-compose up
```

Then, when you launch the conode, export the needed variables for the proxy. The
proxy needs two URLs to connect to the database: one with read/write, and
another one with read-only access.

```sh
# user/password are set in the dockerfile (root) and the schema (read-only user).
export PROXY_DB_URL=postgres://bypros:docker@localhost:5432/bypros
export PROXY_DB_URL_RO=postgres://proxy:1234@localhost:5432/bypros
```

# Run a Cothority

You can run a set of nodes by running the following:

```sh
cd conode
# it will fail if the PROXY_DB_URL* variables are not set!
go build -o conode && ./run_nodes.sh -v 3 -d tmp
```

This command will setup 3 nodes and save their files in conode/tmp.

Once the nodes are running, you may want to create a new skipchain, and perform
basic operations like updating the DARC. This can be done with
[bcadmin](https://github.com/dedis/cothority/tree/master/byzcoin/bcadmin), the
Byzcoin CLI:

```sh
# Tells bcadmin where the config folder is
BC_CONFIG=conode/tmp 
# Create a new skipchain
bcadmin create $BC_CONFIG/public.toml
# Tells bcadmin about the new skipchain configuration
export BC=...
# Print the skipchain info, useful to perform some operations later on
bcadmin info
# Print the admin key, which is stored at the same place specified by BC
bcadmin key -print .../key-ed25519\:...
# Add a DARC rule, the id can be found with 'bcadmin info'
bcadmin darc rule -rule spawn:project -id ed25519:...
```

# Run the GUI demo

The GUI demo is a static webpage that uses typescript and webpack to write and
build the assets. You first need to update `gui/src/roster.ts` to the correct
roster, which can be found in `conode/tmp/public.toml` if you followed the
instructions above.

From the `gui/` folder:

```sh
npm install
npm run protobuf
nom run bundle
```

You can now open `index.html`:

![demo](gui/gui.png)

# Run a production node

Those instructions follow closely the deployment of a cothority node. You can
get more in-depth instruction from the repository documentation:
https://github.com/dedis/cothority/blob/main/conode/README.md.

In this setup we are using docker to configure and deploy a conode. First, build
the docker image:

```sh
# from the root folder
docker build -t medchain:latest .
```

Once the docker image is built, setup the conode with:

```sh
# store `conode_data` in a location of your choice
# this will prompt a configuration setup. You can use the default options.
docker run --rm -it -p 7770-7771:7770-7771 \
    -v /absolute/path/to/conode_data:/conode_data medchain:latest \
    ./conode setup
```

Finally, run the conode with the following command. Be sure you have started
bypros before:

```sh
docker run --restart always -p 7770-7771:7770-7771 \
    -e PROXY_DB_URL=postgres://bypros:docker@localhost:5432/bypros \
    -e PROXY_DB_URL_RO=postgres://proxy:1234@localhost:5432/bypros \
    -v /Users/nkocher/GitHub/medchain/conode_data:/conode_data \
    medchain:latest
```

You will have a conode running locally on `127.0.0.1:7770`. It also exposes
address `127.0.0.1:7771` for client requests. Those requests will come from the
Medchain-frontend.

You should reverse-proxy the client requests with a web-server. We recommend
nginx:

```
location / {
   server_name example.com;
   # ...
   ssl_certificate /etc/nginx/ssl/example.com.certificate.pem;
   ssl_certificate_key /etc/nginx/ssl/example.com.key.pem;
   # ...
   location /conode/ {
      proxy_http_version 1.1;

      proxy_set_header Upgrade $http_upgrade;
      proxy_set_header Connection "upgrade";

      proxy_pass "http://localhost:7771/";
   }
}
```