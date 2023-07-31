package main

import (
	"encoding/json"
	_ "expvar"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"time"
)

var config Config

func main() {
	var err error
	config, err = readConfig()
	if err != nil {
		log.Fatalf("Aborting! Could not read config.json file: %v", err)
	}

	var (
		maxWorkers   = flag.Int("max_workers", config.Max_workers, "The number of workers to start")
		maxQueueSize = flag.Int("max_queue_size", config.Queue_size, "The size of job queue")
		port         = flag.String("port", config.Port, "The server port")
	)

	flag.Parse()

	fmt.Printf("Starting Worker-Pool using: \n> max_workers: %d \n> max_queue_size: %d \n> port: %s \n\n", *maxWorkers, *maxQueueSize, *port)

	//Initiate Log
	logFile, err := os.OpenFile(config.Log_write_location+"search.log", os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		log.Fatalf("error opening file: %v", err)
	}
	defer logFile.Close()
	log.SetOutput(logFile)

	// Create the job queue.
	jobQueue := make(chan Job, *maxQueueSize)

	// Start the dispatcher.
	dispatcher := NewDispatcher(jobQueue, *maxWorkers)
	dispatcher.run()

	// Start the HTTP handler.
	http.HandleFunc("/search", func(w http.ResponseWriter, r *http.Request) {
		searchHandlerGET(w, r, jobQueue)
	})
	//http server parameters
	server := &http.Server{
		Addr:           ":" + *port,
		ReadTimeout:    time.Duration(config.Read_timeout) * time.Millisecond,
		WriteTimeout:   time.Duration(config.Write_timeout) * time.Millisecond,
		IdleTimeout:    120 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	//start the server
	log.Fatal(server.ListenAndServe())

}

func searchHandlerGET(w http.ResponseWriter, r *http.Request, jobQueue chan Job) {
	// Make sure we can only be called with an HTTP GET request.
	if r.Method != "GET" {
		sendFailResponse(w, http.StatusMethodNotAllowed, "You must use GET method")
		fmt.Println("You must use GET method")
		return
	}

	searchTerm := r.URL.Query().Get("q")
	if searchTerm == "" {
		sendFailResponse(w, http.StatusBadRequest, "You must specify a search term, ex: /search?q=internet")
		fmt.Println("search term not provided")
		return
	}
	log.Printf("Request>" + "API:" + r.URL.Path + "|ip:" + r.RemoteAddr + ",|query:" + r.URL.RawQuery)

	typeHead := r.URL.Query().Get("typehead")       // should send mini response
	limit := r.URL.Query().Get("limit")             //Total results to send
	offset := r.URL.Query().Get("offset")           //from which offset result to start
	fuzzy := r.URL.Query().Get("fuzzy")             // should consider fuzzy query, e.x: 'dolar/doller/dolur' will match for 'dollar'
	category := r.URL.Query().Get("category")       //which category to search
	subcategory := r.URL.Query().Get("subcategory") //which subcategory to search

	// Create Job and push the work onto the jobQueue.
	task := NewESSearchTask()
	parameters := map[string]string{"q": searchTerm, "typehead": typeHead, "offset": offset, "limit": limit, "fuzzy": fuzzy, "category": category, "subcategory": subcategory}
	job := NewJob(&task, parameters, NewJobResultChannel())
	jobQueue <- job

	//wati to receuve the response from return channel
	resp := <-job.ReturnChannel
	if resp.Error != nil {
		sendFailResponse(w, http.StatusInternalServerError, fmt.Sprintf("Something went wrong. %s", resp.Error))
		log.Printf("Response>" + "status:fail|error:" + fmt.Sprintf("%s", resp.Error) + "|api:" + r.URL.Path + "|ip:" + r.RemoteAddr + "|query:" + r.URL.RawQuery)
		return
	}
	sendSuccessResponse(w, resp.Value)
	log.Printf("Response>" + "status:success|error:nil" + "|api:" + r.URL.Path + "|ip:" + r.RemoteAddr + "|query:" + r.URL.RawQuery)
}

//#### Config Part ######

const configFileName = "config.json"

type Config struct {
	Max_workers        int    `json:"max_workers"`
	Queue_size         int    `json:"queue_size"`
	Port               string `json:"port"`
	ES_url             string `json:"es_url"`
	ES_username        string `json:"es_username"`
	ES_password        string `json:"es_password"`
	Es_index           string `json:"es_index"`
	Per_page           int    `json:"per_page"`
	ES_api_timeout     int    `json:"es_api_timeout"`
	Log_write_location string `json:"log_write_location"`
	Read_timeout       int    `json:"read_timeout"`
	Write_timeout      int    `json:"write_timeout"`
}

func readConfig() (Config, error) {
	//_, filepath, _, _ := runtime.Caller(0) //gets the current main.go file location in OS
	//pwd := filepath[:len(filepath)-7]      //main.go name has 7 characters
	txt, err := ioutil.ReadFile("./" + configFileName)
	if err != nil {
		return Config{}, err
	}
	var configJson Config
	if err := json.Unmarshal(txt, &configJson); err != nil {
		return Config{}, err
	}
	return configJson, nil
}

// ###### Response part #######
type Response struct {
	Status  bool        `json:"status"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

func sendSuccessResponse(w http.ResponseWriter, response interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(&Response{true, "Success", response})
}

func sendFailResponse(w http.ResponseWriter, httpStatus int, errorMessage string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(httpStatus)
	json.NewEncoder(w).Encode(&Response{false, errorMessage, nil})
}
