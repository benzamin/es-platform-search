package main

import (
	"bytes"
	"crypto/tls"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"
)

const fileName = "search-db.csv"
const index_name = "index_my_poc1"
const elastic_url = "https://0.0.0.0:9200/_bulk"
const elastic_username = "elastic"
const elastic_password = "F5ar6QpY8mjW+yrwuk22"

func main() {
	// Open the CSV file
	csvFile, err := os.Open(fileName)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer csvFile.Close()

	// Read in the CSV data
	reader := csv.NewReader(csvFile)
	var headers []string
	var jsonData []map[string]interface{}
	for {
		record, err := reader.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			fmt.Println(err)
			return
		}
		if headers == nil {
			headers = record
		} else {
			data := make(map[string]interface{})
			for i, value := range record {
				//fmt.Printf("---------------------------------Got value %s in %s \n", value, headers[i])
				if strings.HasPrefix(value, "{") || strings.HasPrefix(value, "[") { //to manage filters
					var objmap map[string]interface{}
					if err := json.Unmarshal([]byte(value), &objmap); err != nil {
						log.Fatal(err)
					}
					data[headers[i]] = objmap
				} else {
					data[headers[i]] = value
				}
			}
			jsonData = append(jsonData, data)
		}
	}

	// Format the JSON data for bulk upload to ElasticSearch
	var buffer bytes.Buffer
	for i, data := range jsonData {
		metaData := map[string]interface{}{
			"index": map[string]interface{}{
				"_index": index_name,
			},
		}
		metaDataBytes, _ := json.Marshal(metaData)
		buffer.Write(metaDataBytes)
		buffer.WriteString("\n")

		dataBytes, _ := json.Marshal(data)
		buffer.Write(dataBytes)
		buffer.WriteString("\n")
		fmt.Printf(" ______Document %v______: %s \n ", i, metaDataBytes)
		fmt.Printf(string(dataBytes) + "\n")
	}

	// Upload the formatted JSON data to ElasticSearch
	fmt.Printf("\n\n ......................Now posting to Elasticsearch.............\n\n")
	url := elastic_url
	req, err := http.NewRequest("POST", url, &buffer)
	if err != nil {
		fmt.Println(err)
		return
	}

	// Set the appropriate headers for the request
	req.Header.Set("Content-Type", "application/json")
	req.SetBasicAuth(elastic_username, elastic_password)

	// Send the request to ElasticSearch
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{Transport: transport, Timeout: time.Second * 60}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(string(body))
}
