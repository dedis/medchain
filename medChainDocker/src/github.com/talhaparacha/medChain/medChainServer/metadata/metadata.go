package metadata

import (
	"github.com/dedis/cothority/omniledger/darc"
	"github.com/dedis/cothority/omniledger/service"
	"github.com/google/uuid"
)

type Hospital struct {
	Name                  string
	SuperAdmin            *GenericUser
	AdminListDarcBaseId   string
	ManagerListDarcBaseId string
	UserListDarcBaseId    string
	Admins                []*GenericUser
	Managers              []*GenericUser
	Users                 []*GenericUser
}

type GenericUser struct {
	Id         darc.Identity
	Name       string
	DarcBaseId string
	Hospital   *Hospital
	IsCreated  bool
	Role       string
	Projects   map[string]*Project
}

type Project struct {
	Id         string
	Name       string
	DarcBaseId string
	Managers   []*GenericUser
	Users      []*GenericUser
	Queries    map[string][]*GenericUser
	IsCreated  bool
}

type Metadata struct {
	Hospitals                map[string]*Hospital
	GenericUsers             map[string]*GenericUser
	WaitingForCreation       map[string]*GenericUser
	Projects                 map[string]*Project
	BaseIdToDarcMap          map[string]*darc.Darc
	DarcIdToBaseIdMap        map[string]string
	AllSuperAdminsDarcBaseId string
	AllAdminsDarcBaseId      string
	AllManagersDarcBaseId    string
	AllUsersDarcBaseId       string
	ProjectCreatorDarcBaseId string
	GenesisBlock             *service.CreateGenesisBlockResponse
	GenesisMsg               *service.CreateGenesisBlock
	GenesisDarcBaseId        string
	SigningServiceUrl        string
}

func NewMetadata() *Metadata {
	return &Metadata{Hospitals: make(map[string]*Hospital), GenericUsers: make(map[string]*GenericUser), WaitingForCreation: make(map[string]*GenericUser), Projects: make(map[string]*Project), BaseIdToDarcMap: make(map[string]*darc.Darc), DarcIdToBaseIdMap: make(map[string]string)}
}

func NewHospital(IdValue darc.Identity, HospitalNameValue string, SuperAdminNameValue string) (*Hospital, *GenericUser) {
	hospital := Hospital{Name: HospitalNameValue, Admins: make([]*GenericUser, 0), Managers: make([]*GenericUser, 0), Users: make([]*GenericUser, 0)}
	super_admin := newSuperAdmin(IdValue, SuperAdminNameValue, &hospital)
	return &hospital, super_admin
}

func newGenericUser(IdValue darc.Identity, NameValue string, role string, HospitalPointer *Hospital) *GenericUser {
	return &GenericUser{Id: IdValue, Name: NameValue, Hospital: HospitalPointer, Role: role, IsCreated: false, Projects: make(map[string]*Project)}
}

func newSuperAdmin(IdValue darc.Identity, NameValue string, HospitalPointer *Hospital) *GenericUser {
	super_admin := newGenericUser(IdValue, NameValue, "super_admin", HospitalPointer)
	HospitalPointer.SuperAdmin = super_admin
	return super_admin
}

func NewProject(name string) (*Project, error) {
	id, err := uuid.NewUUID()
	if err != nil {
		return nil, err
	}
	return &Project{Id: id.String() + name, Name: name, Managers: make([]*GenericUser, 0), Users: make([]*GenericUser, 0), Queries: make(map[string][]*GenericUser), IsCreated: false}, nil
}

func NewAdmin(IdValue darc.Identity, NameValue string, HospitalPointer *Hospital) *GenericUser {
	admin_metadata := newGenericUser(IdValue, NameValue, "admin", HospitalPointer)
	HospitalPointer.Admins = append(HospitalPointer.Admins, admin_metadata)
	return admin_metadata
}

func NewManager(IdValue darc.Identity, NameValue string, HospitalPointer *Hospital) *GenericUser {
	manager_metadata := newGenericUser(IdValue, NameValue, "manager", HospitalPointer)
	HospitalPointer.Managers = append(HospitalPointer.Managers, manager_metadata)
	return manager_metadata
}

func NewUser(IdValue darc.Identity, NameValue string, HospitalPointer *Hospital) *GenericUser {
	user_metadata := newGenericUser(IdValue, NameValue, "user", HospitalPointer)
	HospitalPointer.Users = append(HospitalPointer.Users, user_metadata)
	return user_metadata
}
