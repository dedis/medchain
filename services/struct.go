package medchain

/*
This holds the messages used to communicate with the service over the network.
*/

import (
	"go.dedis.ch/cothority/v3/byzcoin"
	"go.dedis.ch/cothority/v3/darc"
	"go.dedis.ch/cothority/v3/skipchain"
	"go.dedis.ch/onet/v3"
	"go.dedis.ch/onet/v3/network"
)

// We need to register all messages so the network knows how to handle them.
func init() {
	network.RegisterMessages(
		&SearchRequest{}, &SearchReply{},
		&AddQueryRequest{}, &AddQueryReply{},
		&AddDeferredQueryRequest{}, &AddDeferredQueryReply{},
		&QueryRequest{}, &QueryReply{},
		&VerifyStatusRequest{}, &VerifyStatusReply{},
		&SignDeferredTxRequest{}, &SignDeferredTxReply{},
		&PropagateIDRequest{}, &PropagateIDReply{},
		&GetSharedDataRequest{}, &GetSharedDataReply{},
		&ExecuteDeferredTxRequest{}, &ExecuteDeferredTxReply{},
		&AuthorizeQueryRequest{}, &AuthorizeQueryReply{},
	)

}

const (
	// ErrorParse indicates an error while parsing the protobuf-file.
	ErrorParse = iota + 4000
)

// AddQueryRequest includes the query data that is to be authorized
type AddQueryRequest struct {
	ClientID    string
	QueryID     string
	QueryStatus string
	QueryInstID byzcoin.InstanceID
	BlockID     skipchain.SkipBlockID
}

// AddQueryReply is the reply to CreateQueryRequest
// The reply is true if the query is authorized and false otherwise
type AddQueryReply struct {
	QueryInstID byzcoin.InstanceID
	OK          bool
}

// AddDeferredQueryRequest includes the query data that is to be authorized
type AddDeferredQueryRequest struct {
	ClientID    string
	QueryID     string
	QueryStatus string
	QueryInstID byzcoin.InstanceID
	BlockID     skipchain.SkipBlockID
	DarcID      darc.ID
}

// AddDeferredQueryReply is the reply to CreateQueryRequest
// The reply is true if the query is authorized and false otherwise
type AddDeferredQueryReply struct {
	QueryInstID byzcoin.InstanceID
	OK          bool
}

// QueryRequest includes the ID and the status of the query on the skipchain
type QueryRequest struct {
	QueryID string
}

// QueryReply is the reply to QueryRequest
type QueryReply struct {
	OK bool
}

// VerifyStatusRequest includes the status of the query on the skipchain
type VerifyStatusRequest struct {
	ClientID    string
	QueryID     string
	QueryInstID byzcoin.InstanceID
}

// VerifyStatusReply is the reply to VerifyStatusRequest
type VerifyStatusReply struct {
	ClientID    string
	QueryStatus string
	OK          bool
}

// SignDeferredTxRequest message includes the data of the query the client wants to sign
type SignDeferredTxRequest struct {
	// TODO: is the id of the user also needed?
	ClientID    string
	Keys        darc.Signer
	QueryInstID byzcoin.InstanceID
}

// SignDeferredTxReply is the reply to SignDeferredTxRequest
type SignDeferredTxReply struct {
	QueryStatus string
	QueryInstID byzcoin.InstanceID
	OK          bool
}

// SearchRequest includes all the search parameters (AND of all provided search
// parameters). Status == "" means "any status". From == 0 means "from the first
// query", and To == 0 means "until now". From and To should be set using the
// UnixNano() method in package time.
type SearchRequest struct {
	Instance byzcoin.InstanceID
	BlockID  skipchain.SkipBlockID
	// Return queries where Query.Status == Status, if Status != "".
	Status string
	// Return queries where When is > From.
	From int64
	// Return queries where When is <= To.
	To int64
}

// SearchReply is the reply to SearchRequest.
type SearchReply struct {
	Queries []Query
	// Queries does not contain all the results. The caller should formulate
	// a new SearchRequest to continue searching, for instance by setting
	// From to the time of the last received event.
	Truncated bool
}

// PropagateIDRequest includes the data that is shared among all nodes
type PropagateIDRequest struct {
	QueryInstID byzcoin.InstanceID
	Status      []byte
	Roster      *onet.Roster
}

// PropagateIDReply is the reply to PropagateIDRequest
type PropagateIDReply struct {
	OK bool
}

// GetSharedDataRequest implments the request to get data that is shared among nodes
type GetSharedDataRequest struct {
}

// GetSharedDataReply is the reply to GetSharedDataRequest
type GetSharedDataReply struct {
	QueryInstIDs []byzcoin.InstanceID
}

// AuthorizeQueryRequest implements the request to authorize the query
type AuthorizeQueryRequest struct {
	QueryID     string
	QueryStatus string
	QueryInstID byzcoin.InstanceID
	BlockID     skipchain.SkipBlockID
	DarcID      darc.ID
}

// AuthorizeQueryReply is the reply to AuthorizeQeryRequest
type AuthorizeQueryReply struct {
	QueryInstID byzcoin.InstanceID
	OK          bool
}

// ExecuteDeferredTxRequest iplements the reques to exec the deferred query tx
type ExecuteDeferredTxRequest struct {
	ClientID    string
	QueryStatus string
	QueryInstID byzcoin.InstanceID
}

// ExecuteDeferredTxReply the reply to ExecuteDeferredTxRequest
type ExecuteDeferredTxReply struct {
	QueryStatus string
	QueryInstID byzcoin.InstanceID
	OK          bool
}
