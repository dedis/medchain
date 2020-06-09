package admin

import (
	"go.dedis.ch/cothority/v3/byzcoin"
	"go.dedis.ch/onet/v3"
	"go.dedis.ch/onet/v3/network"
)

// We need to register all messages so the network knows how to handle them.
func init() {
	network.RegisterMessages(
		DeferredID{}, DeferredIDReply{}, GetDeferredIDs{}, GetDeferredIDsReply{},
	)
}

const (
	// ErrorParse indicates an error while parsing the protobuf-file.
	ErrorParse = iota + 4000
)

// Request the sending of all deferred instance ids stored locally.
type GetDeferredIDs struct {
}

//  Reply to the GetDeferredIDs message. Reply with the list of all deferred instance ids stored locally.
type GetDeferredIDsReply struct {
	Ids []byzcoin.InstanceID
}

// Deferred ID is a request to store a new deferred instance id and to broadcast it to other conodes.
type DeferredID struct {
	Id     byzcoin.InstanceID
	Roster *onet.Roster
}

// DeferredID is the reply to DeferredID
// The reply is true if the query is authorized and false otherwise
type DeferredIDReply struct {
	OK bool
}

// AdminsList store the list of admins identities in the admin darc
type AdminsList struct {
	List []string
}
