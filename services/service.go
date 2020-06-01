package medchain

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/medchain/protocols"
	"go.dedis.ch/cothority/v3/byzcoin"
	"go.dedis.ch/onet/v3"
	"go.dedis.ch/onet/v3/log"
	"go.dedis.ch/protobuf"
	"golang.org/x/xerrors"
)

// This service is only used because we need to register our contracts to
// the ByzCoin service. So we create this stub and add contracts to it
// from the `contracts` directory.

// ServiceName is the service name for the medchain service.
// It is used for registration on the onet.
var ServiceName = "MedChain"

// ContractName is the name of the contract
const ContractName = "medchain"

var storageKey = []byte("medchainconfig")
var dbVersion = 1

// sid is the onet identifier (service ID).
var sid onet.ServiceID

const defaultBlockInterval = 5 * time.Second

func init() {
	var err error
	sid, err = onet.RegisterNewService(ServiceName, NewService)
	if err != nil {
		log.Fatal(err)
	}

	err = byzcoin.RegisterGlobalContract(ContractName, ContractMedchainFromBytes)
	if err != nil {
		log.ErrFatal(err)
	}
}

// Service is only used to being able to store our contracts
type Service struct {
	// We need to embed the ServiceProcessor, so that incoming messages
	// are correctly handled.
	*onet.ServiceProcessor
	omni        *byzcoin.Service
	PropagateID protocols.PropagationFunc
	Storage     *Storage
}

// Storage is saved to disk.
type Storage struct {
	SharedInstanceIDs []byzcoin.InstanceID
	storageMutex      sync.Mutex
}

// NewService receives the context that holds information about the node it's
// running on. Saving and loading can be done using the context. The data will
// be stored in memory for tests and simulations, and on disk for real
// deployments.
func NewService(c *onet.Context) (onet.Service, error) {
	var err error
	s := &Service{
		ServiceProcessor: onet.NewServiceProcessor(c),
		omni:             c.Service(byzcoin.ServiceName).(*byzcoin.Service),
	}

	if err := s.tryLoad(); err != nil {
		log.Error(err)
		return nil, err
	}

	if err := s.RegisterHandlers(s.HandleSpawnDeferredQuery, s.HandleSignDeferredTx,
		s.HandlePropagateID, s.HandleGetSharedData); err != nil {
		return nil, fmt.Errorf("couldn't register messages: %+v", err)
	}

	s.PropagateID, err = protocols.NewPropagationFunc(s, "MedChainPropagate", -1)
	if err != nil {
		return nil, fmt.Errorf("couldn't create propagation function: %+v", err)
	}

	return s, nil
}

// HandleSpawnDeferredQuery handles requests to submit (= spawn or add) a query
func (s *Service) HandleSpawnDeferredQuery(req *AddDeferredQueryRequest) (*AddDeferredQueryReply, error) {
	reply := &AddDeferredQueryReply{}
	reply.OK = false
	// sanitize params
	if err := emptyInstID(req.QueryInstID); err != nil {
		return reply, xerrors.Errorf("%+v", err)
	}
	if err := checkStatus(req.QueryStatus); err != nil {
		return reply, xerrors.Errorf("%+v", err)
	}

	// if this server is the one receiving the request from the client
	log.Lvl1("[INFO]: ", s.ServerIdentity().String(), " received an AddDeferredQueryRequest for query:", req.QueryID)

	stateTrie, err := s.omni.GetReadOnlyStateTrie(req.BlockID)
	if err != nil {
		return reply, err
	}

	_, err = stateTrie.GetProof([]byte(req.QueryID))
	if err != nil {
		return nil, err
	}

	//TODO: add more checks
	reply.OK = true
	reply.QueryInstID = req.QueryInstID

	return reply, nil
}

// HandleSignDeferredTx handles requests to add signature to a deferred query
func (s *Service) HandleSignDeferredTx(req *SignDeferredTxRequest) (*SignDeferredTxReply, error) {
	reply := &SignDeferredTxReply{}
	reply.OK = false
	// sanitize params
	if err := emptyInstID(req.QueryInstID); err != nil {
		return reply, xerrors.Errorf("%+v", err)
	}

	// if this server is the one receiving the request from the client
	log.Lvl1("[INFO]: ", s.ServerIdentity().String(), " received a SignDeferredTxRequest for query:", req.QueryInstID)

	//TODO: 1. add more checks to see if enough number of signatures are received. If that is the case, exec the query here
	// 2. retrieve status here from skipchain to get more reliable results
	reply.OK = true
	reply.QueryInstID = req.QueryInstID
	reply.QueryStatus = req.QueryStatus

	return reply, nil
}

func emptyInstID(id byzcoin.InstanceID) error {
	if id.String() == "" {
		return fmt.Errorf("instance id is empty")
	}
	return nil
}

func checkStatus(status string) error {
	if len(status) == 0 {
		return fmt.Errorf("empty query status")
	}
	if status != "Submitted" {
		return fmt.Errorf("wrong query status")
	}
	return nil
}

// HandlePropagateID propagates the instance ID of the query
func (s *Service) HandlePropagateID(req *PropagateIDRequest) (*PropagateIDReply, error) {
	log.Lvl1("[INFO]: ", s.ServerIdentity().String(), "received a PropagateIDRequest for query", req.QueryInstID)

	_, err := s.PropagateID(req.Roster, req, 10*time.Minute)
	if err != nil {
		log.Lvl1("could not propagate data", err)
	}
	return &PropagateIDReply{true}, nil
}

// HandleGetSharedData retrieves all the instace IDs saved in the node
func (s *Service) HandleGetSharedData(req *GetSharedDataRequest) (*GetSharedDataReply, error) {
	log.Lvl1("[INFO]: ", s.ServerIdentity().String(), "received a GetSharedDataRequest")

	err := s.tryLoad()
	if err != nil {
		return nil, xerrors.Errorf("could not load the storage to get shared data: %v", err)
	}
	return &GetSharedDataReply{s.Storage.SharedInstanceIDs}, nil
}

func getQueryByID(view byzcoin.ReadOnlyStateTrie, eid []byte) (*Query, error) {
	v0, _, _, _, err := view.GetValues(eid)
	if err != nil {
		return nil, err
	}
	var q Query
	if err := protobuf.Decode(v0, &q); err != nil {
		return nil, err
	}
	return &q, nil
}

// Saves s.Storage in the node
func (s *Service) save() {
	s.Storage.storageMutex.Lock()
	defer s.Storage.storageMutex.Unlock()
	log.Lvl1("[INFO] Saving service to save the data")
	err := s.Save(storageKey, s.Storage)
	if err != nil {
		log.Error("couldn't save the data:", err)
	}
	s.SaveVersion(dbVersion)
}

// Tries to load the configuration and updates the data in the service
// if it finds a valid config-file.
func (s *Service) tryLoad() error {
	msg, err := s.Load(storageKey)
	if err != nil {
		return err
	}
	if msg == nil {
		return nil
	}
	var ok bool
	s.Storage, ok = msg.(*Storage)
	if !ok {
		return errors.New("data is of wrong type")
	}

	return nil
}
