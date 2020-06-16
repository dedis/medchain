package utils

import (
	"encoding/hex"
	"strings"

	"go.dedis.ch/cothority/v3/byzcoin"
	"go.dedis.ch/cothority/v3/darc"
	"golang.org/x/xerrors"
)

// GetDarcByID returns a DARC given its ID as a byte array
func GetDarcByID(cl *byzcoin.Client, id []byte) (*darc.Darc, error) {
	pr, err := cl.GetProofFromLatest(id)
	if err != nil {
		return nil, err
	}

	p := &pr.Proof

	vs, cid, _, err := p.Get(id)
	if err != nil {
		return nil, xerrors.Errorf("could not find darc for %x", id)
	}
	if cid != byzcoin.ContractDarcID {
		return nil, xerrors.Errorf("unexpected contract %v, expected a darc", cid)
	}

	d, err := darc.NewFromProtobuf(vs)
	if err != nil {
		return nil, err
	}

	return d, nil
}

// StringToDarcID converts a string representation of a DARC to a byte array
func StringToDarcID(id string) ([]byte, error) {
	if id == "" {
		return nil, xerrors.New("no string given")
	}
	if strings.HasPrefix(id, "darc:") {
		id = id[5:]
	}
	return hex.DecodeString(id)
}

// GetDarcByString returns a DARC given its ID as a string
func GetDarcByString(cl *byzcoin.Client, id string) (*darc.Darc, error) {
	xrep, err := StringToDarcID(id)
	if err != nil {
		return nil, err
	}
	return GetDarcByID(cl, xrep)
}
