package main

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"

	"github.com/DPPH/MedChain/medChainServer/messages"
	"github.com/DPPH/MedChain/medChainServer/metadata"
	"github.com/DPPH/MedChain/medChainUtils"
	"github.com/dedis/cothority/omniledger/darc"
	"github.com/dedis/cothority/omniledger/darc/expression"
	"github.com/dedis/cothority/omniledger/service"
)

func AddHospital(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if medChainUtils.CheckError(err, w, r) {
		return
	}
	var request messages.AddHospitalRequest
	err = json.Unmarshal(body, &request)
	if medChainUtils.CheckError(err, w, r) {
		return
	}
	identity, transaction, signers, digests, err := prepareNewHospital(&request)
	if medChainUtils.CheckError(err, w, r) {
		return
	}
	reply := messages.ActionReply{Initiator: request.Initiator, ActionType: "add new hospital", Ids: []string{identity}, Transaction: transaction, Signers: signers, InstructionDigests: digests}
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

func prepareNewHospital(request *messages.AddHospitalRequest) (string, string, map[string]int, map[int][]byte, error) {

	initiator_metadata, ok := metaData.GenericUsers[request.Initiator]
	if !ok {
		return "", "", nil, nil, errors.New("Could not find the initiator metadata")
	}
	if !initiator_metadata.IsCreated {
		return "", "", nil, nil, errors.New("The initiator was not approved")
	}
	if initiator_metadata.Role != "super_admin" {
		return "", "", nil, nil, errors.New("You need to be the head of hospital to add a new hospital")
	}

	identityPtr, err := getSuperAdminId(request)
	if err != nil {
		return "", "", nil, nil, err
	}
	identity := *identityPtr

	genesis_darc, signers_ids, signers, err := getGenesisDarcSigners()
	if err != nil {
		return "", "", nil, nil, err
	}

	transaction, new_darcs, err := createNewHospitalTransaction(request.HospitalName, identity, genesis_darc)
	if err != nil {
		return "", "", nil, nil, err
	}

	base_darcs := []*darc.Darc{genesis_darc, genesis_darc, genesis_darc, genesis_darc, new_darcs[1], new_darcs[2], new_darcs[3], new_darcs[4]}
	digests, err := computeTransactionDigests(transaction, signers_ids, base_darcs)
	if err != nil {
		return "", "", nil, nil, err
	}

	if err := addHospitalToMetadata(metaData, identity, request.HospitalName, request.SuperAdminName, new_darcs[0]); err != nil {
		return "", "", nil, nil, err
	}

	transaction_string, err := transactionToString(transaction)
	if err != nil {
		return "", "", nil, nil, err
	}

	return identity.String(), transaction_string, signers, digests, nil
}

func getSuperAdminId(request *messages.AddHospitalRequest) (*darc.Identity, error) {
	var identity darc.Identity
	var err error
	if request.PublicKey != "" {
		identity, err = medChainUtils.LoadIdentityEd25519FromBytesWithErr([]byte(request.PublicKey))
		if err != nil {
			return nil, err
		}
	} else {
		return nil, errors.New("No public key was given for the new user")
	}
	return &identity, nil
}

func getGenesisDarcSigners() (*darc.Darc, map[int]darc.Identity, map[string]int, error) {

	genesis_darc, ok := metaData.BaseIdToDarcMap[metaData.GenesisDarcBaseId]
	if !ok {
		return nil, nil, nil, errors.New("Could not find the genesis darc")
	}
	hash_map := make(map[string]bool)
	err := recursivelyFindUsersOfDarc(genesis_darc, &hash_map)
	if err != nil {
		return nil, nil, nil, err
	}

	chosen_signers, err := findSignersOfDarc(genesis_darc)
	if err != nil {
		return nil, nil, nil, err
	}

	chosen_signers_ids := make(map[int]darc.Identity)

	chosen_signers_to_index := make(map[string]int)

	for i, signer := range chosen_signers {
		chosen_signers_to_index[signer] = i
	}

	for signer, i := range chosen_signers_to_index {
		user_metadata, ok := metaData.Hospitals[signer]
		if !ok {
			return nil, nil, nil, errors.New("Could not find the super admin identity")
		}
		id := user_metadata.SuperAdmin.Id
		chosen_signers_ids[i] = id
	}

	return genesis_darc, chosen_signers_ids, chosen_signers_to_index, nil
}

func getGeneralListDarcs() (*darc.Darc, *darc.Darc, *darc.Darc, *darc.Darc, error) {

	all_super_admins_darc, ok := metaData.BaseIdToDarcMap[metaData.AllSuperAdminsDarcBaseId]
	if !ok {
		return nil, nil, nil, nil, errors.New("Could not find the all super admins darc")
	}

	all_admins_darc, ok := metaData.BaseIdToDarcMap[metaData.AllAdminsDarcBaseId]
	if !ok {
		return nil, nil, nil, nil, errors.New("Could not find the all admins darc")
	}

	all_managers_darc, ok := metaData.BaseIdToDarcMap[metaData.AllManagersDarcBaseId]
	if !ok {
		return nil, nil, nil, nil, errors.New("Could not find the all managers darc")
	}

	all_users_darc, ok := metaData.BaseIdToDarcMap[metaData.AllUsersDarcBaseId]
	if !ok {
		return nil, nil, nil, nil, errors.New("Could not find the all users darc")
	}

	return all_super_admins_darc, all_admins_darc, all_managers_darc, all_users_darc, nil
}

func createNewHospitalTransaction(hospital_name string, identity darc.Identity, genesis_darc *darc.Darc) (*service.ClientTransaction, []*darc.Darc, error) {

	all_super_admins_darc, all_admins_darc, all_managers_darc, all_users_darc, err := getGeneralListDarcs()
	if err != nil {
		return nil, nil, err
	}

	new_hospital_owners := []darc.Identity{darc.NewIdentityDarc(genesis_darc.GetID())}
	new_hospital_rules := darc.InitRulesWith(new_hospital_owners, []darc.Identity{identity}, "invoke:evolve")
	new_hospital_darc := darc.NewDarc(new_hospital_rules, []byte("Darc for a single Hospital"))
	new_hospital_darc_buff, err := new_hospital_darc.ToProto()
	if err != nil {
		return nil, nil, err
	}

	new_admin_list_owners := []darc.Identity{darc.NewIdentityDarc(new_hospital_darc.GetID())}
	new_admin_list_rules := darc.InitRulesWith(new_admin_list_owners, []darc.Identity{}, "invoke:evolve")
	new_admin_list_rules.AddRule("spawn:darc", expression.InitOrExpr())
	new_admin_list_darc := darc.NewDarc(new_admin_list_rules, []byte("List of Admin of Hospital: "+hospital_name))
	new_admin_list_darc_buff, err := new_admin_list_darc.ToProto()
	if err != nil {
		return nil, nil, err
	}

	new_manager_list_owners := []darc.Identity{darc.NewIdentityDarc(new_admin_list_darc.GetID())}
	new_manager_list_rules := darc.InitRulesWith(new_manager_list_owners, []darc.Identity{}, "invoke:evolve")
	new_manager_list_darc := darc.NewDarc(new_manager_list_rules, []byte("List of Manager of Hospital: "+hospital_name))
	new_manager_list_darc_buff, err := new_manager_list_darc.ToProto()
	if err != nil {
		return nil, nil, err
	}

	new_user_list_owners := []darc.Identity{darc.NewIdentityDarc(new_admin_list_darc.GetID())}
	new_user_list_rules := darc.InitRulesWith(new_user_list_owners, []darc.Identity{}, "invoke:evolve")
	new_user_list_darc := darc.NewDarc(new_user_list_rules, []byte("List of User of Hospital: "+hospital_name))
	new_user_list_darc_buff, err := new_user_list_darc.ToProto()
	if err != nil {
		return nil, nil, err
	}

	_, new_all_super_admins_darc_buff, err := evolveAllUserListWithNewDarc(all_super_admins_darc, new_hospital_darc)
	if err != nil {
		return nil, nil, err
	}

	_, new_all_admins_darc_buff, err := evolveAllUserListWithNewDarc(all_admins_darc, new_admin_list_darc)
	if err != nil {
		return nil, nil, err
	}

	_, new_all_managers_darc_buff, err := evolveAllUserListWithNewDarc(all_managers_darc, new_manager_list_darc)
	if err != nil {
		return nil, nil, err
	}

	_, new_all_users_darc_buff, err := evolveAllUserListWithNewDarc(all_users_darc, new_user_list_darc)
	if err != nil {
		return nil, nil, err
	}

	ctx := service.ClientTransaction{
		Instructions: []service.Instruction{
			service.Instruction{
				InstanceID: service.NewInstanceID(genesis_darc.GetBaseID()),
				Nonce:      service.Nonce{},
				Index:      0,
				Length:     8,
				Spawn: &service.Spawn{
					ContractID: service.ContractDarcID,
					Args: []service.Argument{{
						Name:  "darc",
						Value: new_hospital_darc_buff,
					}},
				},
			},
			service.Instruction{
				InstanceID: service.NewInstanceID(genesis_darc.GetBaseID()),
				Nonce:      service.Nonce{},
				Index:      1,
				Length:     8,
				Spawn: &service.Spawn{
					ContractID: service.ContractDarcID,
					Args: []service.Argument{{
						Name:  "darc",
						Value: new_admin_list_darc_buff,
					}},
				},
			},
			service.Instruction{
				InstanceID: service.NewInstanceID(genesis_darc.GetBaseID()),
				Nonce:      service.Nonce{},
				Index:      2,
				Length:     8,
				Spawn: &service.Spawn{
					ContractID: service.ContractDarcID,
					Args: []service.Argument{{
						Name:  "darc",
						Value: new_manager_list_darc_buff,
					}},
				},
			},
			service.Instruction{
				InstanceID: service.NewInstanceID(genesis_darc.GetBaseID()),
				Nonce:      service.Nonce{},
				Index:      3,
				Length:     8,
				Spawn: &service.Spawn{
					ContractID: service.ContractDarcID,
					Args: []service.Argument{{
						Name:  "darc",
						Value: new_user_list_darc_buff,
					}},
				},
			},
			service.Instruction{
				InstanceID: service.NewInstanceID(all_super_admins_darc.GetBaseID()),
				Nonce:      service.Nonce{},
				Index:      4,
				Length:     8,
				Invoke: &service.Invoke{
					Command: "evolve",
					Args: []service.Argument{{
						Name:  "darc",
						Value: new_all_super_admins_darc_buff,
					}},
				},
			},
			service.Instruction{
				InstanceID: service.NewInstanceID(all_admins_darc.GetBaseID()),
				Nonce:      service.Nonce{},
				Index:      5,
				Length:     8,
				Invoke: &service.Invoke{
					Command: "evolve",
					Args: []service.Argument{{
						Name:  "darc",
						Value: new_all_admins_darc_buff,
					}},
				},
			},
			service.Instruction{
				InstanceID: service.NewInstanceID(all_managers_darc.GetBaseID()),
				Nonce:      service.Nonce{},
				Index:      6,
				Length:     8,
				Invoke: &service.Invoke{
					Command: "evolve",
					Args: []service.Argument{{
						Name:  "darc",
						Value: new_all_managers_darc_buff,
					}},
				},
			},
			service.Instruction{
				InstanceID: service.NewInstanceID(all_users_darc.GetBaseID()),
				Nonce:      service.Nonce{},
				Index:      7,
				Length:     8,
				Invoke: &service.Invoke{
					Command: "evolve",
					Args: []service.Argument{{
						Name:  "darc",
						Value: new_all_users_darc_buff,
					}},
				},
			},
		},
	}
	darcs := []*darc.Darc{new_hospital_darc, all_super_admins_darc, all_admins_darc, all_managers_darc, all_users_darc}
	return &ctx, darcs, nil
}

func evolveAllUserListWithNewDarc(all_darc *darc.Darc, new_darc *darc.Darc) (*darc.Darc, []byte, error) {
	new_all_darc := all_darc.Copy()

	hash_map := make(map[string]bool)
	err = recursivelyFindUsersOfDarc(all_darc, &hash_map)
	if err != nil {
		return nil, nil, err
	}
	signers := []string{}
	for user_id, _ := range hash_map {
		signers = append(signers, user_id)
	}

	new_signer := new_darc.GetIdentityString()

	_, ok := hash_map[new_signer]
	if !ok {
		signers = append(signers, new_signer)
	}

	new_all_darc.EvolveFrom(all_darc)
	new_sign_expr := expression.InitOrExpr(signers...)
	new_all_darc.Rules.UpdateSign(new_sign_expr)

	new_all_darc_buff, err := new_all_darc.ToProto()
	if err != nil {
		return nil, nil, err
	}
	return new_all_darc, new_all_darc_buff, nil
}

func addHospitalToMetadata(metaData *metadata.Metadata, identity darc.Identity, hospital_name, super_admin_name string, new_hospital_darc *darc.Darc) error {

	if hospital_name == "" || super_admin_name == "" {
		return errors.New("You have to specify a name for the hospital and one for its super admin")
	}
	hospital_metadata, ok := metaData.Hospitals[identity.String()]
	if ok {
		if hospital_metadata.SuperAdmin.IsCreated {
			return errors.New("This hospitals already exists")
		}
		base_id := medChainUtils.IDToB64String(new_hospital_darc.GetBaseID())
		metaData.WaitingForCreation[base_id] = hospital_metadata.SuperAdmin
	} else {
		hospital_metadata, super_admin_metadata := metadata.NewHospital(identity, hospital_name, super_admin_name)
		metaData.Hospitals[identity.String()] = hospital_metadata
		metaData.GenericUsers[identity.String()] = super_admin_metadata
		base_id := medChainUtils.IDToB64String(new_hospital_darc.GetBaseID())
		metaData.WaitingForCreation[base_id] = super_admin_metadata
	}
	return nil
}
