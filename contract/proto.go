package contract

// Query is the struvutre of key-value pairs stored in the ledger
// Value field of the query holds UserID + Query Definition
type Query struct {
	ID     string
	Status string
}

// QueryData is the structure that will hold all key-value pairs.
type QueryData struct {
	Storage []Query
}
