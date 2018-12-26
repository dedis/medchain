package messages

type UserInfoRequest struct {
	PublicKey string `json:"public_key"`
	Identity  string `json:"identity"`
}

type UserTypeReply struct {
	Type string `json:"type"`
}

type GeneralInfoReply struct {
	SigningServiceUrl        string `json:"signing_service_url"`
	GenesisDarcBaseId        string `json:"genesis_darc_base_id"`
	AllSuperAdminsDarcBaseId string `json:"all_super_admins_darc_base_id"`
	AllAdminsDarcBaseId      string `json:"all_admins_darc_base_id"`
	AllManagersDarcBaseId    string `json:"all_managers_darc_base_id"`
	AllUsersDarcBaseId       string `json:"all_users_darc_base_id"`
	AllUsersDarc             string `json:"all_users_darc"`
	UserProjectsMap          string `json:"user_projects_maps"`
}

type HospitalInfoReply struct {
	SuperAdminId          string `json:"super_admin_id"`
	SuperAdminName        string `json:"super_admin_name"`
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
	Role         string `json:"role"`
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
	Initiator          string   `json:"initiator"`
	PublicKey          string   `json:"new_public_key"`
	SuperAdminIdentity string   `json:"super_admin_id"`
	Name               string   `json:"name"`
	PreferredSigners   []string `json:"preferred_signers"`
}

type CommitNewGenericUserRequest struct {
	Transaction string `json:"transaction"`
}

type AddHospitalRequest struct {
	Initiator      string `json:"initiator"`
	PublicKey      string `json:"new_public_key"`
	HospitalName   string `json:"hospital_name"`
	SuperAdminName string `json:"super_admin_name"`
}

type CommitHospitalRequest struct {
	Transaction string `json:"transaction"`
}

type ListGenericUserRequest struct {
	SuperAdminId string `json:"super_admin_id"`
	Role         string `json:"role"`
}

type ListGenericUserReply struct {
	Users []GenericUserInfoReply `json:"users"`
}

type ListHospitalReply struct {
	Hospitals []HospitalInfoReply `json:"hospitals"`
}

type ProjectInfoRequest struct {
	Id string `json:"id"`
}

type ProjectInfoReply struct {
	Id         string                            `json:"id"`
	Name       string                            `json:"name"`
	DarcBaseId string                            `json:"darc_base_id"`
	Managers   []GenericUserInfoReply            `json:"managers"`
	Users      []GenericUserInfoReply            `json:"users"`
	Queries    map[string][]GenericUserInfoReply `json:"queries"`
	IsCreated  bool                              `json:"is_created"`
}

type ListProjectRequest struct {
	Id string `json:"id"`
}

type ListProjectReply struct {
	Projects []ProjectInfoReply `json:"projects"`
}

type AddProjectRequest struct {
	Initiator string              `json:"initiator"`
	Name      string              `json:"name"`
	Managers  []string            `json:"managers"`
	Queries   map[string][]string `json:"queries"`
}

type ActionReply struct {
	Initiator          string         `json:"initiator"`
	ActionType         string         `json:"action_type"`
	Ids                []string       `json:"ids"`
	Transaction        string         `json:"transaction"`
	InstructionDigests map[int][]byte `json:"instruction_digests"`
	Signers            map[string]int `json:"signers"`
}
