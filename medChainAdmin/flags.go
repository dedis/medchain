package main

import (
	"flag"
)

func getFlags() (string, string) {
	var Port string
	var MedchainUrl string
	flag.StringVar(&Port, "port", "6161", "Port for the server")
	flag.StringVar(&MedchainUrl, "medchain_url", "http://localhost:8989", "Url of the medchain server, useful to allow CORS requests of the UI")
	flag.Parse()
	return Port, MedchainUrl
}
