# Cloud Ops Health Check Tool

This tool queries the health check endpoint on multiple servers and generates an aggregated report of their health status. It processes server endpoints in parallel with rate limiting to prevent overwhelming the servers.

## Features

- Reads server endpoints from a text file
- Concurrent health checks with configurable rate limiting
- Aggregates success rates by application and version
- Outputs human-readable report to stdout
- Saves machine-readable report in JSON format
- Implements timeout handling and error recovery
- Rate limiting with 200ms delay between requests
- Maximum concurrent requests limit

## Prerequisites

- Go 1.19 or higher
- Linux/Unix environment (can run on other OS but commands may differ)

## Setup

1. Clone the repository:

```bash
git clone https://github.com/edeediong/cloud-ops-interview-edeediong
cd cloud-ops-interview-edeediong
```

2. Copy the `servers.txt` file in the Google Drive to the root directory:

```text
server-0001.cloud-ops-interview.sgdev.org
server-0002.cloud-ops-interview.sgdev.org
...
```

**Note**: This project uses only Go standard library packages, so there's no need to initialize a Go module or install dependencies. However, if you prefer to set up proper Go module initialization, you can do:

```bash
go mod init cloud-ops-interview
```

## Running the Application

To run the main application:

```bash
go run main.go
```

The program will:

1. Read server endpoints from `servers.txt`
2. Query the `/healthz` endpoint of each server
3. Display an aggregated report to stdout
4. Save a detailed JSON report to `report.json`

## Running Tests

To run all tests:

```bash
go test -v
```

The test suite includes:

- Server list file reading
- Health data fetching
- Data aggregation
- Rate limiting and concurrency handling

> **NOTE**: As at the time of writing this, the test coverage is 56.4% of statements.

## Configuration

The following parameters can be adjusted in `main.go`:

- `maxConcurrency`: Maximum number of concurrent requests (default: 5)
- HTTP client timeout: Request timeout duration (default: 10 seconds)
- Request delay: Delay between requests (default: 200ms)

## Output Formats

### Standard Output

```yaml
Health Report:
  Application: Memcache2
  Version: 1.0.1
  Success Rate: 79.93%
```

### JSON Output (report.json)

```json
{
  "Memcache2": {
    "1.0.1": {
      "Application": "Memcache2",
      "Version": "1.0.1",
      "TotalRequests": 5194800029,
      "TotalSuccesses": 4151986778
    }
  }
}
```

## Error Handling

The application handles several types of errors:

- File reading errors
- Network connection failures
- Invalid JSON responses
- HTTP status errors
- Timeout issues

Failed requests are logged to stdout but don't halt the program execution.

## Performance Considerations

- Uses goroutines for concurrent processing
- Implements rate limiting to prevent server overload
- Employs connection pooling via HTTP client
- Buffers channel operations for efficient memory usage
- Configurable concurrency limits

## Project Structure

```ini
.
├── main.go           # Main application code
├── main_test.go      # Test suite
├── servers.txt       # Input file with server endpoints
├── README.md         # Documentation (this file)
└── report.json       # Generated report (created after running)
```

## License

This project is open-source and available using the GNU General Public License v3.0.
