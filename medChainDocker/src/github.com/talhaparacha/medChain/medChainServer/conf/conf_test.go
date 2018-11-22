package conf

import (
	"fmt"
	"testing"
)

func TestReadConf(t *testing.T) {
	configuration, err := ReadConf("conf.json")
	if err != nil {
		t.FailNow()
	}
	fmt.Println("dir", configuration.KeyDirectory)
	for _, hospital := range configuration.Hospitals {
		fmt.Println(hospital.Name)
		fmt.Println(hospital.PublicKey)
		for _, admin := range hospital.Admins {
			fmt.Println(admin.PublicKey)
			for _, manager := range admin.Managers {
				fmt.Println(manager.PublicKey)
				for _, user := range manager.Users {
					fmt.Println(user.PublicKey)
				}
			}
		}
	}

	for _, project := range configuration.Projects {
		fmt.Println(project.Name)
		fmt.Println(project.ManagerOwners)
		fmt.Println(project.SigningUsers)
	}
}

func TestBadFile(t *testing.T) {
	_, err := ReadConf("badfile")
	if err == nil {
		t.FailNow()
	}
}
