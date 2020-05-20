package admin

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/medchain/protocols"
	"go.dedis.ch/cothority/v3/byzcoin"
	"go.dedis.ch/onet/v3"
	"go.dedis.ch/onet/v3/log"
	"go.dedis.ch/onet/v3/network"
	"golang.org/x/xerrors"
)

// Used for tests
var adminID onet.ServiceID

func init() {
	var err error
	adminID, err = onet.RegisterNewService("ShareID", newService)
	log.ErrFatal(err)
	network.RegisterMessage(&storage{})
	err = byzcoin.RegisterGlobalContract(ContractAccessRightID, contractAccessRightFromBytes)
	if err != nil {
		log.ErrFatal(err)
	}
}

// Service is our template-service
type Service struct {
	// We need to embed the ServiceProcessor, so that incoming messages
	// are correctly handled.
	*onet.ServiceProcessor
	ShareData protocols.PropagationFunc
	storage   *storage
}

// storageID reflects the data we're storing - we could store more
// than one structure.
var storageID = []byte("main")

// storage is used to save our data.
type storage struct {
	IDs []byzcoin.InstanceID
	sync.Mutex
}

// Clock starts a template-protocol and returns the run-time.
func (s *Service) DefferedID(req *DefferedID) (*DefferedIDReply, error) {
	_, err := s.ShareData(req.Roster, req, 10*time.Minute)
	if err != nil {
		log.Lvl1(err)
	}
	return &DefferedIDReply{true}, nil
}

// // Clock starts a template-protocol and returns the run-time.
func (s *Service) HandleGetDefferedIDs(req *GetDeferredIDs) (*GetDeferredIDsReply, error) {
	err := s.tryLoad()
	if err != nil {
		return nil, xerrors.Errorf("Loading the storage : %w", err)
	}
	return &GetDeferredIDsReply{s.storage.IDs}, nil
}

// saves all data.
func (s *Service) save() {
	s.storage.Lock()
	defer s.storage.Unlock()
	err := s.Save(storageID, s.storage)
	if err != nil {
		log.Error("Couldn't save data:", err)
	}
}

// Tries to load the configuration and updates the data in the service
// if it finds a valid config-file.
func (s *Service) tryLoad() error {
	s.storage = &storage{}
	msg, err := s.Load(storageID)
	if err != nil {
		return err
	}
	if msg == nil {
		return nil
	}
	var ok bool
	s.storage, ok = msg.(*storage)
	if !ok {
		return errors.New("Data of wrong type")
	}
	return nil
}

// newService receives the context that holds information about the node it's
// running on. Saving and loading can be done using the context. The data will
// be stored in memory for tests and simulations, and on disk for real deployments.
func newService(c *onet.Context) (onet.Service, error) {
	s := &Service{
		ServiceProcessor: onet.NewServiceProcessor(c),
	}
	var err error
	s.ShareData, err = protocols.NewPropagationFuncTest(s, "ShareData", -1, func(m network.Message) error {
		s.storage.Lock()
		s.storage.IDs = append(s.storage.IDs, m.(*DefferedID).Id)
		s.storage.Unlock()
		s.save()
		return nil
	},
		func() network.Message {
			return &DefferedIDReply{true}
		})

	if err != nil {
		return nil, fmt.Errorf("couldn't create propagation function: %+v", err)
	}
	if err = s.RegisterHandlers(s.DefferedID, s.HandleGetDefferedIDs); err != nil {
		return nil, errors.New("Couldn't register messages")
	}
	if err = s.tryLoad(); err != nil {
		log.Error(err)
		return nil, err
	}

	return s, nil
}
