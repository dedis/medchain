package admin

import (
	"go.dedis.ch/cothority/v3/byzcoin"
	"go.dedis.ch/cothority/v3/darc"
	"go.dedis.ch/cothority/v3/darc/expression"
	"golang.org/x/xerrors"
	"strings"
)


// ------------------------------------------------------------------------
// Manage slice as sets
// ------------------------------------------------------------------------

// Find the index of an element in the slice. Return -1 and false if the value is not in the slice
func Find(slice []string, val string) (int, bool) {
	for i, item := range slice {
		if item == val {
			return i, true
		}
	}
	return -1, false
}

// Add a new value to the slice. Check if the value is no already in the slice
func Add(slice *[]string, val string) error {
	idx, _ := Find(*slice, val)
	if idx != -1 {
		return xerrors.New("The id is already registered")
	}
	// Add the new querier ID and access rights
	*slice = append(*slice, val)
	return nil
}

// Remove a value from the slice.
func Remove(slice *[]string, val string) error {
	idx, _ := Find(*slice, val)
	if idx == -1 {
		return xerrors.New("There is no such value")
	}
	// Add the new querier ID and access rights
	*slice = append((*slice)[:idx], (*slice)[idx+1:]...)
	return nil
}

// Modify a value in the slice. Check if the new value is not already present in the slice
func Update(slice *[]string, oldVal, newVal string) error {
	idx, _ := Find(*slice, newVal)
	if idx != -1 {
		return xerrors.New("The new value is already present in the slice")
	}
	idx, _ = Find(*slice, oldVal)
	if idx == -1 {
		return xerrors.New("There is no such value")
	}

	(*slice)[idx] = newVal
	return nil
}

// ------------------------------------------------------------------------
// Other helper methods
// ------------------------------------------------------------------------

// Spawn a transaction to Byzcoin. Increment the signer counter after each successful execution
func (cl *Client) spawnTransaction(ctx byzcoin.ClientTransaction) error {
	err := ctx.FillSignersAndSignWith(cl.adminkeys)
	if err != nil {
		return xerrors.Errorf("Signing: %w", err)
	}
	_, err = cl.Bcl.AddTransactionAndWait(ctx, 10)
	if err != nil {
		return xerrors.Errorf("Adding transaction to the ledger: %w", err)
	}
	cl.incrementSignerCounter()
	return nil
}

// This method add rules to darc based on the actions and expression.
func addActionsToDarc(userDarc *darc.Darc, action string, expr expression.Expr) error {
	actions := strings.Split(action, ",")
	for i := 0; i < len(actions); i++ {
		dAction := darc.Action(actions[i])
		err := userDarc.Rules.AddRule(dAction, expr)
		if err != nil {
			return err
		}
	}
	return nil
}