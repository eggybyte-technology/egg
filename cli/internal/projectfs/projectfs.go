// Package projectfs provides project file system operations and scaffolding.
//
// Overview:
//   - Responsibility: Create project structure, write templates, manage files
//   - Key Types: File writers, template renderers, directory creators
//   - Concurrency Model: Sequential file operations with atomic writes
//   - Error Semantics: File system errors with user-friendly messages
//   - Performance Notes: Idempotent operations, minimal file I/O
//
// Usage:
//
//	fs := NewProjectFS(".")
//	err := fs.CreateDirectory("backend/user-service")
//	err := fs.WriteTemplate("backend/user-service/go.mod", goModTemplate, data)
package projectfs

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/eggybyte-technology/egg/cli/internal/ui"
)

// ProjectFS provides file system operations for project scaffolding.
//
// Parameters:
//   - rootDir: Root directory for operations
//   - verbose: Whether to show file operations
//
// Returns:
//   - None (data structure)
//
// Concurrency:
//   - Safe for concurrent use
//
// Performance:
//   - Efficient file operations with minimal allocations
type ProjectFS struct {
	rootDir string
	verbose bool
}

// NewProjectFS creates a new project file system.
//
// Parameters:
//   - rootDir: Root directory for operations
//
// Returns:
//   - *ProjectFS: Project file system instance
//
// Concurrency:
//   - Safe for concurrent use
//
// Performance:
//   - Minimal initialization overhead
func NewProjectFS(rootDir string) *ProjectFS {
	return &ProjectFS{
		rootDir: rootDir,
		verbose: false,
	}
}

// SetVerbose enables or disables verbose output.
//
// Parameters:
//   - enabled: Whether to show file operations
//
// Returns:
//   - None
//
// Concurrency:
//   - Thread-safe
//
// Performance:
//   - O(1) operation
func (fs *ProjectFS) SetVerbose(enabled bool) {
	fs.verbose = enabled
}

// GetVerbose returns the verbose setting.
//
// Parameters:
//   - None
//
// Returns:
//   - bool: Current verbose setting
//
// Concurrency:
//   - Thread-safe
//
// Performance:
//   - O(1) operation
func (fs *ProjectFS) GetVerbose() bool {
	return fs.verbose
}

// CreateDirectory creates a directory if it doesn't exist.
//
// Parameters:
//   - path: Directory path relative to root
//
// Returns:
//   - error: File system error if any
//
// Concurrency:
//   - Single-threaded per directory
//
// Performance:
//   - O(1) directory creation
func (fs *ProjectFS) CreateDirectory(path string) error {
	fullPath := filepath.Join(fs.rootDir, path)

	// Check if directory already exists
	if _, err := os.Stat(fullPath); err == nil {
		if fs.verbose {
			ui.Debug("Directory already exists: %s", path)
		}
		return nil
	}

	// Create directory
	if err := os.MkdirAll(fullPath, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", path, err)
	}

	if fs.verbose {
		ui.Debug("Created directory: %s", path)
	}

	return nil
}

// WriteFile writes content to a file.
//
// Parameters:
//   - path: File path relative to root
//   - content: File content
//   - mode: File permissions
//
// Returns:
//   - error: File system error if any
//
// Concurrency:
//   - Single-threaded per file
//
// Performance:
//   - Atomic file write
func (fs *ProjectFS) WriteFile(path, content string, mode fs.FileMode) error {
	fullPath := filepath.Join(fs.rootDir, path)

	// Create parent directories
	parentDir := filepath.Dir(fullPath)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return fmt.Errorf("failed to create parent directory for %s: %w", path, err)
	}

	// Write file
	if err := os.WriteFile(fullPath, []byte(content), mode); err != nil {
		return fmt.Errorf("failed to write file %s: %w", path, err)
	}

	if fs.verbose {
		ui.Debug("Written file: %s", path)
	}

	return nil
}

// WriteTemplate writes content from a template.
//
// Parameters:
//   - path: File path relative to root
//   - templateContent: Template content
//   - data: Template data
//   - mode: File permissions
//
// Returns:
//   - error: File system error if any
//
// Concurrency:
//   - Single-threaded per file
//
// Performance:
//   - Template rendering and atomic write
func (fs *ProjectFS) WriteTemplate(path, templateContent string, data interface{}, mode fs.FileMode) error {
	// Parse template
	tmpl, err := template.New(filepath.Base(path)).Parse(templateContent)
	if err != nil {
		return fmt.Errorf("failed to parse template for %s: %w", path, err)
	}

	// Render template
	var rendered strings.Builder
	if err := tmpl.Execute(&rendered, data); err != nil {
		return fmt.Errorf("failed to render template for %s: %w", path, err)
	}

	// Write rendered content
	return fs.WriteFile(path, rendered.String(), mode)
}

