package main

import (
	"flag"
)

func getFlags() (string, bool) {
	var Port string
	var TestConf bool
	flag.StringVar(&Port, "port", "8989", "Port for the server")
	flag.BoolVar(&TestConf, "test", false, "Bootstraps the medchain server with the test configuration")
	flag.Parse()
	return Port, TestConf
}
