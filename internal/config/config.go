// Package config loads clipfolio's runtime configuration from environment variables.
package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config is clipfolio's runtime configuration, read from environment
// variables (see Load).
type Config struct {
	Addr string

	DatabaseURL string

	S3Endpoint  string
	S3Region    string
	S3Bucket    string
	S3AccessKey string
	S3SecretKey string
	// S3ForcePathStyle is required by most non-AWS S3-compatible providers
	// (MinIO, R2, B2) since they don't support virtual-hosted-style bucket URLs.
	S3ForcePathStyle bool
	// S3PublicBaseURL is the URL prefix viewers fetch media from (e.g. a CDN in
	// front of the bucket, or the bucket's own public endpoint). Falls back to
	// S3Endpoint/S3Bucket when unset.
	S3PublicBaseURL string

	// SetupToken gates first-run account creation via POST /api/setup. Empty
	// disables setup entirely (use once an admin account already exists).
	SetupToken string
}

// Load reads Config from the process environment, returning an error if a
// required variable is missing.
func Load() (Config, error) {
	cfg := Config{
		Addr:            getEnv("CLIPFOLIO_ADDR", ":8080"),
		DatabaseURL:     os.Getenv("CLIPFOLIO_DATABASE_URL"),
		S3Endpoint:      os.Getenv("CLIPFOLIO_S3_ENDPOINT"),
		S3Region:        getEnv("CLIPFOLIO_S3_REGION", "us-east-1"),
		S3Bucket:        os.Getenv("CLIPFOLIO_S3_BUCKET"),
		S3AccessKey:     os.Getenv("CLIPFOLIO_S3_ACCESS_KEY"),
		S3SecretKey:     os.Getenv("CLIPFOLIO_S3_SECRET_KEY"),
		S3PublicBaseURL: os.Getenv("CLIPFOLIO_S3_PUBLIC_BASE_URL"),
		SetupToken:      os.Getenv("CLIPFOLIO_SETUP_TOKEN"),
	}

	forcePathStyle, err := getEnvBool("CLIPFOLIO_S3_FORCE_PATH_STYLE", true)
	if err != nil {
		return Config{}, err
	}
	cfg.S3ForcePathStyle = forcePathStyle

	if cfg.DatabaseURL == "" {
		return Config{}, fmt.Errorf("CLIPFOLIO_DATABASE_URL is required")
	}
	if cfg.S3Bucket == "" {
		return Config{}, fmt.Errorf("CLIPFOLIO_S3_BUCKET is required")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvBool(key string, fallback bool) (bool, error) {
	v := os.Getenv(key)
	if v == "" {
		return fallback, nil
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return false, fmt.Errorf("%s: invalid bool %q", key, v)
	}
	return b, nil
}
