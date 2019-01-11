package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"sort"

	"github.com/DPPH/MedChain/medChainServer/messages"
	"github.com/DPPH/MedChain/medChainServer/metadata"
	"github.com/DPPH/MedChain/medChainUtils"
	"github.com/DPPH/MedChain/signingService/signing_messages"
	"github.com/dedis/cothority/omniledger/darc"
	"github.com/dedis/cothority/omniledger/darc/expression"
	"github.com/dedis/cothority/omniledger/service"
	"github.com/dedis/onet/network"
)

/**
This file takes care of preparing the transaction for adding a new generic user (= admin, manager, users) to the service
**/

/**
We have a different api entry point for each type of user
**/
func AddAdmin(w http.ResponseWriter, r *http.Request) {
	fmt.Println("/add/admin")
	replyNewGenericUserRequest(w, r, "Admin")
}

func AddUser(w http.ResponseWriter, r *http.Request) {
	fmt.Println("/add/user")
	replyNewGenericUserRequest(w, r, "User")
}

func AddManager(w http.ResponseWriter, r *http.Request) {
	fmt.Println("/add/manager")
	replyNewGenericUserRequest(w, r, "Manager")
}

/**
This function prepares the transaction for adding a new generic user (= admin, manager, users)
The request r must contain a messages.AddGenericUserRequest in its body (encoded in json)
The user_type can be either "Admin", "Manager", or "User"
It returns a messages.ActionReply (encoded in json) in the body of the response w
Or an error message
**/
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
	reply, err := subFunctionAddGenericUser(&request, user_type)
	if medChainUtils.CheckError(err, w, r) {
		return
	}
	json_val, err := json.Marshal(reply)
	if medChainUtils.CheckError(err, w, r) {
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_, err = w.Write(json_val)
	if medChainUtils.CheckError(err, w, r) {
		return
	}
}

func subFunctionAddGenericUser(request *messages.AddGenericUserRequest, user_type string) (*messages.ActionReply, error) {
	identity, transaction, signers, digests, err := prepareNewUser(request, user_type)
	if err != nil {
		return nil, err
	}
	reply := messages.ActionReply{Initiator: request.Initiator, ActionType: "add new " + user_type, Ids: []string{identity}, Transaction: transaction, Signers: signers, InstructionDigests: digests}
	return &reply, nil
}

func prepareNewUser(request *messages.AddGenericUserRequest, user_type string) (string, string, map[string]int, map[int][]byte, error) {

	initiator_metadata, ok := metaData.GenericUsers[request.Initiator]
	if !ok {
		return "", "", nil, nil, errors.New("Could not find the initiator metadata")
	}
	if !initiator_metadata.IsCreated {
		return "", "", nil, nil, errors.New("The initiator was not approved")
	}
	if user_type == "Admin" && initiator_metadata.Role != "super_admin" {
		return "", "", nil, nil, errors.New("You need to be the head of hospital to add a new admin")
	}
	if user_type != "Admin" && initiator_metadata.Role != "admin" {
		return "", "", nil, nil, errors.New("You need to be an admin to add a new " + user_type)
	}

	err = checkRequestForNewUser(request)
	if err != nil {
		return "", "", nil, nil, err
	}

	hospital_metadata, identityPtr, err := getMetadata(request)
	if err != nil {
		return "", "", nil, nil, err
	}
	identity := *identityPtr

	if _, ok := metaData.GenericUsers[identity.String()]; ok {
		return "", "", nil, nil, errors.New("There is already a user with that public key")
	}

	owner_darc, signers_ids, signers, err := getSigners(hospital_metadata, user_type, request.PreferredSigners)
	if err != nil {
		return "", "", nil, nil, err
	}

	list_darc, err := getListDarc(hospital_metadata, user_type)
	if err != nil {
		return "", "", nil, nil, err
	}

	transaction, new_darc, err := createNewGenericUserTransaction(identity, owner_darc, list_darc, user_type)
	if err != nil {
		return "", "", nil, nil, err
	}
	base_darcs := []*darc.Darc{owner_darc, list_darc}
	digests, err := computeTransactionDigests(transaction, signers_ids, base_darcs)
	if err != nil {
		return "", "", nil, nil, err
	}

	if err := addGenericUserToMetadata(metaData, hospital_metadata, identity, request.Name, user_type, new_darc); err != nil {
		return "", "", nil, nil, err
	}

	transaction_string, err := transactionToString(transaction)
	if err != nil {
		return "", "", nil, nil, err
	}

	return identity.String(), transaction_string, signers, digests, nil
}

