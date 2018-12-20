package main

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"

	"github.com/dedis/cothority/omniledger/darc"
	"github.com/dedis/cothority/omniledger/service"
	"github.com/talhaparacha/medChain/medChainServer/messages"
	"github.com/talhaparacha/medChain/medChainServer/metadata"
	"github.com/talhaparacha/medChain/medChainUtils"
)

func CommitHospital(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if medChainUtils.CheckError(err, w, r) {
		return
	}
	var request messages.CommitHospitalRequest
	err = json.Unmarshal(body, &request)
	if medChainUtils.CheckError(err, w, r) {
		return
	}

	transaction, err := extractTransactionFromString(request.Transaction)
	if medChainUtils.CheckError(err, w, r) {
		return
	}
	new_darcs, evolved_darcs, err := checkTransactionForNewHospital(transaction)
	if medChainUtils.CheckError(err, w, r) {
		return
	}
	err = submitTransactionForNewHospitals(transaction, new_darcs, evolved_darcs)
	if medChainUtils.CheckError(err, w, r) {
		return
	}
	hospital_metadata, err := adaptMetadataForNewHospital(new_darcs, evolved_darcs)
	if medChainUtils.CheckError(err, w, r) {
		return
	}
	reply := messages.SuperAdminInfoReply{DarcBaseId: hospital_metadata.DarcBaseId, SuperAdminId: hospital_metadata.Id.String(), HospitalName: hospital_metadata.Name, AdminListDarcBaseId: hospital_metadata.AdminListDarcBaseId, ManagerListDarcBaseId: hospital_metadata.ManagerListDarcBaseId, UserListDarcBaseId: hospital_metadata.UserListDarcBaseId, IsCreated: hospital_metadata.IsCreated}
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

func checkTransactionForNewHospital(transaction *service.ClientTransaction) ([]*darc.Darc, []*darc.Darc, error) {
	if len(transaction.Instructions) < 8 {
		return nil, nil, errors.New("Not enough instructions")
	}
	new_darcs := []*darc.Darc{}
	for i := 0; i < 4; i++ {
		new_darc, err := checkSpawnDarcInstructionForGenericUser(transaction.Instructions[i])
		if err != nil {
			return nil, nil, err
		}
		new_darcs = append(new_darcs, new_darc)
	}

	evolved_darcs := []*darc.Darc{}
	for i := 4; i < 8; i++ {
		evolved_darc, err := checkEvolveDarcInstructionForGenericUser(transaction.Instructions[1])
		if err != nil {
			return nil, nil, err
		}
		evolved_darcs = append(evolved_darcs, evolved_darc)
	}

	new_darc_base_id := medChainUtils.IDToB64String(new_darcs[0].GetBaseID())
	hospital_metadata, ok := metaData.HospitalWaitingForCreation[new_darc_base_id]
	if !ok {
		return nil, nil, errors.New("Could not find the metadata of the new hospital")
	}

	if hospital_metadata.IsCreated {
		return nil, nil, errors.New("This hospital was already created")
	}

	test_base_ids := []string{metaData.AllSuperAdminsDarcBaseId, metaData.AllAdminsDarcBaseId, metaData.AllManagersDarcBaseId, metaData.AllUsersDarcBaseId}

	for i, evolved_darc := range evolved_darcs {
		test_base_id := test_base_ids[i]
		evolved_darc_base_id := medChainUtils.IDToB64String(evolved_darc.GetBaseID())
		if test_base_id != evolved_darc_base_id {
			return nil, nil, errors.New("The evolved darc doesn't correspond to the right all darc")
		}
	}

	return new_darcs, evolved_darcs, nil
}

func adaptMetadataForNewHospital(new_darcs, evolved_darcs []*darc.Darc) (*metadata.Hospital, error) {
	new_darc_base_id := medChainUtils.IDToB64String(new_darcs[0].GetBaseID())
	hospital_metadata, ok := metaData.HospitalWaitingForCreation[new_darc_base_id]
	if !ok {
		return nil, errors.New("Could not find the metadata of the new hospital")
	}
	if hospital_metadata.IsCreated {
		return nil, errors.New("This user was already created")
	}
	hospital_metadata.DarcBaseId = addDarcToMaps(new_darcs[0], metaData)
	hospital_metadata.AdminListDarcBaseId = addDarcToMaps(new_darcs[1], metaData)
	hospital_metadata.ManagerListDarcBaseId = addDarcToMaps(new_darcs[2], metaData)
	hospital_metadata.UserListDarcBaseId = addDarcToMaps(new_darcs[3], metaData)
	hospital_metadata.IsCreated = true
	addDarcToMaps(evolved_darcs[0], metaData)
	addDarcToMaps(evolved_darcs[1], metaData)
	addDarcToMaps(evolved_darcs[2], metaData)
	addDarcToMaps(evolved_darcs[3], metaData)
	return hospital_metadata, nil
}

func submitTransactionForNewHospitals(transaction *service.ClientTransaction, new_darcs, evolved_darcs []*darc.Darc) error {
	// Commit transaction
	if _, err := cl.AddTransaction(*transaction); err != nil {
		return err
	}

	for _, new_darc := range new_darcs {
		// // Verify DARC creation
		instID := service.NewInstanceID(new_darc.GetBaseID())
		pr, err := cl.WaitProof(instID, metaData.GenesisMsg.BlockInterval, nil)
		if err != nil || pr.InclusionProof.Match() == false {
			return errors.New("Could not get proof of the darc creation")
		}
	}

	for _, evolved_darc := range evolved_darcs {
		// Verify DARC evolution
		instID := service.NewInstanceID(evolved_darc.GetBaseID())
		// darcBuf, err := evolved_darc.ToProto()
		// if err != nil {
		// 	return err
		// }
		pr, err := cl.WaitProof(instID, metaData.GenesisMsg.BlockInterval, nil)
		if err != nil || pr.InclusionProof.Match() == false {
			return errors.New("Could not get proof of the darc evolution")
		}
	}
	return nil
}
