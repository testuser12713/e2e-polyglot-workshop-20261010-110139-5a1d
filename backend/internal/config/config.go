// Package config reads the API's configuration from the environment. It is read
// exactly once at startup and validated there, so a missing required variable
// refuses to boot and names itself instead of dying on the first request.
package config

import (
	"fmt"
	"os"
	"strings"
)

// Config holds every value the API needs. No secret is ever given a default
// here — the value comes from the environment, which RUN.json declares.
type Config struct {
	DatabaseURL          string
	ValkeyURL            string
	QueueName            string
	HTTPPort             string
	JWTSecret            string
	CORSOrigin           string
	SeedEmployeeEmail    string
	SeedEmployeePassword string
}

// Load reads and validates the configuration. Every variable that the process
// needs to boot is required; the ones with a legitimate non-secret default fall
// back to it. A missing required variable is reported by name, all of them at
// once, and points at RUN.json.
func Load() (*Config, error) {
	var missing []string

	required := func(key string) string {
		v := strings.TrimSpace(os.Getenv(key))
		if v == "" {
			missing = append(missing, key)
		}
		return v
	}
	optional := func(key, fallback string) string {
		v := strings.TrimSpace(os.Getenv(key))
		if v == "" {
			return fallback
		}
		return v
	}

	cfg := &Config{
		DatabaseURL:          required("DATABASE_URL"),
		ValkeyURL:            required("VALKEY_URL"),
		JWTSecret:            required("JWT_SECRET"),
		CORSOrigin:           required("CORS_ORIGIN"),
		SeedEmployeeEmail:    required("SEED_EMPLOYEE_EMAIL"),
		SeedEmployeePassword: required("SEED_EMPLOYEE_PASSWORD"),
		QueueName:            optional("QUEUE_NAME", "workshop:invoices"),
		HTTPPort:             optional("HTTP_PORT", "8000"),
	}

	if len(missing) > 0 {
		return nil, fmt.Errorf(
			"missing required environment variables: %s (see RUN.json for how to set them)",
			strings.Join(missing, ", "),
		)
	}
	return cfg, nil
}
