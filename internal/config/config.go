// Package config loads statistics-service configuration from environment
// variables, with sane defaults for local development.
package config

import (
	"fmt"
	"os"
	"time"
)

// Config holds all runtime configuration for the service.
type Config struct {
	Env string // "development" or "production"

	HTTPPort            string
	HTTPReadTimeout     time.Duration
	HTTPWriteTimeout    time.Duration
	HTTPIdleTimeout     time.Duration
	HTTPShutdownTimeout time.Duration

	LogLevel string // "debug", "info", "warn", "error"

	// RedisAddr is the shared session store every BeeBase service checks on
	// every request, so an access token can be rejected the instant its
	// session is superseded by a newer one instead of staying valid until
	// its own JWT expiry.
	RedisAddr           string
	RedisConnectTimeout time.Duration

	// AuthJWKSURL points at auth-service's public key endpoint
	// (GET /.well-known/jwks.json), used to verify access tokens without
	// ever holding a key that could mint one.
	AuthJWKSURL string

	// ApiaryServiceURL, HiveServiceURL, InspectionServiceURL, and
	// HarvestServiceURL are the base URLs this service fetches a
	// caller's apiaries, hives, inspections, and harvest records from -
	// it holds no data of its own.
	ApiaryServiceURL       string
	HiveServiceURL         string
	InspectionServiceURL   string
	HarvestServiceURL      string
	SubscriptionServiceURL string
	InternalServiceToken   string
}

// Load builds a Config from environment variables, falling back to
// defaults suitable for local development where a variable is unset.
func Load() (*Config, error) {
	cfg := &Config{
		Env: getEnv("APP_ENV", "development"),

		HTTPPort:            getEnv("HTTP_PORT", "8080"),
		HTTPReadTimeout:     getDuration("HTTP_READ_TIMEOUT", 5*time.Second),
		HTTPWriteTimeout:    getDuration("HTTP_WRITE_TIMEOUT", 10*time.Second),
		HTTPIdleTimeout:     getDuration("HTTP_IDLE_TIMEOUT", 60*time.Second),
		HTTPShutdownTimeout: getDuration("HTTP_SHUTDOWN_TIMEOUT", 15*time.Second),

		LogLevel: getEnv("LOG_LEVEL", "info"),

		RedisAddr:           getEnv("REDIS_ADDR", ""),
		RedisConnectTimeout: getDuration("REDIS_CONNECT_TIMEOUT", 5*time.Second),

		AuthJWKSURL: getEnv("AUTH_JWKS_URL", ""),

		ApiaryServiceURL:       getEnv("APIARY_SERVICE_URL", ""),
		HiveServiceURL:         getEnv("HIVE_SERVICE_URL", ""),
		InspectionServiceURL:   getEnv("INSPECTION_SERVICE_URL", ""),
		HarvestServiceURL:      getEnv("HARVEST_SERVICE_URL", ""),
		SubscriptionServiceURL: getEnv("SUBSCRIPTION_SERVICE_URL", ""),
		InternalServiceToken:   getEnv("INTERNAL_SERVICE_TOKEN", ""),
	}

	required := []struct{ name, value string }{
		{"REDIS_ADDR", cfg.RedisAddr},
		{"AUTH_JWKS_URL", cfg.AuthJWKSURL},
		{"APIARY_SERVICE_URL", cfg.ApiaryServiceURL},
		{"HIVE_SERVICE_URL", cfg.HiveServiceURL},
		{"INSPECTION_SERVICE_URL", cfg.InspectionServiceURL},
		{"HARVEST_SERVICE_URL", cfg.HarvestServiceURL},
		{"SUBSCRIPTION_SERVICE_URL", cfg.SubscriptionServiceURL},
		{"INTERNAL_SERVICE_TOKEN", cfg.InternalServiceToken},
	}
	for _, r := range required {
		if r.value == "" {
			return nil, fmt.Errorf("config: %s is required", r.name)
		}
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

func getDuration(key string, fallback time.Duration) time.Duration {
	v, ok := os.LookupEnv(key)
	if !ok || v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return d
}
