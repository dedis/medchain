package main

import (
	"bufio"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"strings"
	"time"

	medchain "github.com/medchain/contract"
	cli "github.com/urfave/cli"
	"go.dedis.ch/cothority/byzcoin/bcadmin/lib"
	"go.dedis.ch/cothority/darc/expression"
	"go.dedis.ch/cothority/v3"
	"go.dedis.ch/cothority/v3/byzcoin"
	bcadminlib "go.dedis.ch/cothority/v3/byzcoin/bcadmin/lib"
	"go.dedis.ch/cothority/v3/darc"
	"golang.org/x/xerrors"

	"go.dedis.ch/cothority/v3/skipchain"
	"go.dedis.ch/onet/v3"
	"go.dedis.ch/onet/v3/cfgpath"
	"go.dedis.ch/onet/v3/log"
	"go.dedis.ch/onet/v3/network"
	"go.dedis.ch/protobuf"
)

// PLEASE READ THIS
//
// In order to keep a consistant formatting please keep the following
// conventions:
//
// - Keep commands SORTED BY NAME
// - Use the following order for the arguments: Name, Usage, ArgsUsage, Action, Flags
// - "Flags" should always be the last argument

type config struct {
	Name    string
	QueryID byzcoin.InstanceID //
}

type bcConfig struct {
	Roster    onet.Roster
	ByzCoinID skipchain.SkipBlockID
}

var cmds = cli.Commands{
	{
		Name:    "create",
		Usage:   "create a MedChain CLI app",
		Aliases: []string{"c"},
		Action:  create,
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "sign",
				Usage: "the ed25519 private key that will sign the create query transaction",
			},
			cli.StringFlag{
				Name:   "bc",
				EnvVar: "BC",
				Usage:  "the ByzCoin config",
			},
			cli.StringFlag{
				Name:  "darc",
				Usage: "the DarcID that has the spawn:queryContract rule (default is the genesis DarcID)",
			},
		},
	},

	{
		Name:    "query",
		Usage:   "create one or more queries",
		Aliases: []string{"q"},
		Action:  createQuery,
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:  "sign",
				Usage: "the ed25519 private key that will sign transactions",
			},
			cli.StringFlag{
				Name:   "bc",
				EnvVar: "BC",
				Usage:  "the ByzCoin config",
			},
			cli.StringFlag{
				Name:   "mc",
				EnvVar: "MC",
				Usage:  "the MedChain id, from \"mc create\"",
			},
			cli.StringFlag{
				Name:  "id",
				Usage: "the ID of the query",
			},
			cli.StringFlag{
				Name:  "status, stat",
				Usage: "the status of the query",
			},
			cli.IntFlag{
				Name:  "wait, w",
				Usage: "wait for block inclusion (default: do not wait)",
				Value: 0,
			},
		},
	},
	{
		Name:    "darc",
		Usage:   "tool used to manage project darcs",
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
			{
				Name:   "rule",
				Usage:  "Edit DARC rules.",
				Action: darcRule,
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:   "bc",
						EnvVar: "BC",
						Usage:  "the ByzCoin config to use (required)",
					},
					cli.StringFlag{
						Name:  "darc",
						Usage: "the DARC to update (default is the admin DARC)",
					},
					cli.StringFlag{
						Name:  "sign",
						Usage: "public key of the signing entity (default is the admin public key)",
					},
					cli.StringFlag{
						Name:  "rule",
						Usage: "the rule to be added, updated or deleted",
					},
					cli.StringSliceFlag{
						Name:  "identity, id",
						Usage: "the identity of the signer who will be allowed to use the rule. Multiple use of this param is allowed. Each identity is checked by the evaluation parser.",
					},
					cli.UintFlag{
						Name:  "minimum, M",
						Usage: "if this flag is set, the rule is computed to be \"M out of N\" identities. Otherwise it uses ANDs",
					},
					cli.BoolFlag{
						Name:  "replace",
						Usage: "if this rule already exists, replace it with this new one",
					},
					cli.BoolFlag{
						Name:  "delete",
						Usage: "delete the rule",
					},
					cli.BoolFlag{
						Name:  "restricted, r",
						Usage: "evolves the darc in a restricted mode, ie. NOT using the invoke:darc.evolve_unrestricted command",
					},
				},
			},
		},
	},

	// {
	// 	Name:    "search",
	// 	Usage:   "search for queries",
	// 	Aliases: []string{"s"},
	// 	Action: search,
	// 	Flags: []cli.Flag{
	// 		cli.StringFlag{
	// 			Name:   "bc",
	// 			EnvVar: "BC",
	// 			Usage:  "the ByzCoin config",
	// 		},
	// 		cli.StringFlag{
	// 			Name:   "mc",
	// 			EnvVar: "QUERY", //EL
	// 			Usage:  "the query id, from \"mc create\"",
	// 		},
	// 		cli.StringFlag{
	// 			Name:  "id",
	// 			Usage: "limit results to queries with this id",
	// 		},
	// 		cli.IntFlag{
	// 			Name:  "count, c",
	// 			Usage: "limit results to X querries",
	// 		},
	// 		cli.StringFlag{
	// 			Name:  "from",
	// 			Usage: "return queries from this time (accepts mm-dd-yyyy or relative times like '10m ago')",
	// 		},
	// 		cli.StringFlag{
	// 			Name:  "to",
	// 			Usage: "return querries to this time (accepts mm-dd-yyyy or relative times like '10m ago')",
	// 		},
	// 		cli.DurationFlag{
	// 			Name:  "for",
	// 			Usage: "return queries for this long after the from time (when for is given, to is ignored)",
	// 		},
	// 	},

	// },
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
}

