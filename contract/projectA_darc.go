package contract

import (
	"bytes"

	"go.dedis.ch/cothority/v3"
	"go.dedis.ch/cothority/v3/byzcoin"
	"go.dedis.ch/cothority/v3/darc"
	"golang.org/x/xerrors"
)

// ProjectADarcID denotes a secure DARC contract for project A. It
// provide two forms of security. The first is "restricted evolution", where
// the evolve command only allows changes to existing rules, it is not allowed
// to add new rules. There exists an additional command "evolve_unrestricted"
// that allows authorised users to change the rules arbitrarily. Our second
// form of security is "controlled spawn", where the rules of the secure darcs
// spawned using this contract are subject to some restrictions, e.g., the new
// rules must not contain spawn:inseucre_darc.
const ProjectADarcID = "projectA_darc"

type projectADarc struct {
	byzcoin.BasicContract
	darc.Darc
	contract byzcoin.ReadOnlyContractRegistry
}

var _Contract = (*projectADarc)(nil)

const cmdDarcEvolveUnrestriction = "evolve_unrestricted"
const cmdDarcEvolve = "evolve"

func projectADarcFromBytes(in []byte) (byzcoin.Contract, error) {
	d, err := darc.NewFromProtobuf(in)
	if err != nil {
		return nil, xerrors.Errorf("darc decoding: %v", err)
	}
	c := &projectADarc{Darc: *d}
	return c, nil
}

// SetRegistry keeps the reference of the contract registry.
func (c *projectADarc) SetRegistry(r byzcoin.ReadOnlyContractRegistry) {
	c.contract = r
}

// VerifyDeferredInstruction does the same as the standard VerifyInstruction
// method in the diferrence that it does not take into account the counters. We
// need the Darc contract to opt in for deferred transaction because it is used
// by default when spawning new contracts.
func (c *projectADarc) VerifyDeferredInstruction(rst byzcoin.ReadOnlyStateTrie, inst byzcoin.Instruction, ctxHash []byte) error {
	err := inst.VerifyWithOption(rst, ctxHash, &byzcoin.VerificationOptions{IgnoreCounters: true})
	return cothority.ErrorOrNil(err, "instruction verification")
}

func (c *projectADarc) Spawn(rst byzcoin.ReadOnlyStateTrie, inst byzcoin.Instruction, coins []byzcoin.Coin) (sc []byzcoin.StateChange, cout []byzcoin.Coin, err error) {
	cout = coins

	if inst.Spawn.ContractID == ProjectADarcID {
		darcBuf := inst.Spawn.Args.Search("darc")
		d, err := darc.NewFromProtobuf(darcBuf)
		if err != nil {
			return nil, nil, xerrors.Errorf("given DARC could not be decoded: %v", err)
		}
		if d.Version != 0 {
			return nil, nil, xerrors.New("DARC version must start at 0")
		}

		id := d.GetBaseID()

		// This hard-coded constraint prohibits the identities from spawning DARCs.
		if d.Rules.Contains("spawn:projectA_darc") {
			return nil, nil, xerrors.New("a secure DARC is not allowed to spawn a DARC")
		}

		//TODO: check the whitelist of rules for roles and identities.

		return []byzcoin.StateChange{
			byzcoin.NewStateChange(byzcoin.Create, byzcoin.NewInstanceID(id), ProjectADarcID, darcBuf, id),
		}, coins, nil
	}

	// If we got here this is a spawn:xxx in order to spawn
	// a new instance of contract xxx, so do that.

	if c.contract == nil {
		return nil, nil, xerrors.New("contracts registry is missing due to bad initialization")
	}

	cfact, found := c.contract.Search(inst.Spawn.ContractID)
	if !found {
		return nil, nil, xerrors.New("couldn't find this contract type: " + inst.Spawn.ContractID)
	}

	// Pass nil into the contract factory here because this instance does not exist yet.
	// So the factory will make a zero-value instance, and then calling Spawn on it
	// will give it a chance to encode it's zero state and emit one or more StateChanges to put itself
	// into the trie.
	c2, err := cfact(nil)
	if err != nil {
		return nil, nil, xerrors.Errorf("could not spawn new zero instance: %v", err)
	}
	if cwr, ok := c2.(byzcoin.ContractWithRegistry); ok {
		cwr.SetRegistry(c.contract)
	}

	scs, coins, err := c2.Spawn(rst, inst, coins)
	return scs, coins, cothority.ErrorOrNil(err, "spawn instance")
}

