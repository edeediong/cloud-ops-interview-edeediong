package main

import (
	"os"
	"strconv"
	"time"
)

// Config holds all configuration settings
type Config struct {
	HTTPTimeout    time.Duration
	RequestDelay   time.Duration
	MaxConcurrency int
}

// Configuration constants with default values
const (
	defaultHTTPTimeout    = 10 * time.Second
	defaultRequestDelay   = 200 * time.Millisecond
	defaultMaxConcurrency = 5
)

// NewDefaultConfig creates a Config with default values
func NewDefaultConfig() *Config {
	return &Config{
		HTTPTimeout:    defaultHTTPTimeout,
		RequestDelay:   defaultRequestDelay,
		MaxConcurrency: defaultMaxConcurrency,
	}
}

// LoadConfigFromEnv loads configuration from environment variables
func LoadConfigFromEnv() *Config {
	config := NewDefaultConfig()

	if timeout := os.Getenv("HTTP_TIMEOUT"); timeout != "" {
		if v, err := strconv.Atoi(timeout); err == nil {
			config.HTTPTimeout = time.Duration(v) * time.Second
		}
	}

	if delay := os.Getenv("REQUEST_DELAY"); delay != "" {
		if v, err := strconv.Atoi(delay); err == nil {
			config.RequestDelay = time.Duration(v) * time.Millisecond
		}
	}

	if concurrency := os.Getenv("MAX_CONCURRENCY"); concurrency != "" {
		if v, err := strconv.Atoi(concurrency); err == nil {
			config.MaxConcurrency = v
		}
	}

	return config
}
