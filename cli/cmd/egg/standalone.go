// Package main provides the egg CLI standalone command.
//
// Overview:
//   - Responsibility: Manage standalone Go backend services that depend on egg framework
//   - Key Types: Standalone commands for init, build, run, and run-image
//   - Concurrency Model: Sequential command execution with context support
//   - Error Semantics: User-friendly error messages with suggestions
//   - Performance Notes: Fast command resolution, minimal initialization
//
// Usage:
//
//	egg standalone init <name>
//	egg standalone build
//	egg standalone run
//	egg standalone run-image
package main

import (
	"github.com/spf13/cobra"
)

// standaloneCmd represents the standalone command.
var standaloneCmd = &cobra.Command{
	Use:   "standalone",
	Short: "Manage standalone backend services",
	Long: `Manage standalone Go backend services that depend on the egg framework.

This command provides tools for:
- Initializing standalone service projects
- Building Docker images with multi-platform support
- Running services locally with .env configuration
- Running Docker containers with .env configuration

Standalone services follow the same structure as full egg project services
but exist independently without requiring an egg.yaml configuration file.

Examples:
  egg standalone init my-service --proto crud
  egg standalone build --push
  egg standalone run
  egg standalone run-image my-service:latest`,
}

func init() {
	rootCmd.AddCommand(standaloneCmd)
}

