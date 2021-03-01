package service

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/medchain/admin/protocol"
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
}

// Service ...
type Service struct {
	*onet.ServiceProcessor
	ShareData protocol.PropagationFunc // The protocol to broadcast messages to all conodes
	storage   *storage
}

var storageID = []byte("main")

// storage hold the data locally
type storage struct {
	IDs []byzcoin.InstanceID
	sync.Mutex
}

// DeferredID handles the reception of a DeferredID message
func (s *Service) DeferredID(req *ShareDeferredID) (*ShareDeferredIDReply, error) {
	_, err := s.ShareData(req.Roster, req, 10*time.Minute) // broadcast the message to other conodes
	if err != nil {
		log.Lvl1(err)
	}
	return &ShareDeferredIDReply{true}, nil
}

// HandleGetDeferredIDs handles the reception of a GetDeferredIDs request. It
// return the list of all deferred transactions stored
func (s *Service) HandleGetDeferredIDs(req *GetDeferredIDs) (*GetDeferredIDsReply, error) {
	err := s.tryLoad()
	if err != nil {
		return nil, xerrors.Errorf("failed to load storage: %w", err)
	}
	return &GetDeferredIDsReply{s.storage.IDs}, nil
}

// saves all data from local memory to the local db of the conode
func (s *Service) save() {
	s.storage.Lock()
	defer s.storage.Unlock()
	err := s.Save(storageID, s.storage)
	if err != nil {
		log.Error("Couldn't save data:", err)
	}
}

// Tries to load the storage from the local db of the conode into memory
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
// be stored in memory for tests and simulations, and on disk for real
// deployments.
func newService(c *onet.Context) (onet.Service, error) {
	s := &Service{
		ServiceProcessor: onet.NewServiceProcessor(c),
	}
	var err error
	// ShareData is the method that broadcast messages to all conodes
	s.ShareData, err = protocol.NewPropagationFuncTest(s, "ShareData", -1, func(m network.Message) error {
		// What is done when a node receive a DeferredID message from a conode that broadcast it
		s.storage.Lock()
		s.storage.IDs = append(s.storage.IDs, m.(*ShareDeferredID).ID) // store the deferred instance id in the db of the conode
		s.storage.Unlock()
		s.save()
		return nil
	},
		func() network.Message {
			return &ShareDeferredIDReply{true} // What is sent back to the sender of the DeferredID message
		})

	if err != nil {
		return nil, fmt.Errorf("couldn't create propagation function: %+v", err)
	}
	if err = s.RegisterHandlers(s.DeferredID, s.HandleGetDeferredIDs); err != nil {
		return nil, errors.New("Couldn't register messages")
	}
	if err = s.tryLoad(); err != nil {
		log.Error(err)
		return nil, err
	}

	return s, nil
}