var cliApp = cli.NewApp()
var dataDir = ""
var gitTag = "dev"

func init() {
	cliApp.Name = "mc"
	cliApp.Usage = "Create and work with MedChain."
	cliApp.Version = gitTag
	cliApp.Commands = cmds
	cliApp.Flags = []cli.Flag{
		cli.IntFlag{
			Name:  "debug, d",
			Value: 0,
			Usage: "debug-level: 1 for terse, 5 for maximal",
		},
		cli.StringFlag{
			Name:  "config, c",
			Value: cfgpath.GetDataPath(cliApp.Name),
			Usage: "path to configuration-directory",
		},
	}
	cliApp.Before = func(c *cli.Context) error {
		log.SetDebugVisible(c.Int("debug"))
		bcadminlib.ConfigPath = c.String("config")
		return nil
	}

}

func main() {
	rand.Seed(time.Now().UTC().UnixNano())
	log.ErrFatal(cliApp.Run(os.Args))
}

// getClient will create a new medchain.Client, given the input
// available in the commandline. If priv is false, then it will not
// look for a private key and set up the signers. (This is used for
// searching, which does not require having a private key available
// because it does not submit transactions.)
func getClient(c *cli.Context, priv bool) (*medchain.Client, error) {
	bc := c.String("bc")
	if bc == "" {
		return nil, errors.New("--bc flag is required")
	}

	cfgBuf, err := ioutil.ReadFile(bc)
	if err != nil {
		return nil, err
	}
	var cfg bcConfig
	err = protobuf.DecodeWithConstructors(cfgBuf, &cfg,
		network.DefaultConstructors(cothority.Suite))
	if err != nil {
		return nil, err
	}

	cl := medchain.NewClient(byzcoin.NewClient(cfg.ByzCoinID, cfg.Roster))

	d, err := cl.ByzCoin.GetGenDarc()
	if err != nil {
		return nil, err
	}
	cl.DarcID = d.GetBaseID()

	// Initialize project Darcs hash map
	cl.AllDarcs = make(map[string]*darc.Darc)
	cl.AllDarcIDs = make(map[string]darc.ID)

	// The caller doesn't want/need signers.
	if !priv {
		return cl, nil
	}

	// get the private key from the cmdline.
	sstr := c.String("sign")
	if sstr == "" {
		return nil, errors.New("--sign is required")
	}
	signer, err := bcadminlib.LoadKeyFromString(sstr)
	if err != nil {
		return nil, err
	}
	cl.Signers = []darc.Signer{*signer}

	return cl, nil
}

func create(c *cli.Context) error {
	cl, err := getClient(c, true)
	if err != nil {
		return err
	}

	e := c.String("darc")
	if e == "" {
		genDarc, err := cl.ByzCoin.GetGenDarc()
		if err != nil {
			return err
		}
		cl.DarcID = genDarc.GetBaseID()
	} else {
		eb, err := bcadminlib.StringToDarcID(e)
		if err != nil {
			return err
		}
		cl.DarcID = darc.ID(eb)
	}

	err = cl.Create()
	if err != nil {
		return err
	}

	log.Infof("export MC=%x", cl.NamingInstance.Slice())
	return bcadminlib.WaitPropagation(c, cl.ByzCoin)
}

