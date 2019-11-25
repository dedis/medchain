package contract

// Query is the struvutre of key-value pairs stored in the ledger
type Query struct {
	ID    string
	Value []byte
}

// QueryData is the structure that will hold all key-value pairs.
type QueryData struct {
	Storage []Query
}
