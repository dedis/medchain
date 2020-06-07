package admin

import (
	"go.dedis.ch/cothority/v3/byzcoin"
	"go.dedis.ch/onet/v3"
	"go.dedis.ch/onet/v3/network"
)

// We need to register all messages so the network knows how to handle them.
func init() {
	network.RegisterMessages(
		DefferedID{}, DefferedIDReply{}, GetDeferredIDs{}, GetDeferredIDsReply{},
	)
}

const (
	// ErrorParse indicates an error while parsing the protobuf-file.
	ErrorParse = iota + 4000
)

type GetDeferredIDs struct {
}

type GetDeferredIDsReply struct {
	Ids []byzcoin.InstanceID
}

// Count will return how many times the protocol has been run.
type DefferedID struct {
	Id     byzcoin.InstanceID
	Roster *onet.Roster
}

// CountReply returns the number of protocol-runs
type DefferedIDReply struct {
	OK bool
}

// AdminsList store the list of admins identities in the admin darc
type AdminsList struct {
	List []string
}
