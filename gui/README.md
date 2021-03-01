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