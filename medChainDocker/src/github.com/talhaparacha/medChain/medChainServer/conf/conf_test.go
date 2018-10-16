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
	for _, user := range configuration.Users {
		fmt.Println(user.PublicKey)
	}
	for _, admin := range configuration.Admins {
		fmt.Println(admin.PublicKey)
	}
}

func TestBadFile(t *testing.T) {
	_, err := ReadConf("badfile")
	if err == nil {
		t.FailNow()
	}
}