func checkRequestForNewUser(request *messages.AddGenericUserRequest) error {
	if request.Name == "" {
		return errors.New("You need to provide a name")
	}
	if request.PublicKey == "" {
		return errors.New("You need to provide a public_key")
	}
	return nil
}

func transactionToString(transaction *service.ClientTransaction) (string, error) {
	transaction_bytes, err := network.Marshal(transaction)
	if err != nil {
		return "", err
	}
	transaction_b64 := base64.StdEncoding.EncodeToString(transaction_bytes)

	return transaction_b64, nil
}

func computeTransactionDigests(transaction *service.ClientTransaction, signers_ids map[int]darc.Identity, base_darcs []*darc.Darc) (map[int][]byte, error) {
	result := map[int][]byte{}
	for i, instruction := range transaction.Instructions {
		digest, err := computeInstructionDigests(&instruction, signers_ids, base_darcs[i])
		if err != nil {
			return nil, err
		}
		transaction.Instructions[i] = instruction
		result[i] = digest
	}

	return result, nil
}

func computeInstructionDigests(instruction *service.Instruction, signers_ids map[int]darc.Identity, base_darc *darc.Darc) ([]byte, error) {
	// Create the request and populate it with the right identities.  We
	// need to do this prior to signing because identities are a part of
	// the digest of the Instruction.
	sigs := make([]darc.Signature, len(signers_ids))
	for i, signer := range signers_ids {
		sigs[i].Signer = signer
	}
	instruction.Signatures = sigs

	req, err := instruction.ToDarcRequest(base_darc.GetBaseID())
	if err != nil {
		return nil, err
	}

	req.Identities = make([]darc.Identity, len(signers_ids))
	for i, signer := range signers_ids {
		req.Identities[i] = signer
	}
	// Sign the instruction and write the signatures to it.
	digest := req.Hash()
	return digest, nil
}

func addGenericUserToMetadata(metaData *metadata.Metadata, hospital_metadata *metadata.Hospital, identity darc.Identity, name, user_type string, new_darc *darc.Darc) error {
	if name == "" {
		return errors.New("You have to specify a name")
	}

	if _, ok := metaData.GenericUsers[identity.String()]; ok {
		return errors.New("This user already exists")
	} else {
		var user_metadata *metadata.GenericUser
		switch user_type {
		case "Admin":
			user_metadata = metadata.NewAdmin(identity, name, hospital_metadata)
		case "Manager":
			user_metadata = metadata.NewManager(identity, name, hospital_metadata)
		case "User":
			user_metadata = metadata.NewUser(identity, name, hospital_metadata)
		default:
			return errors.New("Unknown user type")
		}
		metaData.GenericUsers[identity.String()] = user_metadata
		base_id := medChainUtils.IDToB64String(new_darc.GetBaseID())
		metaData.WaitingForCreation[base_id] = user_metadata
	}
	return nil
}

