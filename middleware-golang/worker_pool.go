package main

import (
	"fmt"
)

type Job struct {
	Task          Task
	Payload       Payload
	ReturnChannel chan JobResult
}

// Job represents a return type of a job either the result (Value) or Error
type JobResult struct {
	Value interface{}
	Error error
}

// Runs the Job Task
func (j *Job) Run() JobResult {
	return j.Task.Run(j.Payload)
}

func NewJob(task Task, params map[string]string, returnChannel chan JobResult) Job {
	return Job{
		Task:          task,
		Payload:       NewPayload(params),
		ReturnChannel: returnChannel,
	}
}

func NewJobResult(value interface{}, err error) JobResult {
	return JobResult{
		Value: value,
		Error: err,
	}
}

func NewJobResultChannel() chan JobResult {
	return make(chan JobResult)
}

// Data a Job receives as Param
type Payload struct {
	Params map[string]string
}

func NewPayload(params map[string]string) Payload {
	return Payload{
		Params: params,
	}
}

// represents a task handler that receives a payload and returns a generic type response or error
type Task interface {
	Run(payload Payload) JobResult
}

// NewWorker creates takes a numeric id and a channel w/ worker pool.
func NewWorker(id int, workerPool chan chan Job) Worker {
	return Worker{
		id:         id,
		jobQueue:   make(chan Job),
		workerPool: workerPool,
		quitChan:   make(chan bool),
	}
}

type Worker struct {
	id         int
	jobQueue   chan Job
	workerPool chan chan Job
	quitChan   chan bool
}

func (w Worker) start() {
	go func() {
		for {
			// Add my jobQueue to the worker pool.
			w.workerPool <- w.jobQueue

			select {
			case job := <-w.jobQueue:
				// Dispatcher has added a job to my jobQueue.
				fmt.Printf("+++++ worker%d: started %s______\n", w.id, job.Payload.Params)
				result := job.Task.Run(job.Payload)
				if result.Error != nil {
					fmt.Printf("Job has Returned Error XXXXXXXXXXXXX\n")
					job.ReturnChannel <- NewJobResult(nil, result.Error)
				} else {
					fmt.Printf("Job done successfully vvvvvvvvvvvvv\n")
					job.ReturnChannel <- NewJobResult(result.Value, nil)
					fmt.Printf("----- worker%d: completed : %s______\n", w.id, job.Payload.Params)
				}

			case <-w.quitChan:
				// We have been asked to stop.
				fmt.Printf("worker%d stopping\n", w.id)
				return
			}
		}
	}()
}

func (w Worker) stop() {
	go func() {
		w.quitChan <- true
	}()
}

// NewDispatcher creates, and returns a new Dispatcher object.
func NewDispatcher(jobQueue chan Job, maxWorkers int) *Dispatcher {
	workerPool := make(chan chan Job, maxWorkers)

	return &Dispatcher{
		jobQueue:   jobQueue,
		maxWorkers: maxWorkers,
		workerPool: workerPool,
	}
}

type Dispatcher struct {
	workerPool chan chan Job
	maxWorkers int
	jobQueue   chan Job
}

func (d *Dispatcher) run() {
	for i := 0; i < d.maxWorkers; i++ {
		worker := NewWorker(i+1, d.workerPool)
		worker.start()
	}

	go d.dispatch()
}

func (d *Dispatcher) dispatch() {
	for {
		select {
		case job := <-d.jobQueue:
			go func() {
				fmt.Printf("WorkerJobQueue size: %d, fetching for: %s, \n", len(d.workerPool), job.Payload.Params)
				workerJobQueue := <-d.workerPool
				fmt.Printf("Adding %s to workerJobQueue\n", job.Payload.Params)
				workerJobQueue <- job
			}()
		}
	}
}
