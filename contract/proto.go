package contract

import (
	"go.dedis.ch/cothority/v3/byzcoin"
	"go.dedis.ch/cothority/v3/skipchain"
)

// Query is the struvutre of key-value pairs stored in the ledger
// Value field of the query holds UserID + Query Definition
//When should be set using the UnixNano() method
// in package time.
type Query struct {
	ID     string //assumed to be like query_id:user_id:databaseX.<type of query, e.g. patient_list, count_per_site, etc. >
	Status string
	//When   int64 //TODO: to be able to search the queries by timestamp
}

// QueryData is the structure that will hold all key-value pairs.
type QueryData struct {
	Storage []Query
}

// NewQuery returns a new query with k and v as its ID and Status, respectivley.
func NewQuery(k, v string) Query {
	res := Query{
		ID:     k,
		Status: v,
	}
	return res
}

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
