package main

import (
	"flag"
	"fmt"
	"gopkg.in/cheggaaa/pb.v1"
	"os"
	"runtime"
)

var (
	hostName    string
	modelName   string
	inputFile   string
	concurrency int
	clients     int
	runs        int
)

func init() {
	flag.StringVar(&hostName, "host", "", "Server hostname")
	flag.StringVar(&modelName, "model", "default", "Target model name")
	flag.StringVar(&inputFile, "input", "test.json", "Input payload file")
	flag.IntVar(&clients, "c", 1, "Number of client")
	flag.IntVar(&concurrency, "C", 1, "Number of concurrent requests per client")
	flag.IntVar(&runs, "N", 1000, "Number of runs")
}

func main() {
	runtime.GOMAXPROCS(0)
	flag.Parse()

	if hostName == "" {
		fmt.Printf("  -host is required\n")
		os.Exit(1)
	}

	if runs <= 0 {
		fmt.Printf("  -N must be greater than 0\n")
		os.Exit(1)
	}

	// read json file
	request, fileErr := LoadRequestFromJson(modelName, inputFile)

	if fileErr != nil {
		panic(fmt.Sprintf("Unable to load json file: %v", fileErr))
	}

	bmRequest := NewBenchmarkRequest(hostName, request)
	bmRequest.Runs = runs
	bmRequest.Concurrency = concurrency
	bmRequest.Clients = clients

	// TODO: send a test request first

	summary := bmRequest.Run()

	done := false

	fmt.Print("Benchmark Settings:\n")
	fmt.Printf("- Server: %v\n- Input file: %v\n- Runs : %v\n- Clients: %v\n- Concurrency: %v\n\n",
		hostName, inputFile, bmRequest.Runs, bmRequest.Clients, bmRequest.Concurrency,
	)

	bar := pb.New(bmRequest.Runs)
	bar.ShowTimeLeft = false
	bar.SetWidth(100)

	bar.Start()
	for {
		select {
		case <-summary.Done:
			done = true
		case <-summary.Progress:
			bar.Set(summary.Runs)
			bar.Postfix(fmt.Sprintf(", RPS: %.2f", summary.CalculateRPS()))
		}

		if done {
			break
		}
	}
	bar.Finish()

	summary.PrintSummary()
}
