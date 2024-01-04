package main

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

const csvHeader = "IP Address"
const fieldsCountry = "country"
const url = "http://ip-api.com/batch"

type country struct {
	Country string `json:"country"`
}

type ipToCheck struct {
	Query  string `json:"query"`
	Fields string `json:"fields"`
}

func main() {
	filePath := flag.String("file", "access_logs.csv", "Name of the CSV file with Slack logs")
	flag.Parse()

	records, err := readCSVFile(*filePath)
	if err != nil {
		log.Fatalf("failed to read records from CSV: %s", err.Error())
	}

	uniqIPs := sortUniqIPs(records)
	payload, err := prepareRequestPayload(uniqIPs)
	if err != nil {
		log.Fatalf("failed to prepare request payload: %s", err.Error())
	}

	countries, err := getCountryByIP(payload)
	if err != nil {
		log.Fatalf("failed to get location by IPs list: %s", err.Error())
	}

	fmt.Println(countries)
}

func readCSVFile(filePath string) ([][]string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read provided file: %s", err.Error())
	}
	defer file.Close()

	csvReader := csv.NewReader(file)
	records, err := csvReader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("unable to parse file as CSV for "+filePath, err)
	}

	return records, nil
}

func sortUniqIPs(records [][]string) (IPs []string) {
	unique := make(map[string]struct{})
	for i := range records {
		unique[records[i][3]] = struct{}{}
	}

	delete(unique, csvHeader)

	for key := range unique {
		IPs = append(IPs, key)
	}

	return
}

func prepareRequestPayload(IPs []string) ([]byte, error) {
	list := make([]ipToCheck, len(IPs))
	for i := range IPs {
		list[i].Fields = fieldsCountry
		list[i].Query = IPs[i]
	}

	payload, err := json.Marshal(list)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %w", err)
	}

	return payload, nil
}

func getCountryByIP(payload []byte) ([]string, error) {
	payloadStream := bytes.NewReader(payload)

	client := &http.Client{}
	req, err := http.NewRequest(http.MethodPost, url, payloadStream)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Add("Content-Type", "application/json")

	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var countries []country
	err = json.Unmarshal(body, &countries)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal body: %w", err)
	}

	list := make([]string, len(countries))
	for i := range countries {
		list[i] = countries[i].Country
	}

	return list, nil
}
