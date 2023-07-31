package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"
)

type ESSearchTask struct{}

func NewESSearchTask() ESSearchTask {
	return ESSearchTask{}
}

func (e *ESSearchTask) Run(payload Payload) JobResult {

	// Create HTTP client & Send the request to ElasticSearch
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		//Proxy: http.ProxyFromEnvironment,
		MaxIdleConns:          20,
		MaxConnsPerHost:       50,
		MaxIdleConnsPerHost:   config.Max_workers, //important
		IdleConnTimeout:       10 * time.Second,
		TLSHandshakeTimeout:   5 * time.Second,
		ExpectContinueTimeout: 2 * time.Second,
	}
	client := &http.Client{Transport: transport, Timeout: time.Millisecond * time.Duration(config.ES_api_timeout)}

	//which fields you want as response
	sourceArr := `["identifier", "title", "short_desc", "description", "type", "category", "sub-category", "price", "icon", "url", "deeplink"]`
	var typeHeadMode = false
	if payload.Params["typehead"] == "true" { //give mini response
		typeHeadMode = true
		sourceArr = `["title", "short_desc"]`
	}
	var limit, offset, fuzzy, category, subcategory = strconv.Itoa(config.Per_page), "0", "AUTO", "", ""

	if payload.Params["limit"] != "" {
		limit = payload.Params["limit"]
	}
	if payload.Params["offset"] != "" {
		offset = payload.Params["offset"]
	}
	if payload.Params["category"] != "" {
		category = payload.Params["category"]
	}
	if payload.Params["subcategory"] != "" {
		subcategory = payload.Params["subcategory"]
	}
	if payload.Params["fuzzy"] != "" {
		fuzzyVal, err := strconv.Atoi(payload.Params["fuzzy"])
		if err == nil && fuzzyVal >= 0 && fuzzyVal <= 2 {
			fuzzy = payload.Params["fuzzy"]
		}
	}
	var is_catalog_priority = false
	var cagerogy_query, subcategory_query = "", ""
	var boost_query = ""
	var fields = ""
	if category == "" && subcategory == "" {
		is_catalog_priority = true
		if is_catalog_priority {
			fields = `"fields": ["query_text"],`
		}
	} else {

		if category != "" {
			cagerogy_query = `{"term" : { "category" : "` + category + `" }}`
		}
		if subcategory != "" {
			subcategory_query = `,{"term" : { "sub-category" : "` + subcategory + `" }}`
		}
	}
	//print(category, subcategory, is_catalog_priority, fuzzy)

	bodyBytes1 := []byte(`{
		"query": {
			"bool": {
				"should": [` + boost_query + `
					{
						"multi_match": {
							"query": "` + payload.Params["q"] + `",
							` + fields + `
							"fuzziness":"` + fuzzy + `"
						}
					}
				],
				"filter": [` + cagerogy_query + subcategory_query + `],
				"minimum_should_match": 1
			}
		},
		"from":` + offset + `,
		"size": ` + limit + `,
		"_source":` + sourceArr + `
		
	}`)
	// Create HTTP request
	es_url := config.ES_url + "/" + config.Es_index + "/_search"
	//req, err := http.NewRequest("GET", es_url, jsonBody)
	req, err := http.NewRequest("GET", es_url, bytes.NewBuffer(bodyBytes1))
	if err != nil {
		return NewJobResult(nil, err)
	}
	req.SetBasicAuth(config.ES_username, config.ES_password)
	req.Header.Add("Content-Type", "application/json")

	// Send HTTP request
	res, err := client.Do(req)
	if err != nil {
		return NewJobResult(nil, err)
	}
	defer res.Body.Close()

	statusOK := res.StatusCode >= 200 && res.StatusCode < 300
	if !statusOK {
		//fmt.Println(res)
		return NewJobResult(nil, fmt.Errorf("%s [error %d]", res.Status, res.StatusCode))
	}

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return NewJobResult(nil, err)
	}

	var esRes ESResult
	err = json.Unmarshal(body, &esRes)
	if err != nil {
		//fmt.Println("Error parsing the response from the first API:", err)
		return NewJobResult(nil, err)
	}
	//fmt.Println(esRes)
	//var searchResults []models.SourceShort = []models.SourceShort{}
	var searchResults []interface{}
	if len(esRes.Hits.Hits) > 0 {

		for _, src := range esRes.Hits.Hits {
			if typeHeadMode {
				shortSource := SourceShort{Title: src.Source.Title, ShortDesc: src.Source.ShortDesc}
				searchResults = append(searchResults, shortSource)
			} else {
				searchResults = append(searchResults, src.Source)
			}
		}
	}
	finalResp := map[string]interface{}{
		"offset":  offset,
		"limit":   limit,
		"results": searchResults,
	}
	return NewJobResult(finalResp, nil)
}
