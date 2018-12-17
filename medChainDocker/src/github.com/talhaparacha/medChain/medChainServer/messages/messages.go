package messages

type UserInfoRequest struct {
	PublicKey string `json:"public_key"`
	Identity  string `json:"identity"`
}

type SuperAdminInfoReply struct {
	DarcBaseId            string `json:"super_admin_darc_base_id"`
	SuperAdminId          string `json:"super_admin_id"`
	HospitalName          string `json:"hospital_name"`
	AdminListDarcBaseId   string `json:"admin_list_darc_base_id"`
	ManagerListDarcBaseId string `json:"manager_list_darc_base_id"`
	UserListDarcBaseId    string `json:"user_list_darc_base_id"`
	IsCreated             bool   `json:"is_created"`
}

type GenericUserInfoReply struct {
	Id           string `json:"id"`
	Name         string `json:"name"`
	DarcBaseId   string `json:"darc_base_id"`
	SuperAdminId string `json:"super_admin_id"`
	IsCreated    bool   `json:"is_created"`
}

type DarcInfoRequest struct {
	DarcId string `json:"darc_id"`
	BaseId string `json:"base_id"`
}

type DarcInfoReply struct {
	Description string            `json:"description"`
	SignExpr    string            `json:"sign_expression"`
	Rules       []RuleDescription `json:"rules"`
	Bytes       []byte            `json:"bytes"`
}

type RuleDescription struct {
	Action string `json:"action"`
	Expr   string `json:"expression"`
}

type ListReply struct {
	Users []string `json:"users"`
}

type AddGenericUserRequest struct {
	PublicKey          string `json:"new_public_key"`
	Identity           string `json:"new_identity"`
	SuperAdminIdentity string `json:"super_admin_id"`
}

type AddGenericUserReply struct {
	Transaction string   `json:"transaction"`
	Signers     []string `json:"signers"`
	Threshold   int      `json:"threshold"`
}
