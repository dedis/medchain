package main

import (
	"encoding/base64"
	"errors"
	"fmt"

	"github.com/BurntSushi/toml"
	"github.com/DPPH/MedChain/medChainServer/messages"
	"github.com/DPPH/MedChain/medChainServer/metadata"
	"github.com/DPPH/MedChain/medChainUtils"
	"github.com/dedis/cothority"
	"github.com/dedis/cothority/omniledger/darc"
	"github.com/dedis/cothority/omniledger/service"
	"github.com/dedis/onet"
	"github.com/dedis/onet/log"
	"github.com/dedis/onet/network"
	"github.com/dedis/onet/simul/monitor"
)

func init() {
	onet.SimulationRegister("ServiceMedChain", NewSimulationMedChain)
}

// SimulationDrynx state of a simulation.
type SimulationMedChain struct {
	onet.SimulationBFTree
}

// NewSimulationDrynx constructs a full Drynx service simulation.
func NewSimulationMedChain(config string) (onet.Simulation, error) {
	sl := &SimulationMedChain{}
	_, err := toml.Decode(config, sl)
	if err != nil {
		return nil, err
	}

	return sl, nil
}

// Setup creates the tree used for that simulation
func (sim *SimulationMedChain) Setup(dir string, hosts []string) (*onet.SimulationConfig, error) {
	sc := &onet.SimulationConfig{}
	sim.CreateRoster(sc, hosts, 2000)
	err := sim.CreateTree(sc)

	if err != nil {
		return nil, err
	}

	log.Lvl2("Setup done")

	return sc, nil
}

// func (sim *SimulationMedChain) Node(config *onet.SimulationConfig) error {
// 	network.RegisterMessage(&service.ClientTransaction{})
// 	return nil
// }

// Run starts the simulation.
func (sim *SimulationMedChain) Run(config *onet.SimulationConfig) error {
	log.Lvl1("Start")
	metaData = metadata.NewMetadata()
	roster = config.Roster
	// omniledger client
	cl = service.NewClient()

	timer := monitor.NewTimeMeasure("_BootstrapProcess")
	startSystem(metaData, "../conf/conf.json")
	fmt.Println("done boot strapping")
	timer.Record()

	addNewUser()
	addNewHospital()
	return nil
}

func addNewUser() {
	public_key, _ := generateKeyPair()
	requestToAdd := messages.AddGenericUserRequest{
		Initiator:          "ed25519:8378f0dcec5f594e0274e991ce811d6b22ec489221b6781fc61e6c56bf2c0495",
		PublicKey:          public_key,
		Name:               "Test",
		SuperAdminIdentity: "ed25519:fc2ea16063dcefddb21795b593bf68f58a39add33eaaf25f6ad99b78644e1351",
		PreferredSigners:   []string{"ed25519:8378f0dcec5f594e0274e991ce811d6b22ec489221b6781fc61e6c56bf2c0495"},
	}
	replyToAdd, err := subFunctionAddGenericUser(&requestToAdd, "User")
	if err != nil {
		panic(err)
	}
	fmt.Println(replyToAdd.Signers)
	transaction_string := replyToAdd.Transaction
	path_1_public := "../keys/admins/0_public_ed25519:8378f0dcec5f594e0274e991ce811d6b22ec489221b6781fc61e6c56bf2c0495"
	path_1_private := "../keys/admins/0_private_ed25519:8378f0dcec5f594e0274e991ce811d6b22ec489221b6781fc61e6c56bf2c0495"
	signer1 := medChainUtils.LoadSignerEd25519(path_1_public, path_1_private)
	path_2_public := "../keys/admins/1_public_ed25519:341a301dffa5d308c2ad7c1807d2d7395ce5e23237cf01a379b3f8260f797b8e"
	path_2_private := "../keys/admins/1_private_ed25519:341a301dffa5d308c2ad7c1807d2d7395ce5e23237cf01a379b3f8260f797b8e"
	signer2 := medChainUtils.LoadSignerEd25519(path_2_public, path_2_private)
	signer_1_meta, ok := metaData.GenericUsers[signer1.Identity().String()]
	if !ok {
		panic(errors.New("could not retrieve the signer metadata"))
	}
	owner_darc, ok := metaData.BaseIdToDarcMap[signer_1_meta.Hospital.AdminListDarcBaseId]
	if !ok {
		panic(errors.New("could not retrieve the owner darc"))
	}
	list_darc, ok := metaData.BaseIdToDarcMap[signer_1_meta.Hospital.UserListDarcBaseId]
	if !ok {
		panic(errors.New("could not retrieve the list darc"))
	}
	base_darcs := []*darc.Darc{owner_darc, list_darc}
	signed_transaction, err := signTransaction(transaction_string, base_darcs, signer1, signer2)
	fmt.Println("Test", signed_transaction != transaction_string)
	if err != nil {
		panic(err)
	}
	replyToCommit, err := subFunctionCommitGenericUser(signed_transaction, "User")
	if err != nil {
		panic(err)
	}
	fmt.Println(replyToCommit.Name)
}

