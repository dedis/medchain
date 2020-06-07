package main

import (
	"fmt"
	"os"
	"time"

	"go.dedis.ch/cothority/v3"
	"go.dedis.ch/onet/v3/app"
	"go.dedis.ch/onet/v3/log"
	"go.dedis.ch/onet/v3/network"
	cli "gopkg.in/urfave/cli.v1"
)

const (
	// BinaryName represents the name of the binary
	BinaryName = "medchain-server"

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
			Name:  optionBCConfig + ", " + optionBCConfigShort,
			Usage: "Byzcoin config file",
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

	serverFlags := []cli.Flag{
		cli.StringFlag{
			Name:  optionConfig + ", " + optionConfigShort,
			Usage: "Configuration file of the server",
		},
	}

	nonInteractiveSetupFlags := []cli.Flag{
		cli.StringFlag{
			Name:  optionServerBinding + ", " + optionServerBindingShort,
			Usage: "Server binding address in the form of address:port",
		},
		cli.StringFlag{
			Name:  optionDescription + ", " + optionDescriptionShort,
			Usage: "Description of the node for the toml files",
		},
		cli.StringFlag{
			Name:  optionPrivateTomlPath + ", " + optionPrivateTomlPathShort,
			Usage: "Private toml file path",
		},
		cli.StringFlag{
			Name:  optionPublicTomlPath + ", " + optionPublicTomlPathShort,
			Usage: "Public toml file path",
		},
		cli.StringFlag{
			Name:  optionProvidedPubKey,
			Usage: "Provided public key (optional)",
		},
		cli.StringFlag{
			Name:  optionProvidedPrivKey,
			Usage: "Provided private key (optional)",
		},
	}

	cliApp.Commands = []cli.Command{
		// BEGIN CLIENT: Create a MedChain CLI Client ----------
		{
			Name:    "create",
			Aliases: []string{"c"},
			Usage:   "Create a MedChain CLI Client",
			Action:  create,
			Flags: append(clientFlags,
				cli.StringFlag{
					Name:  optionDarcID + ", " + optionDarcIDShort,
					Usage: "The DarcID that has the spawn:medchain rule (default is the genesis DarcID) ",
				}),
		},
		// CLIENT END: Create a MedChain CLI Client ------------

		// BEGIN CLIENT: SUBMIT DEFERRED QUERY ----------
		{
			Name:    "query",
			Aliases: []string{"q"},
			Usage:   "Submit a query for authorization",
			Action:  submitQuery,
			Flags: append(clientFlags,
				cli.StringFlag{
					Name:  optionQueryIDShort + ", " + optionQueryIDShort,
					Usage: "The ID of query as token:project_name:action ",
				}),
		},
		// CLIENT END: SUBMIT DEFERRED QUERY ------------

		// BEGIN CLIENT:  SIGN PROPOSED QUERY ----------
		{
			Name:    "sign",
			Aliases: []string{"s"},
			Usage:   "Add signature to a proposed query",
			Action:  addSignatureToDeferredQuery,
			Flags: append(clientFlags,
				cli.StringFlag{
					Name:  optionDarcID + ", " + optionDarcIDShort,
					Usage: "The DarcID of project darc governing the query ",
				},
				cli.StringFlag{
					Name:  optionQueryIDShort + ", " + optionQueryIDShort,
					Usage: "The ID of query as token:project_name:action ",
				},
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
			Aliases: []string{"d"},
			Usage:   "Execute a proposed query",
			Action:  execDefferedQuery,
			Flags:   clientFlags,
		},

		{
			Name:    "check",
			Aliases: []string{"c"},
			Usage:   "Check if the servers in the group definition are up and running",
			Action:  checkConfig,
			Flags: append(clientFlags,
				cli.BoolFlag{
					Name:  "detail, l",
					Usage: "Show details of all servers",
				}),
		},
		// CLIENT END: SIGN PROPOSED QUERY ------------

		// BEGIN SERVER --------
		{
			Name:  "server",
			Usage: "Start MedChain server",
			Action: func(c *cli.Context) error {
				if err := runServer(c); err != nil {
					return err
				}
				return nil
			},
			Flags: serverFlags,
			Subcommands: []cli.Command{
				{
					Name:    "setup",
					Aliases: []string{"s"},
					Usage:   "Setup server configuration (interactive)",
					Action: func(c *cli.Context) error {
						if c.String(optionConfig) != "" {
							return fmt.Errorf("[-] Configuration file option cannot be used for the 'setup' command")
						}
						if c.String(optionBCConfig) != "" {
							return fmt.Errorf("[-] Path to Byzcoin configuration file option cannot be used for the 'setup' command")
						}
						if c.GlobalIsSet("debug") {
							return fmt.Errorf("[-] Debug option cannot be used for the 'setup' command")
						}
						app.InteractiveConfig(cothority.Suite, BinaryName)

						return nil
					},
				},
				{
					Name:    "setupNonInteractive",
					Aliases: []string{"sni"},
					Usage:   "Setup server configuration (non-interactive)",
					Action:  NonInteractiveSetup,
					Flags:   nonInteractiveSetupFlags,
				},
			},
		},
		// SERVER END ----------
	}

	cliApp.Flags = binaryFlags
	cliApp.Before = func(c *cli.Context) error {
		log.SetDebugVisible(c.GlobalInt("debug"))
		return nil
	}
	err := cliApp.Run(os.Args)
	log.ErrFatal(err)
}
