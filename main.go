package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

type HealthResponse struct {
	Application  string `json:"application"`
	Version      string `json:"version"`
	Uptime       int64  `json:"uptime"`
	RequestCount int64  `json:"requestCount"`
	ErrorCount   int64  `json:"errorCount"`
	SuccessCount int64  `json:"successCount"`
}

type AggregatedData struct {
	Application    string
	Version        string
	TotalRequests  int64
	TotalSuccesses int64
}

// Function to fetch health data from a server with configurable timeout
func fetchHealthData(serverURL string, timeout time.Duration) (HealthResponse, error) {
	var health HealthResponse
	client := &http.Client{Timeout: timeout}

	resp, err := client.Get(serverURL)
	if err != nil {
		return health, fmt.Errorf("failed to reach server %s: %v", serverURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return health, fmt.Errorf("server %s returned status %d: %s", serverURL, resp.StatusCode, string(body))
	}

	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		body, _ := io.ReadAll(resp.Body)
		return health, fmt.Errorf("failed to decode JSON from server %s: %v. Response: %s", serverURL, err, string(body))
	}

	return health, nil
}

func fetchHealthDataWithDelayAndConcurrency(
	servers []string,
	dataChannel chan<- AggregatedData,
	config *Config,
) {
	var wg sync.WaitGroup
	sem := make(chan struct{}, config.MaxConcurrency)

	for _, server := range servers {
		wg.Add(1)
		go func(server string) {
			defer wg.Done()

			if !strings.HasPrefix(server, "http://") && !strings.HasPrefix(server, "https://") {
				server = "https://" + server
			}

			serverURL := server + "/healthz"

			sem <- struct{}{}
			defer func() { <-sem }()
			time.Sleep(config.RequestDelay)

			health, err := fetchHealthData(serverURL, config.HTTPTimeout)
			if err != nil {
				fmt.Printf("Error fetching data from %s: %v\n", serverURL, err)
				return
			}

			dataChannel <- AggregatedData{
				Application:    health.Application,
				Version:        health.Version,
				TotalRequests:  health.RequestCount,
				TotalSuccesses: health.SuccessCount,
			}
		}(server)
	}

	wg.Wait()
	close(dataChannel)
}

func readServersList(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var servers []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		servers = append(servers, scanner.Text())
	}
	return servers, scanner.Err()
}

func aggregateData(data []AggregatedData) map[string]map[string]AggregatedData {
	aggregation := make(map[string]map[string]AggregatedData)
	for _, d := range data {
		if _, exists := aggregation[d.Application]; !exists {
			aggregation[d.Application] = make(map[string]AggregatedData)
		}
		agg := aggregation[d.Application][d.Version]
		agg.Application = d.Application
		agg.Version = d.Version
		agg.TotalRequests += d.TotalRequests
		agg.TotalSuccesses += d.TotalSuccesses
		aggregation[d.Application][d.Version] = agg
	}
	return aggregation
}

func main() {
	// Load configuration
	config := LoadConfigFromEnv()

	// Log current configuration
	fmt.Printf("Running with configuration:\n")
	fmt.Printf("- HTTP Timeout: %v\n", config.HTTPTimeout)
	fmt.Printf("- Request Delay: %v\n", config.RequestDelay)
	fmt.Printf("- Max Concurrency: %d\n\n", config.MaxConcurrency)

	servers, err := readServersList("servers.txt")
	if err != nil {
		fmt.Println("Error reading servers list:", err)
		return
	}

	dataChannel := make(chan AggregatedData, len(servers))

	go fetchHealthDataWithDelayAndConcurrency(servers, dataChannel, config)

	var collectedData []AggregatedData
	for data := range dataChannel {
		collectedData = append(collectedData, data)
	}

	aggregation := aggregateData(collectedData)

	fmt.Println("Health Report:")
	for app, versions := range aggregation {
		for version, data := range versions {
			successRate := float64(data.TotalSuccesses) / float64(data.TotalRequests) * 100
			fmt.Printf("Application: %s, Version: %s, Success Rate: %.2f%%\n",
				app, version, successRate)
		}
	}

	outputFile := "report.json"
	jsonData, err := json.MarshalIndent(aggregation, "", "  ")
	if err != nil {
		fmt.Println("Error encoding JSON:", err)
		return
	}
	if err := os.WriteFile(outputFile, jsonData, 0644); err != nil {
		fmt.Println("Error writing to file:", err)
		return
	}
	fmt.Printf("Report saved to %s\n", outputFile)
}
