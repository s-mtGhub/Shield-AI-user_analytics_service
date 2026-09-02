package config

import (
	"fmt"
	"os"
	"time"
)

// Config holds service configuration loaded from environment variables.
type Config struct {
	Port            string
	DatabaseURL     string
	ServiceTimezone *time.Location
	MigrationsPath  string
}

// Load reads configuration from the environment.
func Load() (*Config, error) {
	dbURL := os.Getenv(envDatabaseURL)
	if dbURL == "" {
		return nil, fmt.Errorf("%s is required", envDatabaseURL)
	}

	tzName := getEnv(envServiceTimezone, defaultServiceTimezone)
	loc, err := time.LoadLocation(tzName)
	if err != nil {
		return nil, fmt.Errorf("invalid %s %q: %w", envServiceTimezone, tzName, err)
	}

	return &Config{
		Port:            getEnv(envPort, defaultPort),
		DatabaseURL:     dbURL,
		ServiceTimezone: loc,
		MigrationsPath:  getEnv(envMigrationsPath, defaultMigrationsPath),
	}, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