func (c *projectADarc) Invoke(rst byzcoin.ReadOnlyStateTrie, inst byzcoin.Instruction, coins []byzcoin.Coin) ([]byzcoin.StateChange, []byzcoin.Coin, error) {
	switch inst.Invoke.Command {
	case cmdDarcEvolve:
		var darcID darc.ID
		_, _, _, darcID, err := rst.GetValues(inst.InstanceID.Slice())
		if err != nil {
			return nil, nil, xerrors.Errorf("reading trie: %v", err)
		}

		darcBuf := inst.Invoke.Args.Search("darc")
		newD, err := darc.NewFromProtobuf(darcBuf)
		if err != nil {
			return nil, nil, xerrors.Errorf("darc encoding: %v", err)
		}
		oldD, err := byzcoin.LoadDarcFromTrie(rst, darcID)
		if err != nil {
			return nil, nil, xerrors.Errorf("darc from trie: %v", err)
		}
		// do not allow modification of evolve_unrestricted
		if isChangingEvolveUnrestricted(oldD, newD) {
			return nil, nil, xerrors.New("the evolve command is not allowed to change the the evolve_unrestricted rule")
		}
		if err := newD.SanityCheck(oldD); err != nil {
			return nil, nil, xerrors.Errorf("sanity check: %v", err)
		}
		// use the subset rule if it's not a genesis Darc
		_, _, _, genesisDarcID, err := byzcoin.GetValueContract(rst, byzcoin.NewInstanceID(nil).Slice())
		if err != nil {
			return nil, nil, xerrors.Errorf("getting contract: %v", err)
		}
		if !genesisDarcID.Equal(oldD.GetBaseID()) {
			if !newD.Rules.IsSubset(oldD.Rules) {
				return nil, nil, xerrors.New("rules in the new version must be a subset of the previous version")
			}
		}
		return []byzcoin.StateChange{
			byzcoin.NewStateChange(byzcoin.Update, inst.InstanceID, ProjectADarcID, darcBuf, darcID),
		}, coins, nil
	case cmdDarcEvolveUnrestriction:
		var darcID darc.ID
		_, _, _, darcID, err := rst.GetValues(inst.InstanceID.Slice())
		if err != nil {
			return nil, nil, xerrors.Errorf("reading trie: %v", err)
		}

		darcBuf := inst.Invoke.Args.Search("darc")
		newD, err := darc.NewFromProtobuf(darcBuf)
		if err != nil {
			return nil, nil, xerrors.Errorf("encoding darc: %v", err)
		}
		oldD, err := byzcoin.LoadDarcFromTrie(rst, darcID)
		if err != nil {
			return nil, nil, xerrors.Errorf("darc from trie: %v", err)
		}
		if err := newD.SanityCheck(oldD); err != nil {
			return nil, nil, xerrors.Errorf("sanity check: %v", err)
		}
		return []byzcoin.StateChange{
			byzcoin.NewStateChange(byzcoin.Update, inst.InstanceID, ProjectADarcID, darcBuf, darcID),
		}, coins, nil
	default:
		return nil, nil, xerrors.New("invalid command: " + inst.Invoke.Command)
	}
}

func isChangingEvolveUnrestricted(oldD *darc.Darc, newD *darc.Darc) bool {
	oldExpr := oldD.Rules.Get(darc.Action("invoke:" + ProjectADarcID + "." + cmdDarcEvolveUnrestriction))
	newExpr := newD.Rules.Get(darc.Action("invoke:" + ProjectADarcID + "." + cmdDarcEvolveUnrestriction))
	if len(oldExpr) == 0 && len(newExpr) == 0 {
		return false
	}
	if bytes.Equal(oldExpr, newExpr) {
		return false
	}
	return true
}
