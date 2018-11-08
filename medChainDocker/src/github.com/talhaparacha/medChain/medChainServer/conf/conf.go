package conf

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
)

type User struct {
	PublicKey  string `json:"PublicKey"`
	PrivateKey string `json:"PrivateKey"`
}

type Manager struct {
	PublicKey  string `json:"PublicKey"`
	PrivateKey string `json:"PrivateKey"`
	Users      []User `json:"Users"`
}

type Admin struct {
	PublicKey  string    `json:"PublicKey"`
	PrivateKey string    `json:"PrivateKey"`
	Managers   []Manager `json:"Managers"`
}

type ManagerCoordinates struct {
	I int `json:"i"`
	J int `json:"j"`
}

type UserCoordinates struct {
	I int `json:"i"`
	J int `json:"j"`
	K int `json:"k"`
}

type Rule struct {
	Action   string            `json:"Action"`
	ExprType string            `json:"ExprType"`
	Users    []UserCoordinates `json:"Users,omitempty"`
}

type Project struct {
	Name          string               `json:"Name"`
	ManagerOwners []ManagerCoordinates `json:"ManagerOwners"`
	SigningUsers  []UserCoordinates    `json:"SigningUsers"`
	Rules         []Rule               `json:"Rules"`
}

type Configuration struct {
	KeyDirectory string    `json:"KeyDirectory"`
	Admins       []Admin   `json:"Admins"`
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
