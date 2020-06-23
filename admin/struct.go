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

// AccessRight holds the access right of a querier in a Medchain project. The Ids slice store the Ids of all queriers.
// Their access rights are store at the respective index in the Access slice.
// TODO : There is a limitation in Byzcoin and we can't use maps, that would be easier to manage.
// The hashmap operation in Golang is not deterministic among all nodes and consensus can't be achieved after a contract execution involving maps
type AccessRight struct {
	Ids    []string
	Access []string
}