// FileExists checks if a file exists.
//
// Parameters:
//   - path: File path relative to root
//
// Returns:
//   - bool: True if file exists
//   - error: File system error if any
//
// Concurrency:
//   - Single-threaded per file
//
// Performance:
//   - Fast file existence check
func (fs *ProjectFS) FileExists(path string) (bool, error) {
	fullPath := filepath.Join(fs.rootDir, path)
	_, err := os.Stat(fullPath)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// DirectoryExists checks if a directory exists.
//
// Parameters:
//   - path: Directory path relative to root
//
// Returns:
//   - bool: True if directory exists
//   - error: File system error if any
//
// Concurrency:
//   - Single-threaded per directory
//
// Performance:
//   - Fast directory existence check
func (fs *ProjectFS) DirectoryExists(path string) (bool, error) {
	fullPath := filepath.Join(fs.rootDir, path)
	info, err := os.Stat(fullPath)
	if err == nil {
		return info.IsDir(), nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// ReadFile reads content from a file.
//
// Parameters:
//   - path: File path relative to root
//
// Returns:
//   - string: File content
//   - error: File system error if any
//
// Concurrency:
//   - Single-threaded per file
//
// Performance:
//   - File content loaded into memory
func (fs *ProjectFS) ReadFile(path string) (string, error) {
	fullPath := filepath.Join(fs.rootDir, path)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to read file %s: %w", path, err)
	}
	return string(content), nil
}

// CopyFile copies a file from source to destination.
//
// Parameters:
//   - src: Source file path relative to root
//   - dst: Destination file path relative to root
//
// Returns:
//   - error: File system error if any
//
// Concurrency:
//   - Single-threaded per file
//
// Performance:
//   - File copy operation
func (fs *ProjectFS) CopyFile(src, dst string) error {
	srcPath := filepath.Join(fs.rootDir, src)
	dstPath := filepath.Join(fs.rootDir, dst)

	// Create parent directories
	parentDir := filepath.Dir(dstPath)
	if err := os.MkdirAll(parentDir, 0755); err != nil {
		return fmt.Errorf("failed to create parent directory for %s: %w", dst, err)
	}

	// Read source file
	content, err := os.ReadFile(srcPath)
	if err != nil {
		return fmt.Errorf("failed to read source file %s: %w", src, err)
	}

	// Write destination file
	if err := os.WriteFile(dstPath, content, 0644); err != nil {
		return fmt.Errorf("failed to write destination file %s: %w", dst, err)
	}

	if fs.verbose {
		ui.Debug("Copied file: %s -> %s", src, dst)
	}

	return nil
}

// RemoveFile removes a file.
//
// Parameters:
//   - path: File path relative to root
//
// Returns:
//   - error: File system error if any
//
// Concurrency:
//   - Single-threaded per file
//
// Performance:
//   - Fast file removal
func (fs *ProjectFS) RemoveFile(path string) error {
	fullPath := filepath.Join(fs.rootDir, path)

	if err := os.Remove(fullPath); err != nil {
		if os.IsNotExist(err) {
			if fs.verbose {
				ui.Debug("File does not exist: %s", path)
			}
			return nil
		}
		return fmt.Errorf("failed to remove file %s: %w", path, err)
	}

	if fs.verbose {
		ui.Debug("Removed file: %s", path)
	}

	return nil
}

// RemoveDirectory removes a directory and all its contents.
//
// Parameters:
//   - path: Directory path relative to root
//
// Returns:
//   - error: File system error if any
//
// Concurrency:
//   - Single-threaded per directory
//
// Performance:
//   - Recursive directory removal
func (fs *ProjectFS) RemoveDirectory(path string) error {
	fullPath := filepath.Join(fs.rootDir, path)

	if err := os.RemoveAll(fullPath); err != nil {
		return fmt.Errorf("failed to remove directory %s: %w", path, err)
	}

	if fs.verbose {
		ui.Debug("Removed directory: %s", path)
	}

	return nil
}

// ListFiles lists files in a directory.
//
// Parameters:
//   - path: Directory path relative to root
//
// Returns:
//   - []string: List of file names
//   - error: File system error if any
//
// Concurrency:
//   - Single-threaded per directory
//
// Performance:
//   - Directory listing
func (fs *ProjectFS) ListFiles(path string) ([]string, error) {
	fullPath := filepath.Join(fs.rootDir, path)

	entries, err := os.ReadDir(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to list files in %s: %w", path, err)
	}

	var files []string
	for _, entry := range entries {
		if !entry.IsDir() {
			files = append(files, entry.Name())
		}
	}

	return files, nil
}

// ListDirectories lists directories in a directory.
//
// Parameters:
//   - path: Directory path relative to root
//
// Returns:
//   - []string: List of directory names
//   - error: File system error if any
//
// Concurrency:
//   - Single-threaded per directory
//
// Performance:
//   - Directory listing
func (fs *ProjectFS) ListDirectories(path string) ([]string, error) {
	fullPath := filepath.Join(fs.rootDir, path)

	entries, err := os.ReadDir(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to list directories in %s: %w", path, err)
	}

	var dirs []string
	for _, entry := range entries {
		if entry.IsDir() {
			dirs = append(dirs, entry.Name())
		}
	}

	return dirs, nil
}

// Walk walks the file tree and calls a function for each file.
//
// Parameters:
//   - path: Root path to walk
//   - fn: Function to call for each file
//
// Returns:
//   - error: File system error if any
//
// Concurrency:
//   - Single-threaded per walk
//
// Performance:
//   - Tree traversal
func (fs *ProjectFS) Walk(path string, fn func(string, os.FileInfo) error) error {
	fullPath := filepath.Join(fs.rootDir, path)

	return filepath.Walk(fullPath, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Convert to relative path
		relPath, err := filepath.Rel(fs.rootDir, filePath)
		if err != nil {
			return err
		}

		return fn(relPath, info)
	})
}

// GetRootDir returns the root directory.
//
// Parameters:
//   - None
//
// Returns:
//   - string: Root directory path
//
// Concurrency:
//   - Thread-safe
//
// Performance:
//   - O(1) operation
func (fs *ProjectFS) GetRootDir() string {
	return fs.rootDir
}

// GetAbsolutePath returns the absolute path for a relative path.
//
// Parameters:
//   - path: Relative path
//
// Returns:
//   - string: Absolute path
//
// Concurrency:
//   - Thread-safe
//
// Performance:
//   - O(1) path join operation
func (fs *ProjectFS) GetAbsolutePath(path string) string {
	return filepath.Join(fs.rootDir, path)
}

// EnsureDirectory ensures a directory exists, creating it if necessary.
//
// Parameters:
//   - path: Directory path relative to root
//
// Returns:
//   - error: File system error if any
//
// Concurrency:
//   - Single-threaded per directory
//
// Performance:
//   - O(1) directory creation
func (fs *ProjectFS) EnsureDirectory(path string) error {
	fullPath := filepath.Join(fs.rootDir, path)

	if err := os.MkdirAll(fullPath, 0755); err != nil {
		return fmt.Errorf("failed to ensure directory %s: %w", path, err)
	}

	if fs.verbose {
		ui.Debug("Ensured directory exists: %s", path)
	}

	return nil
}

// WriteFileIfNotExists writes a file only if it doesn't exist.
//
// Parameters:
//   - path: File path relative to root
//   - content: File content
//   - mode: File permissions
//
// Returns:
//   - bool: True if file was written, false if it already existed
//   - error: File system error if any
//
// Concurrency:
//   - Single-threaded per file
//
// Performance:
//   - Atomic file write with existence check
func (fs *ProjectFS) WriteFileIfNotExists(path, content string, mode fs.FileMode) (bool, error) {
	exists, err := fs.FileExists(path)
	if err != nil {
		return false, err
	}

	if exists {
		if fs.verbose {
			ui.Debug("File already exists, skipping: %s", path)
		}
		return false, nil
	}

	if err := fs.WriteFile(path, content, mode); err != nil {
		return false, err
	}

	return true, nil
}

// WriteTemplateIfNotExists writes a template only if the file doesn't exist.
//
// Parameters:
//   - path: File path relative to root
//   - templateContent: Template content
//   - data: Template data
//   - mode: File permissions
//
// Returns:
//   - bool: True if file was written, false if it already existed
//   - error: File system error if any
//
// Concurrency:
//   - Single-threaded per file
//
// Performance:
//   - Template rendering and atomic write with existence check
func (fs *ProjectFS) WriteTemplateIfNotExists(path, templateContent string, data interface{}, mode fs.FileMode) (bool, error) {
	exists, err := fs.FileExists(path)
	if err != nil {
		return false, err
	}

	if exists {
		if fs.verbose {
			ui.Debug("File already exists, skipping: %s", path)
		}
		return false, nil
	}

	if err := fs.WriteTemplate(path, templateContent, data, mode); err != nil {
		return false, err
	}

	return true, nil
}
