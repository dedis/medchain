package main

import (
	"fmt"

	"github.com/urfave/cli"
	"go.dedis.ch/cothority/v3/byzcoin/bcadmin/clicontracts"
)

// PLEASE READ THIS
//
// In order to keep a consistant formatting please keep the following
// conventions:
//
// - Keep commands SORTED BY NAME
// - Use the following order for the arguments: Name, Usage, ArgsUsage, Action, Flags
// - "Flags" should always be the last argument

var cmds = cli.Commands{
	{
		Name: "contract",
		// Use space instead of tabs for correct formatting
		Usage: "Provides cli interface for contracts",
		Flags: []cli.Flag{
			cli.BoolFlag{
				Name:  "export, x",
				Usage: "redirects the transaction to stdout",
			},
		},
		// UsageText should be used instead, but its not working:
		// see https://github.com/urfave/cli/issues/592
		Description: fmt.Sprint(`
   		medchain-cli-client [--export] contract CONTRACT { 
                               spawn  --bc <byzcoin config> 
                                      [--<arg name> <arg value>, ...]
                                      [--darc <darc id>] 
                                      [--sign <pub key>],
                               invoke <command>
                                      --bc <byzcoin config>
                                      --instid, i <instance ID>
                                      [--<arg name> <arg value>, ...]
                                      [--darc <darc id>] 
                                      [--sign <pub key>],
                               get    --bc <byzcoin config>
                                      --instid, i <instance ID>,
                               delete --bc <byzcoin config>
                                      --instid, i <instance ID>
                                      [--darc <darc id>] 
                                      [--sign <pub key>]     
                             }
   		CONTRACT   {value,deferred,config}`),
		Subcommands: cli.Commands{
			{
				Name:  "deferred",
				Usage: "Manipulate a deferred contract",
				Subcommands: cli.Commands{
					{
						Name:   "spawn",
						Usage:  "spawn a deferred contract with the proposed transaction in stdin",
						Action: clicontracts.DeferredSpawn,
						Flags: []cli.Flag{
							cli.StringFlag{
								Name:   "bc",
								EnvVar: "BC",
								Usage:  "the ByzCoin config to use (required)",
							},
							cli.StringFlag{
								Name:  "darc",
								Usage: "DARC with the right to spawn a deferred contract (default is the admin DARC)",
							},
							cli.StringFlag{
								Name:  "sign",
								Usage: "public key of the signing entity (default is the admin public key)",
							},
						},
					},
					{
						Name:  "invoke",
						Usage: "invoke on a deferred contract ",
						Subcommands: cli.Commands{
							{
								Name:   "addProof",
								Usage:  "adds a signature and an identity on an instruction of the proposed transaction",
								Action: clicontracts.DeferredInvokeAddProof,
								Flags: []cli.Flag{
									cli.StringFlag{
										Name:   "bc",
										EnvVar: "BC",
										Usage:  "the ByzCoin config to use (required)",
									},
									cli.UintFlag{
										Name:  "instrIdx",
										Usage: "the instruction index of the transaction (starts from 0) (default is 0)",
									},
									cli.StringFlag{
										Name:  "hash",
										Usage: "the instruction hash that will be signed",
									},
									cli.StringFlag{
										Name:  "instid, i",
										Usage: "the instance ID of the deferred contract",
									},
									cli.StringFlag{
										Name:  "darc",
										Usage: "DARC with the right to invoke.addProof a deferred contract (default is the admin DARC)",
									},
									cli.StringFlag{
										Name:  "sign",
										Usage: "public key of the signing entity (default is the admin public key)",
									},
								},
							},
							{
								Name:   "execProposedTx",
								Usage:  "executes the proposed transaction if the instructions are correctly signed",
								Action: clicontracts.ExecProposedTx,
								Flags: []cli.Flag{
									cli.StringFlag{
										Name:   "bc",
										EnvVar: "BC",
										Usage:  "the ByzCoin config to use (required)",
									},
									cli.StringFlag{
										Name:  "instid, i",
										Usage: "the instance ID of the deferred contract",
									},
									cli.StringFlag{
										Name:  "darc",
										Usage: "DARC with the right to invoke.execProposedTx a deferred contract (default is the admin DARC)",
									},
									cli.StringFlag{
										Name:  "sign",
										Usage: "public key of the signing entity (default is the admin public key)",
									},
								},
							},
						},
					},
					{
						Name:   "get",
						Usage:  "if the proof matches, get the content of the given deferred instance ID",
						Action: clicontracts.DeferredGet,
						Flags: []cli.Flag{
							cli.StringFlag{
								Name:   "bc",
								EnvVar: "BC",
								Usage:  "the ByzCoin config to use (required)",
							},
							cli.StringFlag{
								Name:  "instid, i",
								Usage: "the instance id (required)",
							},
						},
					},
				},
			},
		},
	},
	{
		Name:    "darc",
		Usage:   "tool used to manage darcs",
		Aliases: []string{"d"},
		Subcommands: cli.Commands{
			{
				Name:   "show",
				Usage:  "Show a DARC",
				Action: darcShow,
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:   "bc",
						EnvVar: "BC",
						Usage:  "the ByzCoin config to use (required)",
					},
					cli.StringFlag{
						Name:  "darc",
						Usage: "the darc to show (admin darc by default)",
					},
				},
			},
			{
				Name:   "cdesc",
				Usage:  "Edit the description of a DARC",
				Action: darcCdesc,
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:   "bc",
						EnvVar: "BC",
						Usage:  "the ByzCoin config to use (required)",
					},
					cli.StringFlag{
						Name:  "darc",
						Usage: "the id of the darc to edit (config admin darc by default)",
					},
					cli.StringFlag{
						Name:  "sign, signer",
						Usage: "public key which will sign the request (default: the ledger admin identity)",
					},
					cli.StringFlag{
						Name:  "desc",
						Usage: "the new description of the darc (required)",
					},
				},
			},
			{
				Name:   "add",
				Usage:  "Add a new DARC with default rules.",
				Action: darcAdd,
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:   "bc",
						EnvVar: "BC",
						Usage:  "the ByzCoin config to use (required)",
					},
					cli.StringFlag{
						Name:  "sign, signer",
						Usage: "public key which will sign the DARC spawn request (default: the ledger admin identity)",
					},
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
						Name:  "desc",
						Usage: "the description for the new DARC (default: random)",
					},
					cli.BoolFlag{
						Name:  "shortPrint",
						Usage: "instead of printing the entire darc, prints the darc baseID in the first line and identity in the second one (optional)",
					},
				},
			},
			{
				Name:   "prule",
				Usage:  "print rule. Will print the rule given identities and a minimum to have M out of N rule",
				Action: darcPrintRule,
				Flags: []cli.Flag{
					cli.StringSliceFlag{
						Name:  "identity, id",
						Usage: "an identity, multiple use of this param is allowed. If empty it will create a new identity. Each provided identity is checked by the evaluation parser.",
					},
					cli.UintFlag{
						Name:  "minimum, M",
						Usage: "if this flag is set, the rule is computed to be \"M out of N\" identities. Otherwise it uses ANDs",
					},
				},
			},
		},
	},
	{
		Name:   "info",
		Usage:  "displays infos about the BC config",
		Action: getInfo,
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:   "bc",
				EnvVar: "BC",
				Usage:  "the ByzCoin config to use (required)",
			},
		},
	},
	{
		Name:  "instance",
		Usage: "displays infos about a query instance",
		Subcommands: cli.Commands{
			{
				Name:   "get",
				Usage:  "Display the content of a query instance",
				Action: getInstance,
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:   "bc",
						EnvVar: "BC",
						Usage:  "the ByzCoin config to use (required)",
					},
					cli.StringFlag{
						Name:  "instid, i",
						Usage: "the instance id (required)",
					},
					cli.BoolFlag{
						Name:  "hex",
						Usage: "if set, the data of the instance is hex encoded",
					},
				},
			},
		},
	},
	{
		Name:    "key",
		Usage:   "generates a new keypair and prints the public key in the stdout",
		Aliases: []string{"k"},
		Action:  key,
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
	},
	{
		Name:   "resolveiid",
		Usage:  "Resolves an instance id given a name and a darc id (using the ResolveInstanceID API call)",
		Action: resolveiid,
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:   "bc",
				EnvVar: "BC",
				Usage:  "the ByzCoin config to use (required)",
			},
			cli.StringFlag{
				Name:  "namingDarc",
				Usage: "the DARC ID that 'guards' the instance (default is the admin darc)",
			},
			cli.StringFlag{
				Name:  "name",
				Usage: "the name that was used to store the instance id (required)",
			},
		},
	},
}
