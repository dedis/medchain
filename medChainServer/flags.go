package main

import (
	"flag"
)

/**
Read the flags values at start-up
**/
func getFlags() (string, string, string) {
	var Port string
	var Conf string
	var SigningUrl string
	flag.StringVar(&Port, "port", "8989", "Port for the server")
	flag.StringVar(&Conf, "conf", "conf/conf.json", "The json configuration file used for the bootstrap")
	flag.StringVar(&SigningUrl, "signing_service", "http://localhost:8383", "The url of the centralized signing service")
	flag.Parse()
	return Port, Conf, SigningUrl
}
