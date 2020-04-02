package admin

import (
	"go.dedis.ch/cothority/v3/byzcoin"
	"go.dedis.ch/onet/v3/log"
)

func init() {
	err := byzcoin.RegisterGlobalContract(ContractValueID, contractValueFromBytes)
	if err != nil {
		log.ErrFatal(err)
	}

}
