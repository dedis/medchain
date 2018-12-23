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
		fmt.Println("Super Admin", hospital.SuperAdmin.Name, hospital.SuperAdmin.PublicKey, hospital.SuperAdmin.PrivateKey)
		for _, admin := range hospital.Admins {
			fmt.Println("Admin", admin.Name, admin.PublicKey)
		}
		for _, manager := range hospital.Managers {
			fmt.Println("Manager", manager.Name, manager.PublicKey)
		}
		for _, user := range hospital.Users {
			fmt.Println("User", user.Name, user.PublicKey)
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
