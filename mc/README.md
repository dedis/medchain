# The CLI for MedChain - mc

Here are some examples of how to use mc.

## Making a new key pair and creating a query

Using the `mc` tool, you can create a key pair:

```
$ mc key
```

The public key is printed on stdout. The private one is stored in the `app`
configuration directory. To use a custom configuration directory use the
`-config $dir`. You will give the public key to the ByzCoin administrator who
will use the "bcadmin darc rule" command to give your private key the right to
make new queries (add "spawn:queryContract" and "invoke:queryConctract.update" rules to a
Darc). In order to understand how to configure ByzCoin, more information can be 
found in the [bcadmin documentation](https://github.com/dedis/cothority/blob/master/byzcoin/bcadmin/README.md).

The ByzCoin admin will give you a ByzCoin config file, which you will use with
the -bc argument, or you can set the BC environment variable to the name of the
ByzCoin config file. He/she will also give you a DarcID to use. 

Assuming ByzCoin is configured with the correct permissions, you can now create
a query like this:

```
$ mc create -bc $file -darc $darcID -sign $key
```

A new query will be spawned, and the query ID will be printed. Set the
QUERY environment variable to communicate it to future calls to the `mc` program.
The $key variable is the key which you created using `mc key`.

## Creating Queries 

```
$ mc query -id <Query ID> -status <Status of Query> -sign $key
```

The above command creates a query. If `-id` is not set, it defaults to
the empty string. If `-status` is not set, `mc query` will read from stdin and adding those with the given `-id`.

An interesting test that creates and adds 100 queries, one every 0.1 second, so
that you can see the queries being added over the course of several
block creation epochs:

```
$ seq 100 | (while read i; do echo $i; sleep .1; done) | ./mc query
```

## Searching the Queries 

Searching through Medchain queries will be added soon.
The idea is to be able to search the queries based on their ID, 
Status, and Timestamp. This funtionality may look like below:

```
$ mc search -stat <Status> -from 12:00 -to 13:00 -count 5
```

The exit code will tell you if the search was successful or not.

If `-id` is not set, it will default to empty string. If you give
`-from`, then you must not give `-to`.

