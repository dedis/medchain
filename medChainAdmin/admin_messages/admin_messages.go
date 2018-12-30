package admin_messages

import "github.com/DPPH/MedChain/signingService/signing_messages"

type SignRequest struct {
	PublicKey  string                            `json:"public_key"`
	PrivateKey string                            `json:"private_key"`
	ActionInfo *signing_messages.ActionInfoReply `json:"action_info"`
}

type SignReply struct {
	SignerId          string                            `json:"signer_id"`
	ActionInfo        *signing_messages.ActionInfoReply `json:"action_info"`
	OldTransaction    string                            `json:"old_transaction"`
	SignedTransaction string                            `json:"signed_transaction"`
}

type IdRequest struct {
	PublicKey string `json:"public_key"`
}

type IdReply struct {
	Identity string `json:"identity"`
}
