// Package envloader provides utilities for loading environment variables from .env files.
//
// Overview:
//   - Responsibility: Parse and load .env files for standalone service execution
//   - Key Types: Environment variable maps and loading functions
//   - Concurrency Model: Single-threaded file reading
//   - Error Semantics: File not found and parse errors are clearly reported
//   - Performance Notes: Simple file parsing, minimal allocations
//
// Usage:
//
//	envMap, err := envloader.LoadEnvFile(".env")
//	envSlice := envloader.MapToSlice(envMap)
package envloader

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// LoadEnvFile loads environment variables from a .env file.
//
// The .env file format supports:
//   - KEY=value format
//   - Comments starting with #
//   - Empty lines
//   - Quoted values (single or double quotes)
//   - Variable references are not expanded
//
// Parameters:
//   - path: Path to .env file
//
// Returns:
//   - map[string]string: Environment variables as key-value pairs
//   - error: File read or parse error if any
//
// Concurrency:
//   - Single-threaded
//
// Performance:
//   - O(n) where n is number of lines
func LoadEnvFile(path string) (map[string]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open .env file: %w", err)
	}
	defer file.Close()

	envMap := make(map[string]string)
	scanner := bufio.NewScanner(file)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse KEY=value
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid line %d: %s (expected KEY=value format)", lineNum, line)
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Remove quotes if present
		if len(value) >= 2 {
			if (value[0] == '"' && value[len(value)-1] == '"') ||
				(value[0] == '\'' && value[len(value)-1] == '\'') {
				value = value[1 : len(value)-1]
			}
		}

		envMap[key] = value
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading .env file: %w", err)
	}

	return envMap, nil
}

// MapToSlice converts environment variable map to slice format.
//
// The slice format is suitable for use with exec.Cmd.Env.
//
// Parameters:
//   - envMap: Environment variables as key-value pairs
//
// Returns:
//   - []string: Environment variables in KEY=value format
//
// Concurrency:
//   - Safe for concurrent use
//
// Performance:
//   - O(n) where n is number of environment variables
func MapToSlice(envMap map[string]string) []string {
	envSlice := make([]string, 0, len(envMap))
	for key, value := range envMap {
		envSlice = append(envSlice, fmt.Sprintf("%s=%s", key, value))
	}
	return envSlice
}

// MergeWithOS merges loaded environment variables with existing OS environment.
//
// Loaded variables take precedence over OS environment variables.
//
// Parameters:
//   - envMap: Environment variables from .env file
//
// Returns:
//   - []string: Merged environment variables in KEY=value format
//
// Concurrency:
//   - Safe for concurrent use
//
// Performance:
//   - O(n + m) where n is OS env size and m is envMap size
func MergeWithOS(envMap map[string]string) []string {
	// Start with OS environment
	merged := os.Environ()

	// Create a map for quick lookup
	osEnvMap := make(map[string]bool)
	for _, env := range merged {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			osEnvMap[parts[0]] = true
		}
	}

	// Add or override with loaded environment
	for key, value := range envMap {
		envStr := fmt.Sprintf("%s=%s", key, value)
		if osEnvMap[key] {
			// Replace existing OS env
			for i, env := range merged {
				if strings.HasPrefix(env, key+"=") {
					merged[i] = envStr
					break
				}
			}
		} else {
			// Add new env
			merged = append(merged, envStr)
		}
	}

	return merged
}

