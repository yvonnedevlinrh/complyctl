// SPDX-License-Identifier: Apache-2.0

package plugin

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/complytime/complyctl/internal/complytime"
)

// PluginInfo holds the identity and filesystem path of a discovered plugin.
type PluginInfo struct {
	PluginID       string
	EvaluatorID    string
	ExecutablePath string
}

// Discovery scans a directory for plugin executables matching the naming convention.
type Discovery struct {
	pluginDir string
}

func NewDiscovery(pluginDir string) *Discovery {
	return &Discovery{
		pluginDir: pluginDir,
	}
}

// DiscoverPlugins scans the plugin directory for executables matching the
// naming convention and derives evaluator IDs from filenames.
func (d *Discovery) DiscoverPlugins() ([]PluginInfo, error) {
	expandedDir := expandPath(d.pluginDir)

	entries, err := os.ReadDir(expandedDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []PluginInfo{}, nil
		}
		return nil, fmt.Errorf("failed to read plugin directory: %w", err)
	}

	plugins := []PluginInfo{}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if !strings.HasPrefix(entry.Name(), complytime.PluginExecutablePrefix) {
			continue
		}

		executablePath := filepath.Join(expandedDir, entry.Name())

		info, err := entry.Info()
		if err != nil {
			continue
		}

		if info.Mode()&0111 == 0 {
			continue
		}

		pluginID := strings.TrimPrefix(entry.Name(), complytime.PluginExecutablePrefix)

		plugins = append(plugins, PluginInfo{
			PluginID:       pluginID,
			EvaluatorID:    pluginID,
			ExecutablePath: executablePath,
		})
	}

	return plugins, nil
}

func expandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(homeDir, path[2:])
	}
	return path
}
