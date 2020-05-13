package admin

// AccessRight holds the access right of a querier in a Medchain project. The Ids slice store the Ids of all queriers.
// Their access rights are store at the respective index in the Access slice.
// TODO : followup Byzcoin development. There is a limitation in Byzcoin and we can't use maps, that would be easier to manage.
// The hashmap operation in Golang is not deterministic among all nodes and consensus can't be achieved after a contract execution involving maps
type AccessRight struct {
	Ids    []string
	Access []string
}
