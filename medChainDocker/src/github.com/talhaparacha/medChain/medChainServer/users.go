package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/dedis/cothority/omniledger/darc"
	"github.com/dedis/cothority/omniledger/darc/expression"
	"github.com/dedis/cothority/omniledger/service"
	"github.com/dedis/onet/network"
	"github.com/talhaparacha/medChain/medChainServer/messages"
	"github.com/talhaparacha/medChain/medChainServer/metadata"
	"github.com/talhaparacha/medChain/medChainUtils"
)

func GetUserInfo(w http.ResponseWriter, r *http.Request) {
	fmt.Println("/info/user")
	getGenericUserInfo(w, r, metaData.Users)
}

func getGenericUserInfo(w http.ResponseWriter, r *http.Request, user_metadata_map map[string]*metadata.GenericUser) {
	body, err := ioutil.ReadAll(r.Body)
	if medChainUtils.CheckError(err, w, r) {
		return
	}
	var request messages.UserInfoRequest
	err = json.Unmarshal(body, &request)
	if medChainUtils.CheckError(err, w, r) {
		return
	}
	var identity string
	if request.Identity != "" {
		identity = request.Identity
	} else if request.PublicKey != "" {
		id := medChainUtils.LoadIdentityEd25519FromBytes([]byte(request.PublicKey))
		identity = id.String()
	} else {
		medChainUtils.CheckError(errors.New("No identity Nor public key was given"), w, r)
		return
	}
	user_metadata, ok := user_metadata_map[identity]
	if !ok {
		medChainUtils.CheckError(errors.New("Identity unknown"), w, r)
		return
	}
	reply := messages.GenericUserInfoReply{Id: user_metadata.Id.String(), Name: user_metadata.Name, DarcBaseId: user_metadata.DarcBaseId, SuperAdminId: user_metadata.Hospital.Id.String(), IsCreated: user_metadata.IsCreated}
	json_val, err := json.Marshal(&reply)
	if medChainUtils.CheckError(err, w, r) {
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(json_val)
	if medChainUtils.CheckError(err, w, r) {
		return
	}
}

func AddUser(w http.ResponseWriter, r *http.Request) {
	fmt.Println("/add/user")
	replyNewGenericUserRequest(w, r, "User")
}

func replyNewGenericUserRequest(w http.ResponseWriter, r *http.Request, user_type string) {
	body, err := ioutil.ReadAll(r.Body)
	if medChainUtils.CheckError(err, w, r) {
		return
	}
	var request messages.AddGenericUserRequest
	err = json.Unmarshal(body, &request)
	if medChainUtils.CheckError(err, w, r) {
		return
	}
	identity, transaction, signers, threshold, err := prepareNewUser(&request, user_type)
	if medChainUtils.CheckError(err, w, r) {
		return
	}
	reply := messages.AddGenericUserReply{Id: identity, Transaction: transaction, Signers: signers, Threshold: threshold}
	json_val, err := json.Marshal(&reply)
	if medChainUtils.CheckError(err, w, r) {
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(json_val)
	if medChainUtils.CheckError(err, w, r) {
		return
	}
}

func prepareNewUser(request *messages.AddGenericUserRequest, user_type string) (string, string, []string, int, error) {

	hospital_metadata, identityPtr, err := getMetadata(request)
	if err != nil {
		return "", "", nil, 0, err
	}
	identity := *identityPtr

	owner_darc, signers, threshold, err := getSigners(hospital_metadata, user_type)
	if err != nil {
		return "", "", nil, 0, err
	}

	list_darc, err := getListDarc(hospital_metadata, user_type)
	if err != nil {
		return "", "", nil, 0, err
	}

	transaction, new_darc, err := createNewGenericUserTransaction(identity, owner_darc, list_darc, user_type)
	if err != nil {
		return "", "", nil, 0, err
	}
	if err := addGenericUserToMetadata(metaData, hospital_metadata, identity, request.Name, user_type, new_darc); err != nil {
		return "", "", nil, 0, err
	}
	return identity.String(), transaction, signers, threshold, nil
}

func addGenericUserToMetadata(metaData *metadata.Metadata, hospital_metadata *metadata.Hospital, identity darc.Identity, name, user_type string, new_darc *darc.Darc) error {
	user_metadata := metadata.NewGenericUser(identity, name, hospital_metadata)
	base_id := medChainUtils.IDToB64String(new_darc.GetBaseID())
	user_metadata.DarcBaseId = base_id
	hospital_metadata.Users = append(hospital_metadata.Users, user_metadata)
	metaData.Users[identity.String()] = user_metadata
	metaData.WaitingForCreation[base_id] = user_metadata
	return nil
}

func createNewGenericUserTransaction(identity darc.Identity, owner_darc, list_darc *darc.Darc, user_type string) (string, *darc.Darc, error) {

	owners := []darc.Identity{darc.NewIdentityDarc(owner_darc.GetID())}
	rules := darc.InitRulesWith(owners, []darc.Identity{identity}, "invoke:evolve")
	new_darc := darc.NewDarc(rules, []byte("Darc for a single "+user_type))
	new_darc_buff, err := new_darc.ToProto()
	if err != nil {
		return "", nil, err
	}

	new_list_darc := list_darc.Copy()
	new_list_darc.EvolveFrom(list_darc)
	sign_expr := list_darc.Rules.GetSignExpr()
	new_sign_expr := expression.InitOrExpr(string(sign_expr), new_darc.GetIdentityString())
	new_list_darc.Rules.UpdateSign(new_sign_expr)
	new_list_darc_buff, err := new_list_darc.ToProto()
	if err != nil {
		return "", nil, err
	}

	ctx := service.ClientTransaction{
		Instructions: []service.Instruction{
			service.Instruction{
				InstanceID: service.NewInstanceID(owner_darc.GetBaseID()),
				Nonce:      service.Nonce{},
				Index:      0,
				Length:     2,
				Spawn: &service.Spawn{
					ContractID: service.ContractDarcID,
					Args: []service.Argument{{
						Name:  "darc",
						Value: new_darc_buff,
					}},
				},
			},
			service.Instruction{
				InstanceID: service.NewInstanceID(list_darc.GetBaseID()),
				Nonce:      service.Nonce{},
				Index:      1,
				Length:     2,
				Invoke: &service.Invoke{
					Command: "evolve",
					Args: []service.Argument{{
						Name:  "darc",
						Value: new_list_darc_buff,
					}},
				},
			},
		},
	}

	transaction_bytes, err := network.Marshal(&ctx)
	if err != nil {
		return "", nil, err
	}
	transaction_b64 := base64.StdEncoding.EncodeToString(transaction_bytes)
	return transaction_b64, new_darc, nil
}

func getMetadata(request *messages.AddGenericUserRequest) (*metadata.Hospital, *darc.Identity, error) {
	var identity darc.Identity
	if request.PublicKey != "" {
		identity = medChainUtils.LoadIdentityEd25519FromBytes([]byte(request.PublicKey))
	} else {
		return nil, nil, errors.New("No public key was given for the new user")
	}
	super_admin_id := request.SuperAdminIdentity
	if super_admin_id == "" {
		return nil, nil, errors.New("No identity was given for the super admin")
	}
	hospital_metadata, ok := metaData.Hospitals[super_admin_id]
	if !ok {
		return nil, nil, errors.New("No super admin with this id")
	}
	return hospital_metadata, &identity, nil
}

func getListDarc(hospital_metadata *metadata.Hospital, user_type string) (*darc.Darc, error) {
	var list_darc *darc.Darc
	var ok bool
	switch user_type {
	case "Admin":
		list_darc, ok = metaData.BaseIdToDarcMap[hospital_metadata.AdminListDarcBaseId]
	case "Manager":
		list_darc, ok = metaData.BaseIdToDarcMap[hospital_metadata.ManagerListDarcBaseId]
	case "User":
		list_darc, ok = metaData.BaseIdToDarcMap[hospital_metadata.UserListDarcBaseId]
	default:
		return nil, errors.New("Wrong user type")
	}
	if !ok {
		return nil, errors.New("Could not find list darc")
	}
	return list_darc, nil
}

func getSigners(hospital_metadata *metadata.Hospital, user_type string) (*darc.Darc, []string, int, error) {
	var owner_darc *darc.Darc
	var threshold int
	var ok bool
	switch user_type {
	case "Admin":
		owner_darc, ok = metaData.BaseIdToDarcMap[hospital_metadata.DarcBaseId]
		threshold = 1
	default:
		owner_darc, ok = metaData.BaseIdToDarcMap[hospital_metadata.AdminListDarcBaseId]
		threshold = 2
	}
	if !ok {
		return nil, nil, 0, errors.New("Could not find the owner darc")
	}
	hash_map := make(map[string]bool)
	err := recursivelyFindUsersOfDarc(owner_darc, &hash_map)
	if err != nil {
		return nil, nil, 0, err
	}
	signers := []string{}
	for user_id, _ := range hash_map {
		signers = append(signers, user_id)
	}
	return owner_darc, signers, threshold, nil
}
