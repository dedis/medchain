package conf

import (
	"fmt"
	"testing"
)

func TestReadConf(t *testing.T) {
	configuration, err := ReadConf("test_conf.json")
	if err != nil {
		t.FailNow()
	}
	fmt.Println("dir", configuration.KeyDirectory)
	for _, hospital := range configuration.Hospitals {
		fmt.Println(hospital.Name)
		fmt.Println(hospital.PublicKey)
		for _, admin := range hospital.Admins {
			fmt.Println(admin.PublicKey)
			fmt.Println(admin.Name)
		}
		for _, manager := range hospital.Managers {
			fmt.Println(manager.PublicKey)
			fmt.Println(manager.Name)
		}
		for _, user := range hospital.Users {
			fmt.Println(user.PublicKey)
			fmt.Println(user.Name)
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
