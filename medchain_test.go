package contract

import (
	"bytes"
	"fmt"

	"go.dedis.ch/cothority/v3/darc"
	"go.dedis.ch/cothority/v3/darc/expression"
)

func medchainTest() {
	// Imagine a client spwans/updates queries in a sepcific project.
	// Thus, we need to check for users authorizaitons
	// for these actions against project darcs.
	// We begin by creating a darc on the server.
	// We can create project darcs as follows:
	client1 := darc.NewSignerEd25519(nil, nil)
	projectARules := darc.InitRules([]darc.Identity{client1.Identity()}, []darc.Identity{})
	pADarc := darc.NewDarc(projectARules, []byte("project A darc"))
	fmt.Println(pADarc.Verify(true))

	// Now the client wants to evolve the darc (i.e., add a new signer to it), so it
	// creates a request and then sends it to the server.
	client2 := darc.NewSignerEd25519(nil, nil)
	rules2 := darc.InitRules([]darc.Identity{client2.Identity()}, []darc.Identity{})
	pADarcEvol := darc.NewDarc(rules2, []byte("project A darc evolved"))
	pADarcEvol.EvolveFrom(pADarc)
	r, pADarcEvolBuf, err := pADarcEvol.MakeEvolveRequest(client1)
	fmt.Println(err)

	// Client sends request r and serialised darc pADarcEvolBuf to the server, and
	// the server must verify it. Usually the server will look in its
	// database for the base ID of the darc in the request and find the
	// latest one. But in this case we assume it already knows. If the
	// verification is successful, then the server should add the darc in
	// the request to its database.
	fmt.Println(r.Verify(pADarc)) // Assume we can find pADarcEvol given the request r.
	pADarcEvolServer, _ := r.MsgToDarc(pADarcEvolBuf)
	fmt.Println(bytes.Equal(pADarcEvolServer.GetID(), pADarcEvol.GetID()))

	// If the darcs stored on the server are trustworthy, then using
	// `Request.Verify` is enough. To do a complete verification,
	// Darc.Verify should be used. This will traverse the chain of
	// evolution and verify every evolution. However, the Darc.Path
	// attribute must be set.
	fmt.Println(pADarcEvolServer.VerifyWithCB(func(s string, latest bool) *darc.Darc {
		if s == darc.NewIdentityDarc(pADarc.GetID()).String() {
			return pADarc
		}
		return nil
	}, true))

	// The above illustrates the basic use of darcs, in the following
	// examples, we show how to create custom rules to enforce custom
	// policies. We begin by making another evolution that has a custom
	// action.
	owner3 := darc.NewSignerEd25519(nil, nil)
	action3 := darc.Action("custom_action")
	expr3 := expression.InitAndExpr(
		owner1.Identity().String(),
		owner2.Identity().String(),
		owner3.Identity().String())
	d3 := d1.Copy()
	d3.Rules.AddRule(action3, expr3)

	// Typically the Msg part of the request is a digest of the actual
	// message. For simplicity in this example, we put the actual message
	// in there.
	r, _ = darc.InitAndSignRequest(d3.GetID(), action3, []byte("example request"), owner3)
	if err := r.Verify(d3); err != nil {
		// not ok because the expression is created using logical and
		fmt.Println("not ok!")
	}

	r, _ = darc.InitAndSignRequest(d3.GetID(), action3, []byte("example request"), owner1, owner2, owner3)
	if err := r.Verify(d3); err == nil {
		fmt.Println("ok!")
	}

	// Output:
	// <nil>
	// <nil>
	// <nil>
	// true
	// <nil>
	// not ok!
	// ok!
}
