package main

import (
	"flag"
)

func getFlags() (string, string) {
	var Port string
	var Conf string
	flag.StringVar(&Port, "port", "8989", "Port for the server")
	flag.StringVar(&Conf, "conf", "conf/conf.json", "The json configuration file used for the bootstrap")
	flag.Parse()
	return Port, Conf
}