func createQuery(c *cli.Context) error {
	cl, err := getClient(c, true)
	if err != nil {
		return err
	}
	e := c.String("mc")
	if e == "" {
		return errors.New("--mc is required")
	}
	eb, err := hex.DecodeString(e)
	if err != nil {
		return err
	}
	cl.NamingInstance = byzcoin.NewInstanceID(eb)

	id := c.String("id")
	stat := c.String("status")
	w := c.Int("wait")

	// Status is set, so one shot query.
	if stat != "" {
		_, _, err := cl.CreateInstance(w, medchain.NewQuery(id, stat))
		return err
	}

	// Status is empty, so read from stdin.
	s := bufio.NewScanner(os.Stdin)
	for s.Scan() {
		_, _, err := cl.CreateInstance(w, medchain.NewQuery(id, s.Text()))
		if err != nil {
			return err
		}
	}

	return bcadminlib.WaitPropagation(c, cl.ByzCoin)
}

var none = time.Unix(0, 0)

// parseTime will accept either dates or "X ago" where X is a duration.
func parseTime(in string) (time.Time, error) {
	if strings.HasSuffix(in, " ago") {
		in = strings.Replace(in, " ago", "", -1)
		d, err := time.ParseDuration(in)
		if err != nil {
			return none, err
		}
		return time.Now().Add(-1 * d), nil
	}
	tm, err := time.Parse("2006-01-02", in)
	if err != nil {
		return none, err
	}
	return tm, nil
}

// NOT FULLY implemented yet.
// func search(c *cli.Context) error {
// 	req := &medchain.SearchRequest{
// 		Status: c.String("status"),
// 	}

// 	f := c.String("from")
// 	if f != "" {
// 		ft, err := parseTime(f)
// 		if err != nil {
// 			return err
// 		}
// 		req.From = ft.UnixNano()
// 	}

// 	forDur := c.Duration("for")
// 	if forDur == 0 {
// 		// No -for, parse -to.
// 		t := c.String("to")
// 		if t != "" {
// 			tt, err := parseTime(t)
// 			if err != nil {
// 				return err
// 			}
// 			req.To = tt.UnixNano()
// 		}
// 	} else {
// 		// Parse -for
// 		req.To = time.Unix(0, req.From).Add(forDur).UnixNano()
// 	}

// 	cl, err := getClient(c, false)
// 	if err != nil {
// 		return err
// 	}
// 	e := c.String("mc")
// 	if e == "" {
// 		return errors.New("--mc is required")
// 	}
// 	eb, err := hex.DecodeString(e)
// 	if err != nil {
// 		return err
// 	}
// 	cl.Instance = byzcoin.NewInstanceID(eb)

// 	resp, err := cl.Search(req)
// 	if err != nil {
// 		return err
// 	}

// 	ct := c.Int("count")

// 	for _, x := range resp.Queries {
// 		const tsFormat = "2006-01-02 15:04:05"
// 		log.Infof("%v\t%v\t%v", time.Unix(0, x.When).Format(tsFormat), x.ID, x.Status)

// 		if ct != 0 {
// 			ct--
// 			if ct == 0 {
// 				break
// 			}
// 		}
// 	}

// 	if resp.Truncated {
// 		return cli.NewExitError("", 1)
// 	}
// 	return nil
// }

func key(c *cli.Context) error {
	if f := c.String("print"); f != "" {
		sig, err := bcadminlib.LoadSigner(f)
		if err != nil {
			return errors.New("couldn't load signer: " + err.Error())
		}
		log.Infof("Private: %s\nPublic: %s", sig.Ed25519.Secret, sig.Ed25519.Point)
		return nil
	}
	newSigner := darc.NewSignerEd25519(nil, nil)
	err := bcadminlib.SaveKey(newSigner)
	if err != nil {
		return err
	}

	var fo io.Writer

	save := c.String("save")
	if save == "" {
		fo = os.Stdout
	} else {
		file, err := os.Create(save)
		if err != nil {
			return err
		}
		fo = file
		defer func() {
			err := file.Close()
			if err != nil {
				log.Error(err)
			}
		}()
	}
	_, err = fmt.Fprintln(fo, newSigner.Identity().String())
	return err
}

