package messages

type UserInfoRequest struct {
	PublicKey string `json:"publick_key"`
	Identity  string `json:"identity"`
}

type UserInfoReply struct {
	DarcBaseId string `json:"darc_base_id"`
}

type DarcInfoRequest struct {
	DarcId string `json:"darc_id"`
	BaseId string `json:"base_id"`
}

type DarcInfoReply struct {
	Description string `json:"description"`
}
