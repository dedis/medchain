package signing_messages

import "github.com/DPPH/MedChain/medChainServer/messages"

type AddNewActionRequest struct {
	Action *messages.ActionReply `json:"action"`
}

type AddNewActionReply struct {
	Id string `json:"id"`
}

type ListRequest struct {
	Id string `json:"id"`
}

type ActionInfoRequest struct {
	Id string `json:"id"`
}

type ActionInfoReply struct {
	Id          string
	Initiator   string
	Description string
	Status      string
	Action      *messages.ActionReply
	Signatures  map[string]bool
}

type ListReply struct {
	Actions []*ActionInfoReply `json:"actions"`
}

type ActionUpdate struct {
	SignerId          string `json:"signer_id"`
	UpdateType        string `json:"type"`
	SignedTransaction string `json:"signed_transaction"`
	OldTransaction    string `json:"old_transaction"`
}
