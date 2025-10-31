// Package version provides version information for the egg CLI tool.
//
// Overview:
//   - Responsibility: CLI version metadata (version, commit, build time)
//   - Key Types: Version constants and functions
//   - Concurrency Model: Immutable constants, safe for concurrent use
//   - Error Semantics: No errors (all constants)
//   - Performance Notes: Zero-cost constants
//
// Usage:
//
//	import "go.eggybyte.com/egg/cli/internal/version"
//	version.GetVersionString()
package version

import (
	"fmt"
	"runtime"
)

// Version is the CLI version.
// This value is set by cli-release.sh during release builds.
var Version = "v0.0.3-alpha.4"

// Commit is the git commit hash.
// This value is set by cli-release.sh during release builds.
var Commit = "c031b17"

// BuildTime is the build timestamp in RFC3339 format.
// This value is set by cli-release.sh during release builds.
var BuildTime = "2025-10-31T10:52:10Z"

// FrameworkVersion is the Egg framework version that this CLI release uses.
// This value is set by cli-release.sh during release builds.
var FrameworkVersion = "v0.3.1"

// GetVersionString returns the full version string in the format:
// egg version v0.3.0 (commit 4a9b2c1, built 2025-10-31T12:10:00Z)
//
// Returns:
//   - string: Formatted version string
//
// Concurrency:
//   - Safe for concurrent use
//
// Performance:
//   - O(1) string concatenation
func GetVersionString() string {
	return fmt.Sprintf("egg version %s (commit %s, built %s)", Version, Commit, BuildTime)
}

// GetFullVersionInfo returns detailed version information including framework version.
//
// Returns:
//   - string: Multi-line version information
//
// Concurrency:
//   - Safe for concurrent use
//
// Performance:
//   - O(1) string concatenation
func GetFullVersionInfo() string {
	return fmt.Sprintf(`egg version %s (commit %s, built %s)
egg framework version %s
go version %s (%s/%s)`,
		Version, Commit, BuildTime,
		FrameworkVersion,
		runtime.Version(), runtime.GOOS, runtime.GOARCH)
}
