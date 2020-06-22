package medchainservices

import (
	"go.dedis.ch/kyber/v3/suites"
)

// TSuite in this case is the ed25519 curve
var TSuite = suites.MustFind("Ed25519")
