package main

import cli "github.com/urfave/cli"

var cmds = cli.Commands{
	{
		Name:    "spawn",
		Usage:   "Spawn a new admin darc",
		Aliases: []string{"s"},
		Action:  spawn,
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "keys",
				Usage: "the ed25519 private key that will sign the create query transaction",
			},
			cli.StringFlag{
				Name:   "bc",
				EnvVar: "BC",
				Usage:  "the ByzCoin config",
			},
		},
	},
	{
		Name:  "admin",
		Usage: "Manage admins in admin darc",
		Subcommands: cli.Commands{
			{
				Name:    "create",
				Usage:   "Create a new admin",
				Aliases: []string{"c"},
				Action:  create,
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:   "bc",
						EnvVar: "BC",
						Usage:  "the ByzCoin config",
					},
				},
			},
			{
				Name:   "add",
				Usage:  "Add a new admin to the admin darc",
				Action: addAdmin,
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:   "bc",
						EnvVar: "BC",
						Usage:  "the ByzCoin config",
					},
					cli.StringFlag{
						Name:  "keys",
						Usage: "the ed25519 private key that will sign the create query transaction",
					},
					cli.StringFlag{
						Name:  "identity",
						Usage: "the new admin identity string",
					},
					cli.StringFlag{
						Name:  "adid",
						Usage: "the admin darc id",
					},
				},
			},
			{
				Name:   "remove",
				Usage:  "Remove an admin from the admin darc",
				Action: removeAdmin,
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:   "bc",
						EnvVar: "BC",
						Usage:  "the ByzCoin config",
					},
					cli.StringFlag{
						Name:  "keys",
						Usage: "the ed25519 private key that will sign the create query transaction",
					},
					cli.StringFlag{
						Name:  "identity",
						Usage: "the admin identity string",
					},
					cli.StringFlag{
						Name:  "adid",
						Usage: "the admin darc id",
					},
				},
			},
			{
				Name:   "modify",
				Usage:  "Modify the admin identity in the admin darc",
				Action: modifyAdminKey,
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:   "bc",
						EnvVar: "BC",
						Usage:  "the ByzCoin config",
					},
					cli.StringFlag{
						Name:  "keys",
						Usage: "the ed25519 private key that will sign the create query transaction",
					},
					cli.StringFlag{
						Name:  "oldkey",
						Usage: "the old admin identity string",
					},
					cli.StringFlag{
						Name:  "newkey",
						Usage: "the new admin identity string",
					},
					cli.StringFlag{
						Name:  "adid",
						Usage: "the admin darc id",
					},
				},
			},
		},
	},
	{
		Name:  "deferred",
		Usage: "Manage deferred transactions",
		Subcommands: cli.Commands{
			{
				Name:   "sync",
				Usage:  "Get the latest deferred transactions instance ids",
				Action: sync,
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:   "bc",
						EnvVar: "BC",
						Usage:  "the ByzCoin config",
					},
					cli.StringFlag{
						Name:  "keys",
						Usage: "the ed25519 private key that will sign the create query transaction",
					},
				},
			},
			{
				Name:   "sign",
				Usage:  "Sign a deferred transaction",
				Action: deferredSign,
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:   "bc",
						EnvVar: "BC",
						Usage:  "the ByzCoin config",
					},
					cli.StringFlag{
						Name:  "keys",
						Usage: "the ed25519 private key that will sign the create query transaction",
					},
					cli.StringFlag{
						Name:  "id",
						Usage: "the instance id of the deffered transaction",
					},
				},
			},
			{
				Name:   "get",
				Usage:  "Get the content of a deferred transaction",
				Action: deferredGet,
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:   "bc",
						EnvVar: "BC",
						Usage:  "the ByzCoin config",
					},
					cli.StringFlag{
						Name:  "id",
						Usage: "the instance id of the deffered transaction",
					},
				},
			},
			{
				Name:   "exec",
				Usage:  "Execute the deferred transaction",
				Action: deferredExec,
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:   "bc",
						EnvVar: "BC",
						Usage:  "the ByzCoin config",
					},
					cli.StringFlag{
						Name:  "keys",
						Usage: "the ed25519 private key that will sign the create query transaction",
					},
					cli.StringFlag{
						Name:  "id",
						Usage: "the instance id of the deffered transaction",
					},
				},
			},
			{
				Name:   "getexecid",
				Usage:  "Get the instance id of the executed deferred transaction",
				Action: getExecResult,
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:   "bc",
						EnvVar: "BC",
						Usage:  "the ByzCoin config",
					},
					cli.StringFlag{
						Name:  "keys",
						Usage: "the ed25519 private key that will sign the create query transaction",
					},
					cli.StringFlag{
						Name:  "id",
						Usage: "the instance id of the deffered transaction",
					},
				},
			},
		},
	},
	{
		Name:  "project",
		Usage: "Manage project darcs and access rights",
		Subcommands: cli.Commands{
			{
				Name:   "create",
				Usage:  "Create a new project",
				Action: projectCreate,
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:   "bc",
						EnvVar: "BC",
						Usage:  "the ByzCoin config",
					},
					cli.StringFlag{
						Name:  "keys",
						Usage: "the ed25519 private key that will sign the create query transaction",
					},
					cli.StringFlag{
						Name:  "adid",
						Usage: "the admin darc id",
					},
					cli.StringFlag{
						Name:  "pname",
						Usage: "the project name",
					},
				},
			},
			{
				Name:   "accessright",
				Usage:  "Create a new accessright contract instance",
				Action: projectCreateAccessRight,
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:   "bc",
						EnvVar: "BC",
						Usage:  "the ByzCoin config",
					},
					cli.StringFlag{
						Name:  "keys",
						Usage: "the ed25519 private key that will sign the create query transaction",
					},
					cli.StringFlag{
						Name:  "pdid",
						Usage: "the project darc id",
					},
					cli.StringFlag{
						Name:  "adid",
						Usage: "the admin darc id",
					},
				},
			},
			{
				Name:   "attach",
				Usage:  "Attach the access right contract instance id to the project id with the naming contract",
				Action: attach,
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:   "bc",
						EnvVar: "BC",
						Usage:  "the ByzCoin config",
					},
					cli.StringFlag{
						Name:  "keys",
						Usage: "the ed25519 private key that will sign the create query transaction",
					},
					cli.StringFlag{
						Name:  "id",
						Usage: "the instance id of the accessright contract",
					},
				},
			},
			{
				Name:   "add",
				Usage:  "Add a new querier to the project",
				Action: addQuerier,
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:   "bc",
						EnvVar: "BC",
						Usage:  "the ByzCoin config",
					},
					cli.StringFlag{
						Name:  "keys",
						Usage: "the ed25519 private key that will sign the create query transaction",
					},
					cli.StringFlag{
						Name:  "pdid",
						Usage: "the project darc id",
					},
					cli.StringFlag{
						Name:  "adid",
						Usage: "the admin darc id",
					},
					cli.StringFlag{
						Name:  "qid",
						Usage: "the querier id",
					},
					cli.StringFlag{
						Name:  "access",
						Usage: "the access rights of the querier",
					},
				},
			},
			{
				Name:   "remove",
				Usage:  "Removes the querier from the project",
				Action: removeQuerier,
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:   "bc",
						EnvVar: "BC",
						Usage:  "the ByzCoin config",
					},
					cli.StringFlag{
						Name:  "keys",
						Usage: "the ed25519 private key that will sign the create query transaction",
					},
					cli.StringFlag{
						Name:  "pdid",
						Usage: "the project darc id",
					},
					cli.StringFlag{
						Name:  "adid",
						Usage: "the admin darc id",
					},
					cli.StringFlag{
						Name:  "qid",
						Usage: "the querier id",
					},
				},
			},
			{
				Name:   "modify",
				Usage:  "Modify the querier access rights in the project",
				Action: modifyQuerier,
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:   "bc",
						EnvVar: "BC",
						Usage:  "the ByzCoin config",
					},
					cli.StringFlag{
						Name:  "keys",
						Usage: "the ed25519 private key that will sign the create query transaction",
					},
					cli.StringFlag{
						Name:  "pdid",
						Usage: "the project darc id",
					},
					cli.StringFlag{
						Name:  "adid",
						Usage: "the admin darc id",
					},
					cli.StringFlag{
						Name:  "qid",
						Usage: "the querier id",
					},
					cli.StringFlag{
						Name:  "access",
						Usage: "the new access rights",
					},
				},
			},
			{
				Name:   "verify",
				Usage:  "Verify the access rights of a user",
				Action: verifyAccess,
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:   "bc",
						EnvVar: "BC",
						Usage:  "the ByzCoin config",
					},
					cli.StringFlag{
						Name:  "keys",
						Usage: "the ed25519 private key that will sign the create query transaction",
					},
					cli.StringFlag{
						Name:  "pdid",
						Usage: "the project darc id",
					},
					cli.StringFlag{
						Name:  "qid",
						Usage: "the querier id",
					},
					cli.StringFlag{
						Name:  "access",
						Usage: "the access rights to check",
					},
				},
			},
		},
	},
}
