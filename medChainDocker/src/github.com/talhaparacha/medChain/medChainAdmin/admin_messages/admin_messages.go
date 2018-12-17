package admin_messages

type SignRequest struct {
	PublicKey   string `json:"public_key"`
	PrivateKey  string `json:"private_key"`
	Transaction string `json:"transaction"`
	Digest      []byte `json:"digest"`
	SignerIndex int    `json:"signer_index"`
}

type SignReply struct {
	Transaction string `json:"transaction"`
}
