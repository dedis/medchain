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
	"go.dedis.ch/cothority/v3"
	"go.dedis.ch/cothority/v3/byzcoin"
	bcadminlib "go.dedis.ch/cothority/v3/byzcoin/bcadmin/lib"
	"go.dedis.ch/cothority/v3/darc"

	//"go.dedis.ch/cothority/v3/eventlog"
	"go.dedis.ch/cothority/v3/skipchain"
	"go.dedis.ch/onet/v3"
	"go.dedis.ch/onet/v3/cfgpath"
	"go.dedis.ch/onet/v3/log"
	"go.dedis.ch/onet/v3/network"
	"go.dedis.ch/protobuf"
)

type config struct {
	Name    string
	QueryID byzcoin.InstanceID //EventLogID
}

type bcConfig struct {
	Roster    onet.Roster
	ByzCoinID skipchain.SkipBlockID
}

var cmds = cli.Commands{
	{
		Name:    "create",
		Usage:   "create a medchain CLI app",
		Aliases: []string{"c"},
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
		Action: create,
	},

	{
		Name:    "query",
		Usage:   "create one or more queries",
		Aliases: []string{"l"},
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
				EnvVar: "QUERY",
				Usage:  "the query id, from \"mc create\"",
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
		Action: createQuery,
	},
	// {
	// 	Name:    "search",
	// 	Usage:   "search for queries",
	// 	Aliases: []string{"s"},
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
	// 	Action: search,
	// },
	{
		Name:    "key",
		Usage:   "generates a new keypair and prints the public key in the stdout",
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
		Action: key,
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

	//network.RegisterMessage(&openidCfg{})
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
		_, err := cl.CreateQueryAndWait(w, medchain.NewQuery(id, stat))
		return err
	}

	// Status is empty, so read from stdin.
	s := bufio.NewScanner(os.Stdin)
	for s.Scan() {
		_, err := cl.CreateQueryAndWait(w, medchain.NewQuery(id, s.Text()))
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

func faultThreshold(n int) int {
	return (n - 1) / 3
}

func threshold(n int) int {
	return n - faultThreshold(n)
}