func addProjectDarc(c *cli.Context) error {
	mcArg := c.String("mc")
	if mcArg == "" {
		return xerrors.New("--mc flag is required")
	}

	cfg, cl, err := lib.LoadConfig(mcArg)
	if err != nil {
		return err
	}

	dstr := c.String("darc")
	if dstr == "" {
		dstr = cfg.AdminDarc.GetIdentityString()
	}
	dSpawn, err := lib.GetDarcByString(cl, dstr)
	if err != nil {
		return err
	}

	var signer *darc.Signer

	identities := c.StringSlice("identity")

	if len(identities) == 0 {
		s := darc.NewSignerEd25519(nil, nil)
		err = lib.SaveKey(s)
		if err != nil {
			return err
		}
		identities = append(identities, s.Identity().String())
	}

	Y := expression.InitParser(func(s string) bool { return true })

	for _, id := range identities {
		expr := []byte(id)
		_, err := expression.Evaluate(Y, expr)
		if err != nil {
			return xerrors.Errorf("failed to parse id: %v", err)
		}
	}

	sstr := c.String("sign")
	if sstr == "" {
		signer, err = lib.LoadKey(cfg.AdminIdentity)
	} else {
		signer, err = lib.LoadKeyFromString(sstr)
	}
	if err != nil {
		return err
	}

	var desc []byte
	if c.String("desc") == "" {
		desc = []byte(lib.RandString(10))
	} else {
		if len(c.String("desc")) > 1024 {
			return xerrors.New("descriptions longer than 1024 characters are not allowed")
		}
		desc = []byte(c.String("desc"))
	}

	deferredExpr := expression.InitOrExpr(identities...)
	adminExpr := expression.InitAndExpr(identities...)

	rules := darc.NewRules()
	rules.AddRule("invoke:"+byzcoin.ContractDarcID+".evolve", adminExpr)
	rules.AddRule("_sign", adminExpr)
	if c.Bool("deferred") {
		rules.AddRule("spawn:deferred", deferredExpr)
		rules.AddRule("invoke:deferred.addProof", deferredExpr)
		rules.AddRule("invoke:deferred.execProposedTx", deferredExpr)
	}
	if c.Bool("unrestricted") {
		err = rules.AddRule("invoke:"+byzcoin.ContractDarcID+".evolve_unrestricted", adminExpr)
		if err != nil {
			return err
		}
	}

	d := darc.NewDarc(rules, desc)

	dBuf, err := d.ToProto()
	if err != nil {
		return err
	}

	instID := byzcoin.NewInstanceID(dSpawn.GetBaseID())

	counters, err := cl.GetSignerCounters(signer.Identity().String())

	spawn := byzcoin.Spawn{
		ContractID: byzcoin.ContractDarcID,
		Args: []byzcoin.Argument{
			{
				Name:  "darc",
				Value: dBuf,
			},
		},
	}

	ctx, err := cl.CreateTransaction(byzcoin.Instruction{
		InstanceID:    instID,
		Spawn:         &spawn,
		SignerCounter: []uint64{counters.Counters[0] + 1},
	})
	if err != nil {
		return err
	}
	err = ctx.FillSignersAndSignWith(*signer)
	if err != nil {
		return err
	}

	_, err = cl.AddTransactionAndWait(ctx, 10)
	if err != nil {
		return err
	}

	_, err = fmt.Fprintln(c.App.Writer, d.String())
	if err != nil {
		return err
	}

	// Saving ID in special file
	output := c.String("out_id")
	if output != "" {
		err = ioutil.WriteFile(output, []byte(d.GetIdentityString()), 0644)
		if err != nil {
			return err
		}
	}

	// Saving key in special file
	output = c.String("out_key")
	if len(c.StringSlice("identity")) == 0 && output != "" {
		err = ioutil.WriteFile(output, []byte(identities[0]), 0600)
		if err != nil {
			return err
		}
	}

	return lib.WaitPropagation(c, cl)

}

func faultThreshold(n int) int {
	return (n - 1) / 3
}

func threshold(n int) int {
	return n - faultThreshold(n)
}
