package main

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"

	"github.com/dedis/cothority"
	"github.com/dedis/cothority/omniledger/darc"
	"github.com/dedis/cothority/omniledger/service"
	"github.com/dedis/onet/network"
	"github.com/talhaparacha/medChain/medChainAdmin/admin_messages"
	"github.com/talhaparacha/medChain/medChainUtils"
)

func processSignTransactionRequest(w http.ResponseWriter, r *http.Request) {
	body, err := ioutil.ReadAll(r.Body)
	if medChainUtils.CheckError(err, w, r) {
		return
	}
	var request admin_messages.SignRequest
	err = json.Unmarshal(body, &request)
	if medChainUtils.CheckError(err, w, r) {
		return
	}
	if request.PublicKey == "" && request.PrivateKey == "" {
		medChainUtils.CheckError(errors.New("No public/private key pair was given"), w, r)
		return
	}

	signer := medChainUtils.LoadSignerEd25519FromBytes([]byte(request.PublicKey), []byte(request.PrivateKey))

	transaction_bytes, err := base64.StdEncoding.DecodeString(request.Transaction)
	if medChainUtils.CheckError(err, w, r) {
		return
	}

	// Load the transaction
	var transaction *service.ClientTransaction
	_, tmp, err := network.Unmarshal(transaction_bytes, cothority.Suite)
	if medChainUtils.CheckError(err, w, r) {
		return
	}
	transaction, ok := tmp.(*service.ClientTransaction)
	if !ok {
		medChainUtils.CheckError(errors.New("could not retrieve the transaction"), w, r)
		return
	}
	err = signTransaction(transaction, request.InstructionDigests, request.Signers, signer)
	if medChainUtils.CheckError(err, w, r) {
		return
	}
	transaction_string, err := transactionToString(transaction)
	if medChainUtils.CheckError(err, w, r) {
		return
	}
	delete(request.Signers, signer.Identity().String())
	reply := admin_messages.SignReply{Transaction: transaction_string, InstructionDigests: request.InstructionDigests, Signers: request.Signers}
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

func transactionToString(transaction *service.ClientTransaction) (string, error) {
	transaction_bytes, err := network.Marshal(transaction)
	if err != nil {
		return "", err
	}
	transaction_b64 := base64.StdEncoding.EncodeToString(transaction_bytes)
	return transaction_b64, nil
}

func signTransaction(transaction *service.ClientTransaction, instruction_digests map[int][]byte, signers map[string]int, signer darc.Signer) error {
	if len(instruction_digests) != len(transaction.Instructions) {
		return errors.New("You should provide as many digests as intructions")
	}
	signer_index, ok := signers[signer.Identity().String()]
	if !ok {
		return errors.New("Your identity is not in the signers list")
	}
	for i, instruction := range transaction.Instructions {
		if err := signInstruction(&instruction, instruction_digests[i], signer_index, signer); err != nil {
			return err
		}
		transaction.Instructions[i] = instruction
	}
	return nil
}

func signInstruction(instruction *service.Instruction, digest []byte, signer_index int, local_signer darc.Signer) error {
	sig, err := local_signer.Sign(digest)
	if err != nil {
		return err
	}
	instruction.Signatures[signer_index].Signature = sig
	return nil
}
