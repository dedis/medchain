package utilclient

import (
	"os"
	"strconv"

	"github.com/sirupsen/logrus"
)

// QueryTimeoutSeconds is the timeout for the client query in seconds (default to 3 minutes)
var QueryTimeoutSeconds int64

// MedChainNodeURL is the URL of the MedChain server this client is attached to
var MedChainNodeURL string

func init() {
	var err error

	QueryTimeoutSeconds, err = strconv.ParseInt(os.Getenv("CLIENT_QUERY_TIMEOUT_SECONDS"), 10, 64)
	if err != nil || QueryTimeoutSeconds < 0 {
		logrus.Warn("invalid client query timeout")
		QueryTimeoutSeconds = 3 * 60
	}

	MedChainNodeURL = os.Getenv("MEDCHAIN_NODE_URL")
}
