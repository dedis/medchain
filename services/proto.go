package medchain

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
