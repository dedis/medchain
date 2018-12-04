package main

import (
	"flag"
)

func getFlags() (string, string) {
	var Port string
	var db_file string
	flag.StringVar(&Port, "port", "8383", "Port for the server")
	flag.StringVar(&db_file, "db", "./transaction.db", "Path to the sql database file")
	flag.Parse()
	return Port, db_file
}
