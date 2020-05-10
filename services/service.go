package medchain

import (
	"errors"
	"fmt"
	"time"

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
	omni *byzcoin.Service
}

// NewService receives the context that holds information about the node it's
// running on. Saving and loading can be done using the context. The data will
// be stored in memory for tests and simulations, and on disk for real
// deployments.
func NewService(c *onet.Context) (onet.Service, error) {
	s := &Service{
		ServiceProcessor: onet.NewServiceProcessor(c),
		omni:             c.Service(byzcoin.ServiceName).(*byzcoin.Service),
	}

	if err := s.RegisterHandlers(s.HandleSpawnDeferredQuery); err != nil {
		return nil, errors.New("Couldn't register messages")
	}

	return s, nil
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

// HandleSpawnDeferredQuery handles requests to submit (= spawn or add) a query
func (s *Service) HandleSpawnDeferredQuery(req *AddDeferredQueryRequest) (*AddDeferredQueryReply, error) {
	// sanitize params
	if err := emptyInstID(req.QueryInstID); err != nil {
		return nil, xerrors.Errorf("%+v", err)
	}
	if err := checkStatus(req.QueryStatus); err != nil {
		return nil, xerrors.Errorf("%+v", err)
	}

	// if this server is the one receiving the request from the client
	log.Lvl1("[INFO]: ", s.ServerIdentity().String(), " received an AddDeferredQueryRequest for query:", req.QueryID)

	stateTrie, err := s.omni.GetReadOnlyStateTrie(req.BlockID)
	if err != nil {
		return nil, err
	}

	_, err = stateTrie.GetProof([]byte(req.QueryID))
	if err != nil {
		return nil, err
	}

	//TODO: add more checks

	reply := &AddDeferredQueryReply{}
	reply.OK = true

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
