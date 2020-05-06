package utilserver

import (
	"os"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
	onetLog "go.dedis.ch/onet/v3/log"
)

// MedChainLogLevel is the log level, assuming the same convention as the cothority / medchain log levels:
// TRACE(5), DEBUG(4), INFO(3), WARNING(2), ERROR(1), FATAL(0)
var MedChainLogLevel int

// MedChainNodesAddress is the slice of the URL of all the MedChain nodes, with the order matching the position in the slice
var MedChainNodesAddress []string

// MedChainNodeIdx is the index of this node in the list of nodes
var MedChainNodeIdx int

// MedChainGroupFilePath is the path of the MedChain group file from which is derived the cothority public key
var MedChainGroupFilePath string

// MedChainTimeoutSeconds is the MedChain communication timeout (seconds)
var MedChainTimeoutSeconds int

func init() {
	SetLogLevel(os.Getenv("MEDCHAIN_LOG_LEVEL"))

	MedChainNodesAddress = strings.Split(os.Getenv("MEDCHAIN_NODES_ADDRESS"), ",")

	idx, err := strconv.ParseInt(os.Getenv("MEDCHAIN_NODE_IDX"), 10, 64)
	if err != nil || idx < 0 {
		logrus.Warn("invalid MedChainNodeIdx")
		idx = 0
	}
	MedChainNodeIdx = int(idx)

	MedChainGroupFilePath = os.Getenv("MEDCHAIN_GROUP_FILE_PATH")

	medchainTimeout, err := strconv.ParseInt(os.Getenv("MEDCHAIN_TIMEOUT_SECONDS"), 10, 64)
	if err != nil || medchainTimeout < 0 {
		logrus.Warn("invalid MedChainTimeoutSeconds, defaulted")
		medchainTimeout = 3 * 60 // 3 minutes
	}
	MedChainTimeoutSeconds = int(medchainTimeout)
}

// SetLogLevel initializes the log levels of all loggers
func SetLogLevel(lvl string) {
	// formatting
	logrus.SetFormatter(&logrus.TextFormatter{
		ForceColors: true,
	})

	intLvl, err := strconv.ParseInt(lvl, 10, 64)
	if err != nil || intLvl < 0 || intLvl > 5 {
		logrus.Warn("invalid MedChainLogLevel, defaulted")
		intLvl = 3
	}
	MedChainLogLevel = int(intLvl)
	logrus.SetLevel(logrus.Level(MedChainLogLevel + 1))
	onetLog.SetDebugVisible(MedChainLogLevel)
}