func createNewGenericUserTransaction(identity darc.Identity, owner_darc, list_darc *darc.Darc, user_type string) (*service.ClientTransaction, *darc.Darc, error) {

	owners := []darc.Identity{darc.NewIdentityDarc(owner_darc.GetID())}
	rules := darc.InitRulesWith(owners, []darc.Identity{identity}, "invoke:evolve")
	new_darc := darc.NewDarc(rules, []byte("Darc for a single "+user_type))
	new_darc_buff, err := new_darc.ToProto()
	if err != nil {
		return nil, nil, err
	}

	new_list_darc := list_darc.Copy()

	hash_map := make(map[string]bool)
	err = recursivelyFindUsersOfDarc(list_darc, &hash_map)
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

	new_list_darc.EvolveFrom(list_darc)
	new_sign_expr := expression.InitOrExpr(signers...)
	new_list_darc.Rules.UpdateSign(new_sign_expr)
	if user_type == "Admin" {
		new_list_darc.Rules.UpdateRule("spawn:darc", medChainUtils.InitAtLeastTwoExpr(signers))
	}
	new_list_darc_buff, err := new_list_darc.ToProto()
	if err != nil {
		return nil, nil, err
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
	return &ctx, new_darc, nil
}

func getMetadata(request *messages.AddGenericUserRequest) (*metadata.Hospital, *darc.Identity, error) {
	var identity darc.Identity
	if request.PublicKey != "" {
		identity, err = medChainUtils.LoadIdentityEd25519FromBytesWithErr([]byte(request.PublicKey))
		if err != nil {
			return nil, nil, err
		}
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
	if !hospital_metadata.SuperAdmin.IsCreated {
		return nil, nil, errors.New("This hospital wasn't approved")
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

func getSigners(hospital_metadata *metadata.Hospital, user_type string, preferred_signers []string) (*darc.Darc, map[int]darc.Identity, map[string]int, error) {
	var owner_darc *darc.Darc
	var threshold int
	var ok bool
	switch user_type {
	case "Admin":
		owner_darc, ok = metaData.BaseIdToDarcMap[hospital_metadata.SuperAdmin.DarcBaseId]
		threshold = 1
	default:
		owner_darc, ok = metaData.BaseIdToDarcMap[hospital_metadata.AdminListDarcBaseId]
		threshold = 2
	}
	if !ok {
		return nil, nil, nil, errors.New("Could not find the owner darc")
	}
	hash_map := make(map[string]bool)
	err := recursivelyFindUsersOfDarc(owner_darc, &hash_map)
	if err != nil {
		return nil, nil, nil, err
	}

	chosen_signers := make([]string, 0)

	for _, preferred := range preferred_signers {
		_, ok := hash_map[preferred]
		if ok {
			chosen_signers = append(chosen_signers, preferred)
			delete(hash_map, preferred)
		}
	}

	signers := []string{}
	for user_id, _ := range hash_map {
		signers = append(signers, user_id)
	}

	remainer := min(threshold-len(chosen_signers), len(signers))

	if remainer > 0 {
		chosen_signers = append(chosen_signers, signers[:remainer]...)
	}
	sort.Strings(chosen_signers)

	chosen_signers_ids := make(map[int]darc.Identity)

	chosen_signers_to_index := make(map[string]int)

	for i, signer := range chosen_signers {
		chosen_signers_to_index[signer] = i
	}

	for signer, i := range chosen_signers_to_index {
		switch user_type {
		case "Admin":
			user_metadata, ok := metaData.Hospitals[signer]
			if !ok {
				return nil, nil, nil, errors.New("Could not find the signer identity")
			}
			id := user_metadata.SuperAdmin.Id
			chosen_signers_ids[i] = id
		default:
			user_metadata, ok := metaData.GenericUsers[signer]
			if !ok {
				return nil, nil, nil, errors.New("Could not find the signer identity")
			}
			id := user_metadata.Id
			chosen_signers_ids[i] = id
		}

	}

	return owner_darc, chosen_signers_ids, chosen_signers_to_index, nil
}

func min(a int, b int) int {
	if a < b {
		return a
	}
	return b
}

func sendNewActionToSigningService(action *messages.ActionReply) error {
	request := signing_messages.AddNewActionRequest{Action: action}
	request_bytes, err := json.Marshal(&request)
	if err != nil {
		return err
	}
	response, err := http.Post(metaData.SigningServiceUrl+"/add/action", "application/json", bytes.NewBuffer(request_bytes))
	if err != nil {
		return err
	}
	if response.StatusCode != 200 {
		body, _ := ioutil.ReadAll(response.Body)
		print("submit error", string(body))
		return errors.New("Could not send the new action to the signing service")
	}
	return nil
}
