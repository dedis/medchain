package medchain

/*
This holds the messages used to communicate with the service over the network.
*/

import (
	"go.dedis.ch/cothority/v3/byzcoin"
	"go.dedis.ch/cothority/v3/skipchain"
	"go.dedis.ch/onet/v3"
	"go.dedis.ch/onet/v3/network"
)

// We need to register all messages so the network knows how to handle them.
func init() {
	network.RegisterMessages(
		Count{}, CountReply{},
		Clock{}, ClockReply{},
		&SearchRequest{}, &SearchResponse{},
	)

}

const (
	// ErrorParse indicates an error while parsing the protobuf-file.
	ErrorParse = iota + 4000
)

// SearchRequest includes all the search parameters (AND of all provided search
// parameters). Status == "" means "any status". From == 0 means "from the first
// query", and To == 0 means "until now". From and To should be set using the
// UnixNano() method in package time.
type SearchRequest struct {
	Instance byzcoin.InstanceID
	ID       skipchain.SkipBlockID
	// Return queries where Query.Status == Status, if Status != "".
	Status string
	// Return queries where When is > From.
	From int64
	// Return queries where When is <= To.
	To int64
}

// SearchResponse is the reply to QueryRequest.
type SearchResponse struct {
	Queries []Query
	// Queries does not contain all the results. The caller should formulate
	// a new SearchRequest to continue searching, for instance by setting
	// From to the time of the last received event.
	Truncated bool
}

// Clock will run the tepmlate-protocol on the roster and return
// the time spent doing so.
type Clock struct {
	Roster *onet.Roster
}

// ClockReply returns the time spent for the protocol-run.
type ClockReply struct {
	Time     float64
	Children int
}

// Count will return how many times the protocol has been run.
type Count struct {
}

// CountReply returns the number of protocol-runs
type CountReply struct {
	Count int
}
