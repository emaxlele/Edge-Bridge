package security

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Sandbox enforces path and command restrictions.
type Sandbox struct {
	AllowedPaths    []string
	BlockedCommands []string
	MaxFileSizeMB   int
}

// NewSandbox creates a Sandbox with environment-expanded, absolute paths.
func NewSandbox(allowedPaths, blockedCommands []string, maxFileSizeMB int) *Sandbox {
	resolved := make([]string, 0, len(allowedPaths))
	for _, p := range allowedPaths {
		expanded := expandEnvVars(p)
		abs, err := filepath.Abs(expanded)
		if err == nil {
			resolved = append(resolved, strings.ToLower(filepath.Clean(abs)))
		}
	}
	lower := make([]string, len(blockedCommands))
	for i, c := range blockedCommands {
		lower[i] = strings.ToLower(c)
	}
	return &Sandbox{
		AllowedPaths:    resolved,
		BlockedCommands: lower,
		MaxFileSizeMB:   maxFileSizeMB,
	}
}

// ValidatePath checks that the given path resides under one of the
// allowed directories and contains no path-traversal sequences.
func (s *Sandbox) ValidatePath(path string) error {
	if strings.Contains(path, "..") {
		return fmt.Errorf("path traversal (..) is not allowed")
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("cannot resolve path: %w", err)
	}
	normalised := strings.ToLower(filepath.Clean(abs))
	for _, allowed := range s.AllowedPaths {
		if strings.HasPrefix(normalised, allowed) {
			return nil
		}
	}
	return fmt.Errorf("path %q is outside the allowed directories", path)
}

// ValidateCommand checks whether the command string contains a blocked
// sub-command.
func (s *Sandbox) ValidateCommand(cmd string) error {
	lower := strings.ToLower(cmd)
	for _, blocked := range s.BlockedCommands {
		if strings.Contains(lower, blocked) {
			return fmt.Errorf("command contains blocked keyword %q", blocked)
		}
	}
	return nil
}

// expandEnvVars replaces %VAR% with the value of the environment variable.
func expandEnvVars(s string) string {
	re := regexp.MustCompile(`%([^%]+)%`)
	return re.ReplaceAllStringFunc(s, func(m string) string {
		varName := m[1 : len(m)-1]
		if val := os.Getenv(varName); val != "" {
			return val
		}
		return m
	})
}
