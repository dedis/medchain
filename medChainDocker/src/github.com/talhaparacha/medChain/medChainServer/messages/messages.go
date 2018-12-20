package messages

type UserInfoRequest struct {
	PublicKey string `json:"public_key"`
	Identity  string `json:"identity"`
}

type GeneralInfoReply struct {
	GenesisDarcBaseId        string `json:"genesis_darc_base_id"`
	AllSuperAdminsDarcBaseId string `json:"all_super_admins_darc_base_id"`
	AllAdminsDarcBaseId      string `json:"all_admins_darc_base_id"`
	AllManagersDarcBaseId    string `json:"all_managers_darc_base_id"`
	AllUsersDarcBaseId       string `json:"all_users_darc_base_id"`
	AllUsersDarc             string `json:"all_users_darc"`
	UserProjectsMap          string `json:"user_projects_maps"`
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
	PublicKey          string   `json:"new_public_key"`
	SuperAdminIdentity string   `json:"super_admin_id"`
	Name               string   `json:"name"`
	PreferredSigners   []string `json:"preferred_signers"`
}

type AddGenericUserReply struct {
	Id                 string         `json:"user_id"`
	Transaction        string         `json:"transaction"`
	InstructionDigests map[int][]byte `json:"instruction_digests"`
	Signers            map[string]int `json:"signers"`
}

type CommitNewGenericUserRequest struct {
	Transaction string `json:"transaction"`
}

type AddHospitalRequest struct {
	PublicKey string `json:"new_public_key"`
	Name      string `json:"name"`
}

type AddHospitalReply struct {
	Id                 string         `json:"hospital_id"`
	Transaction        string         `json:"transaction"`
	InstructionDigests map[int][]byte `json:"instruction_digests"`
	Signers            map[string]int `json:"signers"`
}

type CommitHospitalRequest struct {
	Transaction string `json:"transaction"`
}
