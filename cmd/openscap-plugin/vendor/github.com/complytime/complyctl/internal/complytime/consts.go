// SPDX-License-Identifier: Apache-2.0

package complytime

import (
	"os"
	"path/filepath"
	"strings"
	"time"
)

const stateSubdir = ".complytime"
const pluginSubdir = "providers"

// WorkspaceDir is the workspace-local directory for all complyctl artifacts
// (generation state, scan output). Separate from the global ~/.complytime/ cache.
const WorkspaceDir = ".complytime"

const StateFileName = "state.json"
const PoliciesSubdir = "policies"
const WorkspaceConfigFile = "complytime.yaml"
const PackManifestFile = "complypack.yaml"

const CurrentWorkspaceVersion = 1
const CurrentPackSchemaVersion = 1

const (
	APIMethodGetDefinitions    = "GetDefinitions"
	APIMethodDefinitionVersion = "DefinitionVersion"
)

const (
	OutputFormatOSCAL  = "oscal"
	OutputFormatPretty = "pretty"
	OutputFormatSARIF  = "sarif"
)

const ScanOutputDir = "scan"

// DefaultCommandTimeout is the default deadline for scan and generate operations.
// Individual plugin RPCs may use a shorter per-call timeout as a safety net.
const DefaultCommandTimeout = 5 * time.Minute

const PluginExecutablePrefix = "complyctl-provider-"

// Gemara OCI layer media types for identifying layer content within multi-layer OCI manifests.
const (
	MediaTypeCatalog  = "application/vnd.gemara.catalog.v1+yaml"
	MediaTypeGuidance = "application/vnd.gemara.guidance.v1+yaml"
	MediaTypePolicy   = "application/vnd.gemara.policy.v1+yaml"
)

const OCIEmptyConfig = "application/vnd.oci.empty.v1+json"

// Scan result status emoji indicators for terminal summary table (FR-037).
const (
	StatusPassed  = "✅"
	StatusFailed  = "❌"
	StatusSkipped = "⏭️"
	StatusError   = "⚠️"
)

// FilenameSafe replaces characters unsafe for filenames (e.g., path separators)
// so that policy IDs like "policies/nist-800-53-r5" produce flat filenames.
func FilenameSafe(s string) string {
	return strings.ReplaceAll(s, "/", "-")
}

// ExpandPath resolves a leading ~/ to the user's home directory.
func ExpandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(homeDir, path[2:])
	}
	return path
}

// ResolveCacheDir returns the absolute path to the cache directory.
func ResolveCacheDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, stateSubdir), nil
}

// ResolvePluginDir returns the absolute path to the provider directory.
func ResolvePluginDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(homeDir, stateSubdir, pluginSubdir), nil
}
