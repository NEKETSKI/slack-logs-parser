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
	"strings"
	"time"
)

const (
	fieldsCountry = "country"
	url           = "http://ip-api.com/batch"
)

type country struct {
	Country string `json:"country"`
}

type ipToCheck struct {
	Query  string `json:"query"`
	Fields string `json:"fields"`
}

func main() {
	filePath := flag.String("file", "access_logs.csv", "Name of the CSV file with Slack logs")
	dateFrom := flag.String("date", "", "Date from which until today you are interested in records (DD-MM-YYYY")
	flag.Parse()

	records, err := readCSVFile(*filePath)
	if err != nil {
		log.Fatalf("failed to read records from CSV: %s", err.Error())
	}

	records, err = filterByDate(records, *dateFrom)
	if err != nil {
		log.Fatalf("failed to sort records by date: %s", err)
	}

	uniqIPs := getUniqIPs(records)
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

func filterByDate(records [][]string, dateFrom string) ([][]string, error) {
	parsedDateFrom, err := time.Parse("02-01-2006", dateFrom)
	if err != nil {
		return nil, fmt.Errorf("failed to parse the date from which records will be filtered: %w", err)
	}

	i := 1
	for ; i < len(records)-1; i++ {
		records[i][0], _, _ = strings.Cut(records[i][0], "GMT")
		t, err := time.Parse("Mon Jan 02 2006 15:04:05", strings.TrimSpace(records[i][0]))
		if err != nil {
			return nil, fmt.Errorf("failed to parse the time: %w", err)
		}

		if t.Before(parsedDateFrom) {
			break
		}
	}

	return records[:i], nil
}

func getUniqIPs(records [][]string) (IPs []string) {
	unique := make(map[string]struct{})
	for i := 1; i < len(records)-1; i++ {
		unique[records[i][3]] = struct{}{}
	}

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
