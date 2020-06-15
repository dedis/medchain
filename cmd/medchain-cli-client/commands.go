package main

import (
	"os"
	"time"

	"github.com/urfave/cli"
	"go.dedis.ch/onet/v3/log"
	"go.dedis.ch/onet/v3/network"
)

const (
	// BinaryName represents the name of the binary
	BinaryName = "medchain-cli-client"

	// Version of the binary
	Version = "1.00"

	// DefaultGroupFile is the name of the default file to lookup for group definition
	DefaultGroupFile = "group.toml"

	optionConfig      = "config"
	optionConfigShort = "c"

	optionBCConfig      = "bc-config"
	optionBCConfigShort = "bc"

	optionGroupFile      = "file"
	optionGroupFileShort = "f"

	// DefaultGroupInstanceIDFile is the name of the default file to lookup for submitted query instances
	DefaultGroupInstanceIDFile     = "instanceIDs.txt"
	optionGroupInstanceIDFile      = "idfile"
	optionGroupInstanceIDFileShort = "idf"

	optionClientID      = "client"
	optionClientIDShort = "cid"

	optionInstanceID      = "instid"
	optionInstanceIDShort = "iid"

	optionDarcID      = "darc"
	optionDarcIDShort = "did"

	optionQueryID      = "queryid"
	optionQueryIDShort = "qid"

	optionServerAddress      = "address"
	optionServerAddressShort = "adrs"

	optionKey      = "key"
	optionKeyShort = "k"

	// setup options
	optionServerBinding      = "serverBinding"
	optionServerBindingShort = "sb"

	optionDescription      = "description"
	optionDescriptionShort = "desc"

	optionPrivateTomlPath      = "privateTomlPath"
	optionPrivateTomlPathShort = "priv"

	optionPublicTomlPath      = "publicTomlPath"
	optionPublicTomlPathShort = "pub"

	optionProvidedPubKey = "pubKey"

	optionProvidedPrivKey = "privKey"

	optionProvidedSecrets      = "secrets"
	optionProvidedSecretsShort = "s"

	optionNodeIndex      = "nodeIndex"
	optionNodeIndexShort = "i"

	// RequestTimeOut defines when the client stops waiting for MedChain to reply
	RequestTimeOut = time.Second * 10
)