func addNewHospital() {
	public_key, _ := generateKeyPair()
	requestToAdd := messages.AddHospitalRequest{
		Initiator:      "ed25519:fc2ea16063dcefddb21795b593bf68f58a39add33eaaf25f6ad99b78644e1351",
		PublicKey:      public_key,
		HospitalName:   "HospitalTest",
		SuperAdminName: "SuperAdminNameTest",
	}
	replyToAdd, err := subFunctionAddHospital(&requestToAdd)
	if err != nil {
		panic(err)
	}
	fmt.Println(replyToAdd.Signers)
	transaction_string := replyToAdd.Transaction
	path_1_public := "../keys/super_admins/0_public_ed25519:fc2ea16063dcefddb21795b593bf68f58a39add33eaaf25f6ad99b78644e1351"
	path_1_private := "../keys/super_admins/0_private_ed25519:fc2ea16063dcefddb21795b593bf68f58a39add33eaaf25f6ad99b78644e1351"
	signer1 := medChainUtils.LoadSignerEd25519(path_1_public, path_1_private)
	path_2_public := "../keys/super_admins/1_public_ed25519:71f39da8251eabc0ec5d8edc38facae73bec33f650a20639dee30b59ac860975"
	path_2_private := "../keys/super_admins/1_public_ed25519:71f39da8251eabc0ec5d8edc38facae73bec33f650a20639dee30b59ac860975"
	signer2 := medChainUtils.LoadSignerEd25519(path_2_public, path_2_private)

	signed_transaction, err := signTransactionForHospital(replyToAdd, signer1, signer2)
	fmt.Println("Test", signed_transaction != transaction_string)
	if err != nil {
		panic(err)
	}
	replyToCommit, err := subFunctionCommitHospital(signed_transaction)
	if err != nil {
		panic(err)
	}
	fmt.Println(replyToCommit.HospitalName)
}

func signTransactionForHospital(replyToAdd *messages.ActionReply, signers ...darc.Signer) (string, error) {
	transaction_bytes, err := base64.StdEncoding.DecodeString(replyToAdd.Transaction)
	if err != nil {
		panic(err)
	}

	var transaction *service.ClientTransaction

	_, tmp, err := network.Unmarshal(transaction_bytes, cothority.Suite)
	if err != nil {
		panic(err)
	}
	transaction, ok := tmp.(*service.ClientTransaction)
	if !ok {
		panic(errors.New("could not retrieve the transaction"))
	}
	for _, signer := range signers {
		err := signTransactionForHospital2(transaction, replyToAdd.InstructionDigests, replyToAdd.Signers, signer)
		if err != nil {
			panic(err)
		}
	}
	new_transaction_string, err := transactionToString(transaction)
	if err != nil {
		return "", err
	}
	return new_transaction_string, nil
}

func signTransactionForHospital2(transaction *service.ClientTransaction, instruction_digests map[int][]byte, signers map[string]int, signer darc.Signer) error {
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

func signTransaction(transaction_string string, baseDarc []*darc.Darc, signers ...darc.Signer) (string, error) {
	transaction_bytes, err := base64.StdEncoding.DecodeString(transaction_string)
	if err != nil {
		panic(err)
	}

	var transaction *service.ClientTransaction

	_, tmp, err := network.Unmarshal(transaction_bytes, cothority.Suite)
	if err != nil {
		panic(err)
	}
	transaction, ok := tmp.(*service.ClientTransaction)
	if !ok {
		panic(errors.New("could not retrieve the transaction"))
	}
	for i := 0; i < len(transaction.Instructions); i++ {
		fmt.Println(len(transaction.Instructions[i].Signatures))
		err = transaction.Instructions[i].SignBy(baseDarc[i].GetBaseID(), signers...)
		if err != nil {
			panic(err)
		}
		fmt.Println(len(transaction.Instructions[i].Signatures))
	}
	new_transaction_string, err := transactionToString(transaction)
	if err != nil {
		return "", err
	}
	return new_transaction_string, nil
}

func generateKeyPair() (string, string) {
	temp := darc.NewSignerEd25519(nil, nil)

	private, _ := temp.GetPrivate()
	privateInBytes, _ := private.MarshalBinary()
	public := temp.Identity().Ed25519.Point
	publicInBytes, _ := public.MarshalBinary()
	public_string := base64.StdEncoding.EncodeToString(publicInBytes)
	private_string := base64.StdEncoding.EncodeToString(privateInBytes)
	return public_string, private_string
}
