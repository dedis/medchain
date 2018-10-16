package conf

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
)

type User struct {
	PublicKey    string `json:"PublicKey"`
	PrivateKey   string `json:"PrivateKey"`
	ManagerIndex int    `json:"ManagerIndex"`
}

type Manager struct {
	PublicKey  string `json:"PublicKey"`
	PrivateKey string `json:"PublicKey"`
	AdminIndex int    `json:"AdminIndex"`
}

type Admin struct {
	PublicKey  string `json:"PublicKey"`
	PrivateKey string `json:"PrivateKey"`
}

type Rule struct {
	Action   string `json:"Action"`
	ExprType string `json:"ExprType"`
	Users    []int  `json:"Users,omitempty"`
}

type Project struct {
	Name          string `json:"Name"`
	ManagerOwners []int  `json:"ManagerOwners"`
	SigningUsers  []int  `json:"SigningUsers"`
	Rules         []Rule `json:"Rules"`
}

type Configuration struct {
	KeyDirectory string    `json:"KeyDirectory"`
	Admins       []Admin   `json:"Admins"`
	Managers     []Manager `json:"Managers"`
	Users        []User    `json:"Users"`
	Projects     []Project `json:"Projects"`
}

func ReadConf(confFileName string) (*Configuration, error) {
	// Open our jsonFile
	jsonFile, err := os.Open(confFileName)
	// if we os.Open returns an error then handle it
	if err != nil {
		return nil, err
	}
	fmt.Printf("Successfully Opened %s\n", confFileName)
	// defer the closing of our jsonFile so that we can parse it later on
	defer jsonFile.Close()

	byteValue, _ := ioutil.ReadAll(jsonFile)

	// we initialize our Configuration
	var configuration Configuration

	// we unmarshal our byteArray which contains our
	// jsonFile's content into 'configuration' which we defined above
	json.Unmarshal(byteValue, &configuration)

	return &configuration, nil
}
