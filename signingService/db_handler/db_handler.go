package db_handler

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/DPPH/MedChain/medChainServer/messages"
	"github.com/DPPH/MedChain/signingService/signing_messages"
	"github.com/DPPH/MedChain/signingService/status"
	"github.com/google/uuid"
)

func RegisterNewAction(action *messages.ActionReply, db *sql.DB) (string, error) {
	// fmt.Println("adding new action to the db")
	id := uuid.New().String()
	action_bytes, err := json.Marshal(action)
	if err != nil {
		return "", nil
	}
	action_string := string(action_bytes)
	err = addNewActionToDB(id, action.Initiator, status.Waiting, action_string, db)
	if err != nil {
		return "", nil
	}
	for signer, _ := range action.Signers {
		err = addNewSignerToDB(id, signer, false, db)
		if err != nil {
			return "", nil
		}
	}
	return id, nil
}

func addNewActionToDB(id, initiator, status, action_string string, db *sql.DB) error {
	// fmt.Println("adding new action to the db")
	stmt, err := db.Prepare("INSERT INTO Action(id, initiator, status, action_value) VALUES(?,?,?,?);")
	if err != nil {
		return err
	}
	_, err = stmt.Exec(id, initiator, status, action_string)
	return err
}

func addNewSignerToDB(id, signer_identity string, signed bool, db *sql.DB) error {
	// fmt.Println("adding new signer to the db")
	stmt, err := db.Prepare("INSERT INTO SignatureStatus(action_id, signer_identity, signed) VALUES(?,?,?);")
	if err != nil {
		return err
	}
	_, err = stmt.Exec(id, signer_identity, signed)
	return err
}

func GetActionsInitiatedBy(id string, db *sql.DB) ([]*signing_messages.ActionInfoReply, error) {
	action_ids, err := getActionIdsInitiatedBy(id, db)
	if err != nil {
		return nil, err
	}
	info_list := []*signing_messages.ActionInfoReply{}
	for _, action_id := range action_ids {
		action_info, err := GetInfoForAction(action_id, db)
		if err != nil && err != sql.ErrNoRows {
			return nil, err
		} else if err == nil {
			info_list = append(info_list, action_info)
		}
	}
	return info_list, nil
}

func GetActionsWaitingFor(id string, db *sql.DB) ([]*signing_messages.ActionInfoReply, error) {
	action_ids, err := getActionIdsWaitingFor(id, db)
	if err != nil {
		return nil, err
	}
	fmt.Println()
	info_list := []*signing_messages.ActionInfoReply{}
	for _, action_id := range action_ids {
		action_info, err := GetInfoForAction(action_id, db)
		if err != nil && err != sql.ErrNoRows {
			return nil, err
		} else if err == nil {
			info_list = append(info_list, action_info)
		}
	}
	return info_list, nil
}

func getActionIdsWaitingFor(id string, db *sql.DB) ([]string, error) {
	statement := "SELECT action_id FROM SignatureStatus, Action WHERE id = action_id AND status =? AND signer_identity=? AND signed=0;"
	stmt, err := db.Prepare(statement)
	if err != nil {
		return nil, err
	}
	rows, err := stmt.Query(status.Waiting, id)
	if err != nil {
		return nil, err
	}
	action_ids := []string{}
	for rows.Next() {
		var action_id string
		err = rows.Scan(&action_id)
		if err != nil {
			return nil, err
		}
		action_ids = append(action_ids, action_id)
	}
	rows.Close()
	return action_ids, nil
}

func getActionIdsInitiatedBy(id string, db *sql.DB) ([]string, error) {
	statement := "SELECT id FROM Action WHERE initiator=?;"
	stmt, err := db.Prepare(statement)
	if err != nil {
		return nil, err
	}
	rows, err := stmt.Query(id)
	if err != nil {
		return nil, err
	}
	action_ids := []string{}
	for rows.Next() {
		var action_id string
		err = rows.Scan(&action_id)
		if err != nil {
			return nil, err
		}
		action_ids = append(action_ids, action_id)
	}
	rows.Close()
	return action_ids, nil
}

func GetInfoForAction(action_id string, db *sql.DB) (*signing_messages.ActionInfoReply, error) {
	statement := "SELECT initiator, status, action_value FROM Action WHERE id=?;"
	stmt, err := db.Prepare(statement)
	if err != nil {
		return nil, err
	}
	row := stmt.QueryRow(action_id)
	var initiator, status, action_value string
	err = row.Scan(&initiator, &status, &action_value)
	if err != nil {
		fmt.Println("could not scan info")
		return nil, err
	}
	signatures, err := getActionSignatureMap(action_id, db)
	if err != nil {
		fmt.Println("could not scan signature")
		return nil, err
	}
	action_reply, err := fromJSONStringToActionReply(action_value)
	return &signing_messages.ActionInfoReply{Id: action_id, Initiator: initiator, Status: status, Action: action_reply, Signatures: signatures}, nil
}

func getActionSignatureMap(action_id string, db *sql.DB) (map[string]bool, error) {
	statement := "SELECT signer_identity, signed FROM SignatureStatus WHERE action_id=?;"
	stmt, err := db.Prepare(statement)
	if err != nil {
		return nil, err
	}
	rows, err := stmt.Query(action_id)
	if err != nil {
		return nil, err
	}
	signature_map := make(map[string]bool)
	for rows.Next() {
		var signer_id string
		var signed int
		err = rows.Scan(&signer_id, &signed)
		if err != nil {
			fmt.Println("could not scan signature")
			return nil, err
		}
		signature_map[signer_id] = (signed != 0)
	}
	rows.Close()
	return signature_map, nil
}

func fromJSONStringToActionReply(action_value string) (*messages.ActionReply, error) {
	var reply messages.ActionReply
	err := json.Unmarshal([]byte(action_value), &reply)
	if err != nil {
		return nil, err
	}
	return &reply, nil
}
