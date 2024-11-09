package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

// Mock response for the /healthz endpoint
var mockResponse = `{
	"application": "Memcache2",
	"version": "1.0.1",
	"uptime": 4637719417,
	"requestCount": 5194800029,
	"errorCount": 1042813251,
	"successCount": 4151986778
}`

// Test reading servers list from file
func TestReadServersList(t *testing.T) {
	content := "server-0001.cloud-ops-interview.sgdev.org\nserver-0002.cloud-ops-interview.sgdev.org\n"
	file, err := os.CreateTemp("", "servers.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(file.Name())

	if _, err := file.WriteString(content); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}

	servers, err := readServersList(file.Name())
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	expected := []string{
		"server-0001.cloud-ops-interview.sgdev.org",
		"server-0002.cloud-ops-interview.sgdev.org",
	}

	if len(servers) != len(expected) {
		t.Errorf("Expected %d servers, got %d", len(expected), len(servers))
	}
	for i, server := range servers {
		if server != expected[i] {
			t.Errorf("Expected server %s, got %s", expected[i], server)
		}
	}
}

// Mock server to simulate /healthz endpoint
func setupMockServer() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(mockResponse))
	}))
}

// Test fetching health data with a mock server
func TestFetchHealthData(t *testing.T) {
	server := setupMockServer()
	defer server.Close()

	// Pass the full URL with "http://"
	data, err := fetchHealthData(server.URL)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if data.Application != "Memcache2" {
		t.Errorf("Expected application 'Memcache2', got %s", data.Application)
	}
	if data.Version != "1.0.1" {
		t.Errorf("Expected version '1.0.1', got %s", data.Version)
	}
	if data.SuccessCount != 4151986778 {
		t.Errorf("Expected successCount 4151986778, got %d", data.SuccessCount)
	}
}

// Test the aggregation of data
func TestAggregateData(t *testing.T) {
	data := []AggregatedData{
		{Application: "Memcache2", Version: "1.0.1", TotalRequests: 5194800029, TotalSuccesses: 4151986778},
		{Application: "Memcache2", Version: "1.0.1", TotalRequests: 1000000000, TotalSuccesses: 800000000},
	}

	aggregation := aggregateData(data)

	expectedRequests := int64(6194800029)
	expectedSuccesses := int64(4951986778)

	agg, exists := aggregation["Memcache2"]["1.0.1"]
	if !exists {
		t.Fatalf("Expected application 'Memcache2' with version '1.0.1' in aggregation")
	}

	if agg.TotalRequests != expectedRequests {
		t.Errorf("Expected total requests %d, got %d", expectedRequests, agg.TotalRequests)
	}
	if agg.TotalSuccesses != expectedSuccesses {
		t.Errorf("Expected total successes %d, got %d", expectedSuccesses, agg.TotalSuccesses)
	}
}

// Test rate limiting and concurrency handling
func TestFetchHealthDataWithDelayAndConcurrency(t *testing.T) {
	servers := []string{}
	mockServer := setupMockServer()
	defer mockServer.Close()

	// Use the full mock server URL
	for i := 0; i < 3; i++ {
		servers = append(servers, mockServer.URL) // No manual protocol prefix
	}

	dataChannel := make(chan AggregatedData, len(servers))
	go fetchHealthDataWithDelayAndConcurrency(servers, dataChannel, 2) // Concurrency set to 2

	var result []AggregatedData
	for data := range dataChannel {
		result = append(result, data)
	}

	if len(result) != len(servers) {
		t.Errorf("Expected %d results, got %d", len(servers), len(result))
	}
	for _, data := range result {
		if data.Application != "Memcache2" {
			t.Errorf("Expected application 'Memcache2', got %s", data.Application)
		}
		if data.Version != "1.0.1" {
			t.Errorf("Expected version '1.0.1', got %s", data.Version)
		}
	}
}
