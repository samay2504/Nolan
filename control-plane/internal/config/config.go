package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds all configuration for the control-plane service.
type Config struct {
	// HTTP server
	Port int

	// Valkey (Redis-compatible)
	ValkeyAddr     string
	ValkeyUser     string
	ValkeyPassword string

	// MinIO
	MinIOEndpoint  string
	MinIOAccessKey string
	MinIOSecretKey string
	MinIOUseSSL    bool

	// Postgres
	PostgresDSN string

	// Presigned URL TTLs
	UploadURLTTL   time.Duration
	DownloadURLTTL time.Duration

	// Reaper
	ReaperInterval time.Duration
	ReclaimTimeout time.Duration
	MaxRetries     int

	// Request timeout
	RequestTimeout time.Duration
}

// Load reads configuration from environment variables and validates all
// required fields. It returns an error immediately on any missing or
// invalid value.
func Load() (*Config, error) {
	c := &Config{}

	port, err := envInt("PORT", 8080)
	if err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}
	c.Port = port

	c.ValkeyAddr, err = envRequired("VALKEY_ADDR")
	if err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}
	c.ValkeyUser = os.Getenv("VALKEY_USER")
	c.ValkeyPassword = os.Getenv("VALKEY_PASSWORD")

	c.MinIOEndpoint, err = envRequired("MINIO_ENDPOINT")
	if err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}
	c.MinIOAccessKey, err = envRequired("MINIO_ACCESS_KEY")
	if err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}
	c.MinIOSecretKey, err = envRequired("MINIO_SECRET_KEY")
	if err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}
	c.MinIOUseSSL = os.Getenv("MINIO_USE_SSL") == "true"

	c.PostgresDSN, err = envRequired("POSTGRES_DSN")
	if err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}

	uploadTTL, err := envInt("UPLOAD_URL_TTL_SECONDS", 3600)
	if err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}
	c.UploadURLTTL = time.Duration(uploadTTL) * time.Second

	downloadTTL, err := envInt("DOWNLOAD_URL_TTL_SECONDS", 3600)
	if err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}
	c.DownloadURLTTL = time.Duration(downloadTTL) * time.Second

	reaperSec, err := envInt("REAPER_INTERVAL_SECONDS", 30)
	if err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}
	c.ReaperInterval = time.Duration(reaperSec) * time.Second

	reclaimSec, err := envInt("RECLAIM_TIMEOUT_SECONDS", 300)
	if err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}
	c.ReclaimTimeout = time.Duration(reclaimSec) * time.Second

	c.MaxRetries, err = envInt("MAX_RETRIES", 3)
	if err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}

	reqTimeout, err := envInt("REQUEST_TIMEOUT_SECONDS", 30)
	if err != nil {
		return nil, fmt.Errorf("config: %w", err)
	}
	c.RequestTimeout = time.Duration(reqTimeout) * time.Second

	return c, nil
}

func envRequired(key string) (string, error) {
	v := os.Getenv(key)
	if v == "" {
		return "", fmt.Errorf("required environment variable %s is not set", key)
	}
	return v, nil
}

func envInt(key string, defaultVal int) (int, error) {
	v := os.Getenv(key)
	if v == "" {
		return defaultVal, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("environment variable %s must be an integer: %w", key, err)
	}
	return n, nil
}
