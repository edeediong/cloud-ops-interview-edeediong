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

// Function to read server list from file
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

// Function to fetch health data from a server with a custom timeout
func fetchHealthData(serverURL string) (HealthResponse, error) {
	var health HealthResponse
	client := &http.Client{Timeout: 10 * time.Second} // Increased timeout to 10 seconds
	resp, err := client.Get(serverURL)
	if err != nil {
		return health, fmt.Errorf("failed to reach server %s: %v", serverURL, err)
	}
	defer resp.Body.Close()

	// Check if the status is 200 OK
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return health, fmt.Errorf("server %s returned status %d: %s", serverURL, resp.StatusCode, string(body))
	}

	// Attempt to decode JSON if status is OK
	if err := json.NewDecoder(resp.Body).Decode(&health); err != nil {
		body, _ := io.ReadAll(resp.Body) // Read the body to log it for debugging
		return health, fmt.Errorf("failed to decode JSON from server %s: %v. Response: %s", serverURL, err, string(body))
	}

	return health, nil
}

// Fetch health data with delay and concurrency control, ensuring URL scheme and endpoint path
func fetchHealthDataWithDelayAndConcurrency(servers []string, dataChannel chan<- AggregatedData, maxConcurrency int) {
	var wg sync.WaitGroup
	sem := make(chan struct{}, maxConcurrency) // Limit concurrent requests

	for _, server := range servers {
		wg.Add(1)
		go func(server string) {
			defer wg.Done()

			// Ensure serverURL has a protocol scheme
			if !strings.HasPrefix(server, "http://") && !strings.HasPrefix(server, "https://") {
				server = "https://" + server
			}

			// Append the /healthz path to the server URL
			serverURL := server + "/healthz"

			sem <- struct{}{}                  // Acquire a slot
			defer func() { <-sem }()           // Release the slot
			time.Sleep(200 * time.Millisecond) // Delay between requests

			health, err := fetchHealthData(serverURL)
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

func main() {
	servers, err := readServersList("servers.txt")
	if err != nil {
		fmt.Println("Error reading servers list:", err)
		return
	}

	dataChannel := make(chan AggregatedData, len(servers))
	maxConcurrency := 5 // Limit to 5 concurrent requests

	go fetchHealthDataWithDelayAndConcurrency(servers, dataChannel, maxConcurrency)

	var collectedData []AggregatedData
	for data := range dataChannel {
		collectedData = append(collectedData, data)
	}

	// Aggregate and print data as before
	aggregation := aggregateData(collectedData)

	fmt.Println("Health Report:")
	for app, versions := range aggregation {
		for version, data := range versions {
			successRate := float64(data.TotalSuccesses) / float64(data.TotalRequests) * 100
			fmt.Printf("Application: %s, Version: %s, Success Rate: %.2f%%\n", app, version, successRate)
		}
	}

	// Optional: Write to JSON file
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

// aggregateData aggregates health data by application and version
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
