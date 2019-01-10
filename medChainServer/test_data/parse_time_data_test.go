package timedata_test

import (
	"testing"

	"github.com/DPPH/MedChain/medChainServer/test_data"
)

const filenameRead = "medchain.csv"
const filenameWrite = "result.txt"
const filenameToml = "../medchain.toml"

var flags = []string{"bf", "depth", "rounds", "runwait", "servers", "\n", "BootstrapProcess"}

func TestReadTomlSetup(t *testing.T) {
	timedata.ReadTomlSetup(filenameToml, 0)
}

func TestReadDataToCSVFile(t *testing.T) {
	timedata.ReadDataFromCSVFile(filenameRead, flags)
}

func TestWriteDataFromCSVFile(t *testing.T) {
	testTimeData := timedata.ReadDataFromCSVFile(filenameRead, flags)

	timedata.CreateCSVFile(filenameWrite)
	for i := 0; i < len(testTimeData[flags[0]]); i++ {
		setup := timedata.ReadTomlSetup(filenameToml, i)
		timedata.WriteDataFromCSVFile(filenameWrite, flags, testTimeData, i, setup)
	}
}
