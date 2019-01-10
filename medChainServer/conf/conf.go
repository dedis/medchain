package conf

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
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
	Action string `json:"Action"`
	Users  []int  `json:"Users,omitempty"`
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
	dir, err := filepath.Abs(filepath.Dir(os.Args[0]))
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(dir)

	jsonFile, err := os.Open(confFileName)
	// if we os.Open returns an error then handle it
	if err != nil {
		return nil, err
	}
	fmt.Printf("Successfully Opened %s\n", confFileName)
	// defer the closing of our jsonFile so that we can parse it later on
	defer jsonFile.Close()

	byteValue, _ := ioutil.ReadAll(jsonFile)

	// jsonString := `{
	// 	  "KeyDirectory":"../keys/",
	// 	  "Hospitals": [
	// 	    {
	// 	      "Name":"Hospital1",
	// 	      "SuperAdmin": {
	// 	        "PublicKey":"super_admins/0_public_ed25519:fc2ea16063dcefddb21795b593bf68f58a39add33eaaf25f6ad99b78644e1351",
	// 	        "PrivateKey":"super_admins/0_private_ed25519:fc2ea16063dcefddb21795b593bf68f58a39add33eaaf25f6ad99b78644e1351",
	// 	        "Name":"Alice"
	// 	      },
	// 	      "Admins":[
	// 	        {
	// 	          "PublicKey":"admins/0_public_ed25519:8378f0dcec5f594e0274e991ce811d6b22ec489221b6781fc61e6c56bf2c0495",
	// 	          "Name":"Charles"
	// 	        },
	// 	        {
	// 	          "PublicKey":"admins/1_public_ed25519:341a301dffa5d308c2ad7c1807d2d7395ce5e23237cf01a379b3f8260f797b8e",
	// 	          "Name":"David"
	// 	        }
	// 	      ],
	// 	      "Managers":[
	// 	        {
	// 	          "PublicKey":"managers/0_public_ed25519:bbf4be834e40bc2c3220884ad0dd5d7c62b9e731a1b731643073d1d2d092e877",
	// 	          "Name":"Henry"
	// 	        },
	// 	        {
	// 	          "PublicKey":"managers/1_public_ed25519:33ba4118f9f42625d34c253383c6a4c3d5c1f3d3873a5a43382694c8768ab02f",
	// 	          "Name":"Irene"
	// 	        }
	// 	      ],
	// 	      "Users":[
	// 	        {
	// 	          "PublicKey":"users/0_public_ed25519:cecceeee2d0572f785590abe5d1afcc77eb5bc10c94c8441e2e2eccfc94749fe",
	// 	          "Name":"Lola"
	// 	        },
	// 	        {
	// 	          "PublicKey":"users/1_public_ed25519:f8862b83144ec3af0cf2ce404bec72cf429cd16c0eda8ac3184e5c49c87aa1eb",
	// 	          "Name":"Marin"
	// 	        }
	// 	      ]
	// 	    },
	// 	    {
	// 	      "Name":"Hospital2",
	// 	      "SuperAdmin": {
	// 	        "PublicKey":"super_admins/1_public_ed25519:71f39da8251eabc0ec5d8edc38facae73bec33f650a20639dee30b59ac860975",
	// 	        "PrivateKey":"super_admins/1_private_ed25519:71f39da8251eabc0ec5d8edc38facae73bec33f650a20639dee30b59ac860975",
	// 	        "Name":"Bob"
	// 	      },
	// 	      "Admins":[
	// 	        {
	// 	          "PublicKey":"admins/2_public_ed25519:120c4eff487ee4bba222a4db9e0514898dee52d73a2ec53a66f5258a35a67d75",
	// 	          "Name":"Edward"
	// 	        },
	// 	        {
	// 	          "PublicKey":"admins/3_public_ed25519:c3fa074c11f51654d9456d7b7e1783763757b00f1f01758bdb3a6117218edf2b",
	// 	          "Name":"Fanny"
	// 	        }
	// 	      ],
	// 	      "Managers":[
	// 	        {
	// 	          "PublicKey":"managers/2_public_ed25519:ab2e9fe6075fe626fc46a84016b91978b970bdafd35e3ec49921eba4ebf20985",
	// 	          "Name":"Jack"
	// 	        }
	// 	      ],
	// 	      "Users":[
	// 	        {"PublicKey":"users/2_public_ed25519:38b82ce9b75d5d9b98b211d1d9e758d79a94a667aee2ec19253feb7892397649", "Name":"Noah"}
	// 	      ]
	// 	    }
	// 	  ]
	// 	  ,
	// 	  "Projects":[
	// 	    {
	// 	      "Name":"ProjectX",
	// 	      "ManagerOwners":[{"i":0, "j":0}, {"i":1, "j":0}],
	// 	      "SigningUsers":[{"i":0, "j":0}, {"i":1, "j":0}],
	// 	      "Rules": [
	// 	        {
	// 	          "Action":"ObfuscatedQuery",
	// 	          "Users":[0,1]
	// 	        },
	// 	        {
	// 	          "Action":"AggregatedQuery",
	// 	          "Users":[0]
	// 	        }
	// 	      ]
	// 	    }
	// 	  ]
	// 	}
	// `
	// byteValue := []byte(jsonString)
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
