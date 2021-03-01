# Client

This is a small example on how to use the Cothority javascript library to use
the MedChain service.

# Set up

## Run a Cothority

```sh
$ cd conode
$ go build -o conode && ./run_nodes.sh -v 3 -d tmp
```

## Build the app

First configure the correct roster in `src/roster.ts`. Then you can install the
dependencies, generate the proto file, and bundle the app:

```sh
$ cd gui
$ npm install
$ npm run protobuf
$ npm run bundle
```

You can now open `index.html`.

## Spawn a project

To spawn a project you'll need the ByzCoin ID, admin DARC, and the private admin
key.

You can get the Byzcoin ID and admin DARC with `bcadmin -c conode/tmp info`. You
can get the private admin key with:

```sh
bcadmin k --print ed25519:XXX  
```

Lastly, you need to add the `spawn:project` rule in the admin DARC:

```sh
# note the identity displayed with info:
$ bcadmin -c conode/tmp info
# add the identity in a new DARC rule:
$ bcadmin -c conode/tmp darc rule -rule spawn:project -id ed25519:XXX
```