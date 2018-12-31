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
		return "", err
	}
	action_string := string(action_bytes)
	err = addNewActionToDB(id, action.Initiator, status.Waiting, action_string, db)
	if err != nil {
		return "", err
	}
	for signer, _ := range action.Signers {
		err = addNewSignerToDB(id, signer, status.SignerWaiting, db)
		if err != nil {
			return "", err
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

func addNewSignerToDB(id, signer_identity string, status string, db *sql.DB) error {
	// fmt.Println("adding new signer to the db")
	stmt, err := db.Prepare("INSERT INTO SignatureStatus(action_id, signer_identity, status) VALUES(?,?,?);")
	if err != nil {
		return err
	}
	_, err = stmt.Exec(id, signer_identity, status)
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
	statement := "SELECT action_id FROM SignatureStatus AS Sig, Action WHERE Action.id = Sig.action_id AND Action.status =? AND Sig.signer_identity=? AND Sig.Status=?;"
	stmt, err := db.Prepare(statement)
	if err != nil {
		return nil, err
	}
	rows, err := stmt.Query(status.Waiting, id, status.SignerWaiting)
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
	signatures, err := GetActionSignatureMap(action_id, db)
	if err != nil {
		fmt.Println("could not scan signature")
		return nil, err
	}
	action_reply, err := fromJSONStringToActionReply(action_value)
	return &signing_messages.ActionInfoReply{Id: action_id, Initiator: initiator, Status: status, Action: action_reply, Signatures: signatures}, nil
}

func GetActionStatus(action_id string, db *sql.DB) (string, error) {
	statement := "SELECT status FROM Action WHERE id=?;"
	stmt, err := db.Prepare(statement)
	if err != nil {
		return "", err
	}
	row := stmt.QueryRow(action_id)
	var status_val string
	err = row.Scan(&status_val)
	if err != nil {
		return "", err
	}
	return status_val, nil
}

func GetActionInitiator(action_id string, db *sql.DB) (string, error) {
	statement := "SELECT initiator FROM Action WHERE id=?;"
	stmt, err := db.Prepare(statement)
	if err != nil {
		return "", err
	}
	row := stmt.QueryRow(action_id)
	var status_val string
	err = row.Scan(&status_val)
	if err != nil {
		return "", err
	}
	return status_val, nil
}

func GetActionString(action_id string, db *sql.DB) (string, error) {
	statement := "SELECT action_value FROM Action WHERE id=?;"
	stmt, err := db.Prepare(statement)
	if err != nil {
		return "", err
	}
	row := stmt.QueryRow(action_id)
	var action_value string
	err = row.Scan(&action_value)
	if err != nil {
		fmt.Println("could not scan info")
		return "", err
	}
	return action_value, nil
}

func GetActionSignatureMap(action_id string, db *sql.DB) (map[string]string, error) {
	statement := "SELECT signer_identity, status FROM SignatureStatus WHERE action_id=?;"
	stmt, err := db.Prepare(statement)
	if err != nil {
		return nil, err
	}
	rows, err := stmt.Query(action_id)
	if err != nil {
		return nil, err
	}
	signature_map := make(map[string]string)
	for rows.Next() {
		var signer_id, status_val string
		err = rows.Scan(&signer_id, &status_val)
		if err != nil {
			fmt.Println("could not scan signature")
			return nil, err
		}
		signature_map[signer_id] = status_val
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

func UpdateActionValue(id string, new_action_string string, db *sql.DB) error {
	statement := "UPDATE Action SET action_value=? WHERE id=?;"
	stmt, err := db.Prepare(statement)
	if err != nil {
		return err
	}
	_, err = stmt.Exec(new_action_string, id)
	return err
}

func UdpdateSignatureStatus(action_id string, signer_id string, status_val string, db *sql.DB) error {
	statement := "UPDATE SignatureStatus SET status=? WHERE action_id=? AND signer_identity=?;"
	stmt, err := db.Prepare(statement)
	if err != nil {
		return err
	}
	_, err = stmt.Exec(status_val, action_id, signer_id)
	return err
}

func UdpdateActionStatus(action_id, status_val string, db *sql.DB) error {
	statement := "UPDATE Action SET status=? WHERE id=?;"
	stmt, err := db.Prepare(statement)
	if err != nil {
		return err
	}
	_, err = stmt.Exec(status_val, action_id)
	return err
}
