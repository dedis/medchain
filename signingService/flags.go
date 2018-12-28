package main

import (
	"flag"
)

var default_db_file = "./data/actions.db"

func getFlags() (string, string) {
	var Port string
	var db_file string
	flag.StringVar(&Port, "port", "8383", "Port for the server")
	flag.StringVar(&db_file, "db", default_db_file, "Path to the sql database file")
	flag.Parse()
	return Port, db_file
}
