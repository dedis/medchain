package metadata

import (
	"github.com/dedis/cothority/omniledger/darc"
	"github.com/dedis/cothority/omniledger/service"
)

type Hospital struct {
	Id                    darc.Identity
	Name                  string
	DarcBaseId            string
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
}

type Project struct {
	Name       string
	DarcBaseId string
}

type Metadata struct {
	Hospitals                map[string]*Hospital
	Admins                   map[string]*GenericUser
	Managers                 map[string]*GenericUser
	Users                    map[string]*GenericUser
	Projects                 map[string]*Project
	BaseIdToDarcMap          map[string]*darc.Darc
	DarcIdToBaseIdMap        map[string]string
	AllSuperAdminsDarcBaseId string
	AllAdminsDarcBaseId      string
	AllManagersDarcBaseId    string
	AllUsersDarcBaseId       string
	GenesisBlock             *service.CreateGenesisBlockResponse
	GenesisMsg               *service.CreateGenesisBlock
	GenesisDarc              *darc.Darc
}

func NewMetadata() *Metadata {
	return &Metadata{Hospitals: make(map[string]*Hospital), Admins: make(map[string]*GenericUser), Managers: make(map[string]*GenericUser), Users: make(map[string]*GenericUser), Projects: make(map[string]*Project), BaseIdToDarcMap: make(map[string]*darc.Darc), DarcIdToBaseIdMap: make(map[string]string)}
}

func NewHospital(IdValue darc.Identity, NameValue string) *Hospital {
	return &Hospital{Id: IdValue, Name: NameValue, Admins: make([]*GenericUser, 0), Managers: make([]*GenericUser, 0), Users: make([]*GenericUser, 0)}
}

func NewGenericUser(IdValue darc.Identity, NameValue string, HospitalPointer *Hospital) *GenericUser {
	return &GenericUser{Id: IdValue, Name: NameValue, Hospital: HospitalPointer}
}