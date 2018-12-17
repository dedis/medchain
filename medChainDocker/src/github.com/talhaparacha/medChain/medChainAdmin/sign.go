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
	err = signTransaction(transaction, request.Digest, request.SignerIndex, signer)
	if medChainUtils.CheckError(err, w, r) {
		return
	}
}

func signTransaction(transaction *service.ClientTransaction, digest []byte, signer_index int, signer darc.Signer) error {
	for _, instruction := range transaction.Instructions {
		if err := signInstruction(&instruction, digest, signer_index, signer); err != nil {
			return err
		}
	}
	return nil
}

func signInstruction(instruction *service.Instruction, digest []byte, signer_index int, local_signer darc.Signer) error {
	sig, err := local_signer.Sign(digest)
	if err != nil {
		return err
	}
	instruction.Signatures[signer_index] = darc.Signature{
		Signature: sig,
		Signer:    local_signer.Identity(),
	}
	return nil
}
