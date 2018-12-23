package conf

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
)

type Signer struct {
	PublicKey  string `json:"PublicKey"`
	PrivateKey string `json:"PrivateKey"`
	Name       string `json:"Name"`
}

type Identity struct {
	PublicKey string `json:"PublicKey"`
	Name      string `json:"Name"`
}

type Hospital struct {
	Name       string     `json:"Name"`
	SuperAdmin Signer     `json:"SuperAdmin"`
	Admins     []Identity `json:"Admins"`
	Managers   []Identity `json:"Managers,omitempty"`
	Users      []Identity `json:"Users,omitempty"`
}

type Coordinates struct {
	I int `json:"i"`
	J int `json:"j"`
}

type Rule struct {
	Action   string        `json:"Action"`
	ExprType string        `json:"ExprType"`
	Users    []Coordinates `json:"Users,omitempty"`
}

type Project struct {
	Name          string        `json:"Name"`
	ManagerOwners []Coordinates `json:"ManagerOwners"`
	SigningUsers  []Coordinates `json:"SigningUsers"`
	Rules         []Rule        `json:"Rules"`
}

type Configuration struct {
	KeyDirectory string     `json:"KeyDirectory"`
	Hospitals    []Hospital `json:"Hospitals"`
	Projects     []Project  `json:"Projects,omitempty"`
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

	// for _, user := range configuration.Users {
	// 	fmt.Println("user", user.PublicKey, user.PrivateKey)
	// }
	//
	// for _, manager := range configuration.Managers {
	// 	fmt.Println("manager", manager.PublicKey, manager.PrivateKey)
	// }
	//
	// for _, admin := range configuration.Admins {
	// 	fmt.Println("admin", admin.PublicKey, admin.PrivateKey)
	// }

	return &configuration, nil
}
