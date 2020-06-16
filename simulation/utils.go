package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"math/rand"
	"os"
	"strings"
	"time"
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

func stringWithCharset(length int, charset string) string {
	var seededRand *rand.Rand = rand.New(
		rand.NewSource(time.Now().UnixNano()))
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

func randomString(length int, charset string) string {
	if charset == "" {
		charset = "abcdefghijklmnopqrstuvwxyz" +
			"ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	}
	return stringWithCharset(length, charset)
}

func randomAction() string {
	var actionsList = "patient_list,count_per_site,count_per_site_obfuscated,count_per_site_shuffled,count_per_site_shuffled_obfuscated,count_global,count_global_obfuscated"

	actions := strings.Split(actionsList, ",")
	n := rand.Intn(len(actions))
	return actions[n]
}
