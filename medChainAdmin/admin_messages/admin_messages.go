package admin_messages

type SignRequest struct {
	PublicKey          string         `json:"public_key"`
	PrivateKey         string         `json:"private_key"`
	Transaction        string         `json:"transaction"`
	InstructionDigests map[int][]byte `json:"instruction_digests"`
	Signers            map[string]int `json:"signers"`
}

type SignReply struct {
	Transaction        string         `json:"transaction"`
	InstructionDigests map[int][]byte `json:"instruction_digests"`
	Signers            map[string]int `json:"signers"`
}

type IdRequest struct {
	PublicKey string `json:"public_key"`
}

type IdReply struct {
	Identity string `json:"identity"`
}
