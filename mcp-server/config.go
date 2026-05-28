package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
)

// Config holds all runtime configuration loaded from config.json.
type Config struct {
	AllowedPaths          []string `json:"allowed_paths"`
	BlockedCommands       []string `json:"blocked_commands"`
	MaxFileSizeMB         int      `json:"max_file_size_mb"`
	CommandTimeoutSeconds int      `json:"command_timeout_seconds"`
	SearchMaxResults      int      `json:"search_max_results"`
}

// LoadConfig reads config.json from the same directory as the running
// executable and returns a parsed Config.
func LoadConfig() (*Config, error) {
	exePath, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("cannot determine executable path: %w", err)
	}
	cfgPath := filepath.Join(filepath.Dir(exePath), "config.json")

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		return nil, fmt.Errorf("cannot read config.json: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("cannot parse config.json: %w", err)
	}

	// Expand environment variables in allowed_paths.
	for i, p := range cfg.AllowedPaths {
		cfg.AllowedPaths[i] = expandEnvVars(p)
	}

	return &cfg, nil
}

// expandEnvVars replaces every occurrence of %VARNAME% with the value
// of the corresponding environment variable.
func expandEnvVars(s string) string {
	re := regexp.MustCompile(`%([^%]+)%`)
	return re.ReplaceAllStringFunc(s, func(m string) string {
		varName := m[1 : len(m)-1]
		if val := os.Getenv(varName); val != "" {
			return val
		}
		return m // keep original if env var is not set
	})
}
