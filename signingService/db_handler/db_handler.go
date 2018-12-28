package db_handler

import (
	"database/sql"
	"encoding/json"

	"github.com/DPPH/MedChain/medChainServer/messages"
	"github.com/DPPH/MedChain/signingService/status"
	"github.com/google/uuid"
)

func RegisterNewAction(action *messages.ActionReply, db *sql.DB) (string, error) {
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
	stmt, err := db.Prepare("INSERT INTO Action(id, initiator, status, action_value) VALUES(?,?,?,?);")
	if err != nil {
		return err
	}
	_, err = stmt.Exec(id, initiator, status, action_string)
	return err
}

func addNewSignerToDB(id, signer_identity string, signed bool, db *sql.DB) error {
	stmt, err := db.Prepare("INSERT INTO SignatureStatus(action_id, signer_identity, signed) VALUES(?,?,?);")
	if err != nil {
		return err
	}
	_, err = stmt.Exec(id, signer_identity, signed)
	return err
}
