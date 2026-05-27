// SPDX-License-Identifier: Apache-2.0

package complytime

import (
	"fmt"
	"os"
	"path/filepath"
)

// Workspace manages loading, saving, and validating a complytime configuration
// file within a resolved workspace directory. It supports both the new
// .complytime/complytime.yaml location and legacy root-level complytime.yaml
// with automatic detection and deprecation warnings.
type Workspace struct {
	baseDir    string
	configPath string
	config     *WorkspaceConfig
}

// NewWorkspace returns a Workspace that operates on the complytime.yaml
// configuration file within the specified base directory.
// It detects the config location (new .complytime/ or legacy root) and
// prints a deprecation warning if the legacy location is detected.
func NewWorkspace(baseDir string) *Workspace {
	configPath, isLegacy, err := DetectConfigPath(baseDir)
	if err != nil {
		// If detection fails, default to new location path
		configPath = filepath.Join(baseDir, WorkspaceDir, WorkspaceConfigFile)
	} else if isLegacy {
		printDeprecationWarning()
	}
	return &Workspace{
		baseDir:    baseDir,
		configPath: configPath,
	}
}

// Load reads the workspace configuration from disk.
func (w *Workspace) Load() error {
	config, err := LoadFrom(w.configPath)
	if err != nil {
		return err
	}
	w.config = config
	return nil
}

// LoadAndValidate loads the workspace config and runs structural validation.
// Prefer this over separate Load() + Validate() calls in CLI entry points.
func (w *Workspace) LoadAndValidate() error {
	if err := w.Load(); err != nil {
		return err
	}
	return Validate(w.config)
}

// Save writes the workspace configuration to disk.
func (w *Workspace) Save() error {
	if w.config == nil {
		return fmt.Errorf("no configuration to save")
	}
	return SaveTo(w.config, w.configPath)
}

// Config returns the loaded workspace configuration, or nil if not yet loaded.
func (w *Workspace) Config() *WorkspaceConfig {
	return w.config
}

// SetConfig replaces the workspace configuration in memory.
func (w *Workspace) SetConfig(config *WorkspaceConfig) {
	w.config = config
}

// Exists returns true if the workspace configuration file exists on disk.
func (w *Workspace) Exists() bool {
	_, err := os.Stat(w.configPath)
	return err == nil
}

// Path returns the absolute path to the workspace configuration file.
func (w *Workspace) Path() string {
	return w.configPath
}

// BaseDir returns the resolved workspace base directory.
func (w *Workspace) BaseDir() string {
	return w.baseDir
}

// EnsureDir creates the directory containing the configuration file if it does not exist.
func (w *Workspace) EnsureDir() error {
	dir := filepath.Dir(w.configPath)
	if dir == "." {
		return nil
	}
	return os.MkdirAll(dir, 0700)
}

// ResolveWorkspaceDir determines the workspace directory using precedence:
// 1. flag value (if non-empty)
// 2. COMPLYTIME_WORKSPACE env var (if set)
// 3. current working directory
//
// Expands ~ to home directory and converts relative to absolute paths.
// Validates that the resolved path exists and is a directory.
func ResolveWorkspaceDir(flag string) (string, error) {
	path := flag
	if path == "" {
		path = os.Getenv(WorkspaceEnvVar)
	}
	if path == "" {
		path = "."
	}

	// Expand tilde to home directory
	if len(path) > 0 && path[0] == '~' {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory: %w", err)
		}
		if len(path) == 1 {
			path = home
		} else if len(path) > 1 && path[1] == filepath.Separator {
			path = filepath.Join(home, path[2:])
		}
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("failed to resolve absolute path: %w", err)
	}

	// Validate that the path exists and is a directory
	info, err := os.Stat(absPath) // #nosec G703 — path sanitized via filepath.Abs
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("workspace directory does not exist: %s", absPath)
		}
		return "", fmt.Errorf("failed to stat workspace directory: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("workspace path is not a directory: %s", absPath)
	}

	return absPath, nil
}

// DetectConfigPath locates the complytime.yaml configuration file with fallback logic.
// Checks .complytime/complytime.yaml (new location) first, then complytime.yaml (legacy).
// Returns the absolute path to the config file, whether legacy location was used, and any error.
// Returns an error if neither location exists.
func DetectConfigPath(baseDir string) (string, bool, error) {
	newConfigPath := filepath.Join(baseDir, WorkspaceDir, WorkspaceConfigFile)
	if _, err := os.Stat(newConfigPath); err == nil {
		return newConfigPath, false, nil
	}

	legacyConfigPath := filepath.Join(baseDir, WorkspaceConfigFile)
	if _, err := os.Stat(legacyConfigPath); err == nil {
		return legacyConfigPath, true, nil
	}

	return "", false, fmt.Errorf("config file not found in %s (checked .complytime/complytime.yaml and complytime.yaml)", baseDir)
}

func printDeprecationWarning() {
	fmt.Fprintf(os.Stderr, `WARNING: complytime.yaml found at repository root (legacy location).
Please move it to .complytime/complytime.yaml for better organization.
Run: mkdir -p .complytime && mv complytime.yaml .complytime/complytime.yaml

`)
}
