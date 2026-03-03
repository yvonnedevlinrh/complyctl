// SPDX-License-Identifier: Apache-2.0
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hashicorp/go-hclog"

	"github.com/complytime/complyctl/internal/complytime"
)

const (
	// PluginDir is the subdirectory name for ampel artifacts within the workspace.
	PluginDir = "ampel"
	// DefaultPolicyDir is the default directory name for granular AMPEL policy source files.
	DefaultPolicyDir = "policy"
	// GeneratedPolicyDir is the workspace subdirectory for generated policy artifacts.
	GeneratedPolicyDir = "policy"
	// DefaultResultsDir is the default directory name for per-repository result files.
	DefaultResultsDir = "results"
	// DefaultTargetsFile is the default filename for the target repository configuration.
	DefaultTargetsFile = "ampel-targets.yaml"
)

// Config holds the plugin configuration with hardcoded workspace-relative paths.
type Config struct{}

// NewConfig returns a new Config.
func NewConfig() *Config {
	return &Config{}
}

// ampelDir returns the path to the ampel subdirectory within the workspace.
func ampelDir() string {
	return filepath.Join(complytime.WorkspaceDir, PluginDir)
}

// EnsureDirectories creates the workspace directory structure required by the plugin.
func EnsureDirectories() error {
	directories := []string{
		ampelDir(),
		PolicyDirPath(),
		ResultsDirPath(),
		SpecDirPath(),
	}
	for _, dir := range directories {
		if err := os.MkdirAll(dir, 0750); err != nil {
			return fmt.Errorf("creating directory %s: %w", dir, err)
		}
		hclog.Default().Debug("ensured directory", "path", dir)
	}
	return nil
}

// PolicyDirPath returns the path for the granular policy source directory.
func PolicyDirPath() string {
	return filepath.Join(ampelDir(), DefaultPolicyDir)
}

// ResultsDirPath returns the path for the per-repository result files.
func ResultsDirPath() string {
	return filepath.Join(ampelDir(), DefaultResultsDir)
}

// TargetsFilePath returns the path for the targets configuration file.
func TargetsFilePath() string {
	return filepath.Join(ampelDir(), DefaultTargetsFile)
}

// GeneratedPolicyDirPath returns the workspace path for generated policy output.
func GeneratedPolicyDirPath() string {
	return filepath.Join(ampelDir(), GeneratedPolicyDir)
}

// SpecDirPath returns the path for the embedded spec files directory.
func SpecDirPath() string {
	return filepath.Join(ampelDir(), "specs")
}
