package main

import (
	"fmt"
	"github.com/gavv/monotime"
	"google.golang.org/grpc"
	"sync"
	pb "tensorflow_serving/apis"
	"time"
)

type BenchmarkRequest struct {
	Server         string             `json:"server"`
	Runs           int                `json:"runs"`
	Clients        int                `json:"clients"`
	Concurrency    int                `json:"concurrency"`
	PredictRequest *pb.PredictRequest `json:"predict_request"`
	MaxTimeout     float32            `json:"max_timeout"`
}

type BenchmarkResponse struct {
	StartedTime  time.Time     `json:"started_time"`
	EndedTime    time.Time     `json:"ended_time"`
	Duration     time.Duration `json:"duration"`
	ErrorMessage string
	Success      bool `json:"success"`
	WorkerIndex  int
	RequestIndex int
}

type BenchmarkSummary struct {
	StartedTime time.Time `json:"started_time"`
	EndedTime   time.Time `json:"ended_time"`
	Request     *BenchmarkRequest
	Responses   []*BenchmarkResponse
	Runs        int
	Errors      int
	Done        chan bool
	Progress    chan float32
}

func (bmSummary *BenchmarkSummary) CalculateRPS() (rps float64) {
	duration := time.Now().UTC().Sub(bmSummary.StartedTime)

	return float64(bmSummary.Runs) / duration.Seconds()
}

func NewBenchmarkRequest(server string, predictRequest *pb.PredictRequest) (bmRequest *BenchmarkRequest) {
	bmRequest = &BenchmarkRequest{
		Server:         server,
		PredictRequest: predictRequest,
	}

	return
}

func (bmRequest *BenchmarkRequest) Run() (summary *BenchmarkSummary) {
	summary = &BenchmarkSummary{
		StartedTime: time.Now().UTC(),
		Done:        make(chan bool),
		Progress:    make(chan float32),
		Responses:   make([]*BenchmarkResponse, bmRequest.Runs),
	}
	responses := make(chan *BenchmarkResponse)

	totalRuns := bmRequest.Runs
	runsPerClient := totalRuns / bmRequest.Clients
	extra := totalRuns % bmRequest.Clients

	// spin up workers and assign runs per workers
	for i := 0; i < bmRequest.Clients; i++ {
		var runs int

		if i < extra {
			runs = runsPerClient + 1
		} else {
			runs = runsPerClient
		}

		go func(index int, runs int) {
			bmRequest.runClient(index, runs, responses)
		}(i, runs)
	}

	// drain the channel
	go func() {
		ticks := bmRequest.Runs / 100
		if ticks > 100 {
			ticks = 100
		} else if ticks <= 0 {
			ticks = 1
		}

		progress := 0
		for i := 0; i < bmRequest.Runs; i++ {
			r := <-responses
			summary.Responses[i] = r
			summary.Runs = i + 1

			if !r.Success {
				summary.Errors += 1
			}

			if (i+1)%ticks == 0 {
				progress = 100 * (i + 1) / bmRequest.Runs
				summary.Progress <- float32(progress)
			}
		}

		if progress < 100 {
			summary.Progress <- 100.0
		}

		summary.EndedTime = time.Now().UTC()
		summary.Done <- true
		close(summary.Done)
		close(summary.Progress)
	}()

	return
}

func (bmRequest *BenchmarkRequest) runClient(workerIndex int, runs int, responses chan *BenchmarkResponse) {
	// connect to server
	conn, err := grpc.Dial(bmRequest.Server, grpc.WithInsecure())

	// TODO: better error handling
	if err != nil {
		panic(err)
	}

	defer conn.Close()

	wg := sync.WaitGroup{}
	wg.Add(runs)

	queues := make(chan int, bmRequest.Concurrency)

	for i := 0; i < runs; i++ {
		queues <- i
		go func() {
			defer wg.Done()

			response := &BenchmarkResponse{
				StartedTime: time.Now().UTC(),
			}

			now := monotime.Now()
			_, err := SendRequestToClient(conn, bmRequest.PredictRequest)
			response.Duration = monotime.Now() - now
			response.EndedTime = time.Now().UTC()

			response.RequestIndex = <-queues

			response.Success = err == nil
			if !response.Success {
				response.ErrorMessage = fmt.Sprintf("%v", err)
				fmt.Printf("Error: %v\n", response.ErrorMessage)
			}

			response.WorkerIndex = workerIndex

			responses <- response
		}()
	}

	wg.Wait()
}
