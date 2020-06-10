
package main

import "github.com/urfave/cli"

const(
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

	DefaultGroupInstanceIDFile     = "instanceIDs.txt"
	optionGroupInstanceIDFile      = "idfile"
	optionGroupInstanceIDFileShort = "idf"

	// DefaultGroupFile = "group.toml"

	optionConfig      = "config"
	optionConfigShort = "c"

	optionBCConfig      = "bc-config"
	optionBCConfigShort = "bc"

	optionGroupFile      = "file"
	optionGroupFileShort = "f"

)	

var clientFlags = []cli.Flag{
	cli.StringFlag{
		Name:  optionBCConfig + ", " + optionBCConfigShort,
		Usage: "Byzcoin config file",
	},
	// cli.StringFlag{
	// 	Name:  optionGroupFile + ", " + optionGroupFileShort,
	// 	Value: DefaultGroupFile,
	// 	Usage: "MedChain group definition file",
	// },
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

var cmds = cli.Commands{
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

	{
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
	},
	{
		Name:    "darc",
		Usage:   "tool used to manage project darcs",
		Aliases: []string{"d"},
		Subcommands: cli.Commands{
			{
				Name:  "show",
				Usage: "Show a DARC",
				Action: darcShow,
				Flags: append(clientFlags,[]cli.Flag{
					cli.StringFlag{
						Name:  "darc",
						Usage: "the darc to show (admin darc by default)",
					},
				}...),
			},
			{
				Name:   "add",
				Usage:  "Add a new project DARC with default rules.",
				Action: addProjectDarc,
				Flags: append(clientFlags,[]cli.Flag{
					cli.StringFlag{
						Name:  "darc",
						Usage: "DARC with the right to create a new DARC (default is the admin DARC)",
					},
					cli.StringSliceFlag{
						Name:  "identity, id",
						Usage: "an identity, multiple use of this param is allowed. If empty it will create a new identity. Each provided identity is checked by the evaluation parser.",
					},
					cli.BoolFlag{
						Name:  "unrestricted",
						Usage: "add the invoke:evolve_unrestricted rule",
					},
					cli.BoolFlag{
						Name:  "deferred",
						Usage: "adds rules related to deferred contract: spawn:deferred, invoke:deferred.addProof, invoke:deferred.execProposedTx",
					},
					cli.StringFlag{
						Name:  "out_id",
						Usage: "output file for the darc id (optional)",
					},
					cli.StringFlag{
						Name:  "out_key",
						Usage: "output file for the darc key (optional)",
					},
					cli.StringFlag{
						Name:  "name",
						Usage: "the name for the new DARC (default: random)",
					},
				}...),
			},
			{
				Name:  "prule",
				Usage: "print rule. Will print the rule given identities and a minimum to have M out of N rule",
				//Action: darcPrintRule,
				Flags: append(clientFlags,[]cli.Flag{
					cli.StringSliceFlag{
						Name:  "identity, id",
						Usage: "an identity, multiple use of this param is allowed. If empty it will create a new identity. Each provided identity is checked by the evaluation parser.",
					},
					cli.UintFlag{
						Name:  "minimum, M",
						Usage: "if this flag is set, the rule is computed to be \"M out of N\" identities. Otherwise it uses ANDs",
					},
				}...),
			},
			{
				Name:   "rule",
				Usage:  "Edit project DARC rules.",
				Action: darcRule,
				Flags: append(clientFlags,[]cli.Flag{
					cli.StringFlag{
						Name:  "id",
						Usage: "the ID of the DARC to update (default is the admin DARC)",
					},
					cli.StringFlag{
						Name:  "name",
						Usage: "the name of the DARC to update (default is the admin DARC)",
					},
					cli.StringFlag{
						Name:  "rule",
						Usage: "the rule to be added, updated or deleted",
					},
					cli.StringSliceFlag{
						Name:  "identity, signer_id",
						Usage: "the identity of the signer who will be allowed to use the rule. Multiple use of this param is allowed. Each identity is checked by the evaluation parser.",
					},
					cli.BoolFlag{
						Name:  "replace",
						Usage: "if this rule already exists, replace it with this new one",
					},
					cli.BoolFlag{
						Name:  "delete",
						Usage: "delete the rule",
					},
				}...),
			},
		},
	},
	{
		Name:    "key",
		Usage:   "generates a new keypair and prints the public key in the stdout",
		Aliases: []string{"k"},
		Action:  key,
		Flags: append(clientFlags,[]cli.Flag{
			cli.StringFlag{
				Name:  "save",
				Usage: "file in which the user wants to save the public key instead of printing it",
			},
			cli.StringFlag{
				Name:  "print",
				Usage: "print the private and public key",
			},
		}...),
	},
}

