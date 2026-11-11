// Package kit is the shared platform library for all Go services of ethio-fair-market:
// configuration, logging, HTTP server, middleware, metrics, health checks, JWT and Postgres.
// It uses only the Go standard library plus the lib/pq driver, to keep the supply chain small.
package kit

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Env returns the environment variable or a default.
func Env(key, def string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return def
}

// MustEnv returns the environment variable or an error if it is missing.
func MustEnv(key string) (string, error) {
	v := os.Getenv(key)
	if v == "" {
		return "", fmt.Errorf("required environment variable %s is not set", key)
	}
	return v, nil
}

// EnvInt parses an integer environment variable.
func EnvInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

// EnvBool parses a boolean environment variable (true/false/1/0).
func EnvBool(key string, def bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return def
}

// EnvDuration parses a duration environment variable such as "5m" or "30s".
func EnvDuration(key string, def time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}
