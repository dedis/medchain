package main

import (
	"flag"
)

func getFlags() string {
	var Port string
	flag.StringVar(&Port, "port", "6161", "Port for the server")
	flag.Parse()
	return Port
}