/*
Return system error codes signification
0: success
1: failed to init client
2: error in the query
*/
func main() {
	// increase maximum in onet.tcp.go to allow for big packets (for now is the max value for uint32)
	network.MaxPacketSize = network.Size(^uint32(0))

	cliApp := cli.NewApp()
	cliApp.Name = "medchain"
	cliApp.Usage = "Distributed authorization of medical queries"
	cliApp.Version = Version

	binaryFlags := []cli.Flag{
		cli.IntFlag{
			Name:  "debug, d",
			Value: 0,
			Usage: "debug-level: 1 for terse, 5 for maximal",
		},
	}

	clientFlags := []cli.Flag{
		cli.StringFlag{
			Name:   optionBCConfig + ", " + optionBCConfigShort,
			EnvVar: "BC",
			Usage:  "ByzCoin config file",
		},
		cli.StringFlag{
			Name:  optionGroupFile + ", " + optionGroupFileShort,
			Value: DefaultGroupFile,
			Usage: "MedChain group definition file",
		},
		cli.StringFlag{
			Name:  optionClientID + ", " + optionClientIDShort,
			Usage: "ID of the client interacting with MedChain server",
		},
		cli.StringFlag{
			Name:  optionServerAddress + ", " + optionServerAddressShort,
			Usage: "Address of server to contact",
		},
		cli.StringFlag{
			Name:  optionKey + ", " + optionKeyShort,
			Usage: "The ed25519 private key that will sign the transactions",
		},
	}

	cliApp.Commands = []cli.Command{
		// BEGIN CLIENT: Create a MedChain CLI Client ----------
		{
			Name:    "create",
			Aliases: []string{"cr"},
			Usage:   "Create a MedChain CLI Client",
			Action:  create,
			Flags:   clientFlags,
		},
		// CLIENT END: Create a MedChain CLI Client ------------

		// BEGIN CLIENT: SUBMIT DEFERRED QUERY ----------
		{
			Name:    "query",
			Aliases: []string{"q"},
			Usage:   "Submit a query for authorization",
			Action:  submitQuery,
			Flags: append(clientFlags, []cli.Flag{
				cli.StringFlag{
					Name:  optionQueryID + ", " + optionQueryIDShort,
					Usage: "The ID of query as token:project_name:action ",
				},
				cli.StringFlag{
					Name:  optionDarcID + ", " + optionDarcIDShort,
					Usage: "The ID of project darc associated with query ",
				},
				cli.StringFlag{
					Name:  optionGroupInstanceIDFile + ", " + optionGroupInstanceIDFileShort,
					Usage: "The name of file to save instance IDs ",
				},
			}...),
		},
		// CLIENT END: SUBMIT DEFERRED QUERY ------------

		// BEGIN CLIENT:  SIGN PROPOSED QUERY ----------
		{
			Name:    "sign",
			Aliases: []string{"s"},
			Usage:   "Add signature to a proposed query",
			Action:  addSignature,
			Flags: append(clientFlags,
				cli.StringFlag{
					Name:  optionInstanceID + ", " + optionInstanceIDShort,
					Usage: "The instance ID of query to add signature to ",
				}),
		},
		// CLIENT END: SIGN PROPOSED QUERY ------------

		// BEGIN CLIENT: VERIFY QUERY STATUS ----------
		{
			Name:    "verify",
			Aliases: []string{"v"},
			Usage:   "Verify the status of a query",
			Action:  verifyStatus,
			Flags:   clientFlags,
		},
		// CLIENT END: VERIFY QUERY STATUS ------------

		// BEGIN CLIENT:  EXECUTE PROPOSED QUERY ----------
		{
			Name:    "exec",
			Aliases: []string{"e"},
			Usage:   "Execute a proposed query",
			Action:  execDefferedQuery,
			Flags: append(clientFlags,
				cli.StringFlag{
					Name:  optionInstanceID + ", " + optionInstanceIDShort,
					Usage: "The instance ID of query to execute",
				}),
		},
		// CLIENT END: EXECUTE PROPOSED QUERY  ------------

		// BEGIN CLIENT: GET DEFERRED DATA  ----------
		{
			Name:    "get",
			Aliases: []string{"g"},
			Usage:   "Get deferred data",
			Action:  getQuery,
			Flags: append(clientFlags,
				cli.StringFlag{
					Name:  optionInstanceID + ", " + optionInstanceIDShort,
					Usage: "The instance ID of deferred data to retrieve ",
				}),
		},
		// CLIENT END: GET DEFERRED DATA  ------------

		// BEGIN CLIENT: FETCH DEFERRED QUERY INSTANCE IDs ----------
		{
			Name:    "fetch",
			Aliases: []string{"f"},
			Usage:   "Fetch deferred query instance IDs",
			Action:  fetchInstanceIDs,
			Flags:   clientFlags,
		},
		// CLIENT END: FETCH DEFERRED QUERY INSTANCE IDs ------------

		// BEGIN CLIENT: CREATE KEY ----------
		{
			Name:    "key",
			Usage:   "Generate a new keypair and print the public key in the stdout",
			Aliases: []string{"k"},
			Flags: []cli.Flag{
				cli.StringFlag{
					Name:  "save",
					Usage: "file in which the user wants to save the public key instead of printing it",
				},
				cli.StringFlag{
					Name:  "print",
					Usage: "print the private and public key",
				},
			},
			Action: createKey,
		},
		// CLIENT END: CREATE KEY  ------------
		// BEGIN CLIENT: MANIPULATE DARCS ----------
		{
			Name:    "darc",
			Usage:   "Tool used to manage project darcs",
			Aliases: []string{"d"},
			Subcommands: cli.Commands{
				{
					Name:   "show",
					Usage:  "Show a DARC",
					Action: darcShow,
					Flags: append(clientFlags,
						cli.StringFlag{
							Name:  "darc",
							Usage: "The ID of darc to show (admin darc by default)",
						}),
				},
				{
					Name:   "update",
					Usage:  "Update the genesis darc",
					Action: updateGenesisDarc,
					Flags: append(clientFlags,
						cli.StringSliceFlag{
							Name:  "identity, signer_id",
							Usage: "The identity of the signer who will be allowed to use the genesis darc rules. multiple use of this param is allowed",
						}),
				},
				{
					Name:   "add",
					Usage:  "Add a new project DARC with default rules.",
					Action: addProjectDarc,
					Flags: append(clientFlags, []cli.Flag{
						cli.StringFlag{
							Name:  "save",
							Usage: "Output file for the darc id (optional)",
						},
						cli.StringFlag{
							Name:  "name",
							Usage: "The name for the new DARC (default: random)",
						},
					}...),
				},
				{
					Name:   "rule",
					Usage:  "Add signer to a project rule or delete the rule.",
					Action: addSigner,
					Flags: append(clientFlags, []cli.Flag{
						cli.StringFlag{
							Name:  "darc",
							Usage: "The ID of the DARC to update",
						},
						cli.StringFlag{
							Name:  "name",
							Usage: "The name of the DARC to update",
						},
						cli.StringSliceFlag{
							Name:  "rule",
							Usage: "The rule to which signer is added. multiple use of this is allowed except for --delete",
						},
						cli.StringFlag{
							Name:  "identity, signer_id",
							Usage: "The identity of the signer who will be allowed to use the rule.",
						},
						cli.StringFlag{
							Name:  "type, t",
							Usage: "Type of rule to use. Either AND or OR ",
						},
						cli.BoolFlag{
							Name:  "delete",
							Usage: "Delete the rule",
						},
					}...),
				},
			},
		},
		// CLIENT END: MANIPULATE DARCS ----------
	}

	cliApp.Flags = binaryFlags
	cliApp.Before = func(c *cli.Context) error {
		log.SetDebugVisible(c.Int("debug"))
		return nil
	}
	err := cliApp.Run(os.Args)
	log.ErrFatal(err)
}
