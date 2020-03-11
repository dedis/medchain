package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
)

func readCSV() {
	// Open the file
	csvfile, err := os.Open("./test_data/service.csv")
	if err != nil {
		log.Fatalln("couldn't open the csv file", err)
	}

	// Parse the file
	r := csv.NewReader(csvfile)

	// Iterate through the records
	for {
		// Read each record from csv
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatal(err)
		}
		fmt.Printf("%s : %s\n", record[0], record[1])
	}
}
