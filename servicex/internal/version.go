// Package internal provides internal implementation for the servicex package.
package internal

// BuildTime holds the build/release timestamp in format YYYYMMDDHHMMSS.
// This value is automatically updated by the release script during make release.
// Format: 20251030132945 represents October 30, 2025 at 13:29:45
//
// Example:
//   BuildTime = "20251030132945"  // Year: 2025, Month: 10, Day: 30, Hour: 13, Minute: 29, Second: 45
//
// This value is set by the release script and should not be manually edited.
// It is displayed at service startup to help identify which build/release is running.
var BuildTime = "20251031144156" // Default fallback: January 1, 2025 00:00:00

