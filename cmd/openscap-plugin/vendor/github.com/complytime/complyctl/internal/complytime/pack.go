// SPDX-License-Identifier: Apache-2.0

package complytime

import (
	"fmt"
	"os"

	"github.com/goccy/go-yaml"
)

// PackManifest declares what a comply-pack contains. Owned by the pack
// developer, immutable after build, ships in the OCI pack artifact.
// Each policy carries its own OCI URL, enabling multi-registry packs.
type PackManifest struct {
	SchemaVersion      int                 `yaml:"schema-version,omitempty"`
	ID                 string              `yaml:"id"`
	Version            string              `yaml:"version"`
	Description        string              `yaml:"description,omitempty"`
	Platform           *PlatformConfig     `yaml:"platform,omitempty"`
	Policies           []PackPolicyEntry   `yaml:"policies"`
	Providers          []PackProviderEntry `yaml:"providers"`
	SystemDependencies []SystemDependency  `yaml:"system-dependencies,omitempty"`
}

// PlatformConfig identifies the target OS and optional datastream path.
type PlatformConfig struct {
	OS         string `yaml:"os"`
	Datastream string `yaml:"datastream,omitempty"`
}

// PackPolicyEntry declares a policy bundled in the pack. URL is the full
// OCI reference (registry + repo + version). ID is a shortname for the
// policy. Policies from different registries can coexist in one pack.
type PackPolicyEntry struct {
	URL     string `yaml:"url"`
	ID      string `yaml:"id"`
	Profile string `yaml:"profile,omitempty"`
	Catalog string `yaml:"catalog,omitempty"`
	Source  string `yaml:"source,omitempty"`
}

// PackProviderEntry identifies a provider binary bundled in the pack.
type PackProviderEntry struct {
	ID     string `yaml:"id"`
	Binary string `yaml:"binary"`
	Source string `yaml:"source,omitempty"`
}

// DependencyCheckKind identifies the strategy used to verify a system
// dependency is installed. Each kind maps to a hardcoded check in
// doctor — pack authors declare *what* to check, complyctl controls *how*.
type DependencyCheckKind string

const (
	CheckBinary DependencyCheckKind = "binary"
	CheckRPM    DependencyCheckKind = "rpm"
	CheckDEB    DependencyCheckKind = "deb"
	CheckPath   DependencyCheckKind = "path"
)

var validCheckKinds = map[DependencyCheckKind]bool{
	CheckBinary: true,
	CheckRPM:    true,
	CheckDEB:    true,
	CheckPath:   true,
}

// SystemDependency is an OS package required at runtime. Doctor uses
// the Kind + Value pair to run a safe, hardcoded verification check.
// Install is documentation-only remediation guidance (never executed).
type SystemDependency struct {
	Name    string              `yaml:"name"`
	Kind    DependencyCheckKind `yaml:"kind"`
	Value   string              `yaml:"value"`
	Install string              `yaml:"install,omitempty"`
}

// LoadPackManifest reads and parses a pack manifest from the given path.
func LoadPackManifest(path string) (*PackManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("pack manifest not found: %s", path)
		}
		return nil, fmt.Errorf("failed to read pack manifest %s: %w", path, err)
	}

	var manifest PackManifest
	if err := yaml.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("corrupted pack manifest %s: invalid YAML: %w", path, err)
	}

	return &manifest, nil
}

// PackManifestExists reports whether a pack manifest file is present
// in the current working directory.
func PackManifestExists() bool {
	_, err := os.Stat(PackManifestFile)
	return err == nil
}

// ValidatePackManifest checks required fields and uniqueness constraints
// on a parsed pack manifest.
func ValidatePackManifest(m *PackManifest) error {
	if m.SchemaVersion != 0 && m.SchemaVersion != CurrentPackSchemaVersion {
		return fmt.Errorf(
			"pack manifest: unsupported schema-version %d (expected %d)",
			m.SchemaVersion, CurrentPackSchemaVersion,
		)
	}

	if m.ID == "" {
		return fmt.Errorf("pack manifest: id is required")
	}
	if m.Version == "" {
		return fmt.Errorf("pack manifest: version is required")
	}
	if len(m.Policies) == 0 {
		return fmt.Errorf("pack manifest: at least one policy is required")
	}
	if len(m.Providers) == 0 {
		return fmt.Errorf("pack manifest: at least one provider is required")
	}

	policyIDs := make(map[string]bool)
	policyURLs := make(map[string]bool)
	for _, p := range m.Policies {
		if p.URL == "" {
			return fmt.Errorf("pack manifest: policies[].url cannot be empty")
		}
		if p.ID == "" {
			return fmt.Errorf("pack manifest: policies[].id cannot be empty")
		}
		if policyURLs[p.URL] {
			return fmt.Errorf("pack manifest: duplicate policy url %s", p.URL)
		}
		policyURLs[p.URL] = true
		if policyIDs[p.ID] {
			return fmt.Errorf("pack manifest: duplicate policy %s", p.ID)
		}
		policyIDs[p.ID] = true
	}

	providerIDs := make(map[string]bool)
	for _, prov := range m.Providers {
		if prov.ID == "" {
			return fmt.Errorf("pack manifest: providers[].id cannot be empty")
		}
		if prov.Binary == "" {
			return fmt.Errorf("pack manifest: providers[%s].binary is required", prov.ID)
		}
		if providerIDs[prov.ID] {
			return fmt.Errorf("pack manifest: duplicate provider %s", prov.ID)
		}
		providerIDs[prov.ID] = true
	}

	for _, dep := range m.SystemDependencies {
		if dep.Name == "" {
			return fmt.Errorf("pack manifest: system-dependencies[].name cannot be empty")
		}
		if dep.Kind == "" {
			return fmt.Errorf("pack manifest: system-dependencies[%s].kind is required", dep.Name)
		}
		if !validCheckKinds[dep.Kind] {
			return fmt.Errorf(
				"pack manifest: system-dependencies[%s].kind %q is not valid (use: binary, rpm, deb, path)",
				dep.Name, dep.Kind,
			)
		}
		if dep.Value == "" {
			return fmt.Errorf("pack manifest: system-dependencies[%s].value is required", dep.Name)
		}
	}

	if m.Platform != nil && m.Platform.OS == "" {
		return fmt.Errorf("pack manifest: platform.os is required when platform is specified")
	}

	return nil
}

// ToPolicyEntry converts a pack policy entry to a workspace PolicyEntry,
// carrying only the fields relevant to the consumer's complytime.yaml.
func (p PackPolicyEntry) ToPolicyEntry() PolicyEntry {
	return PolicyEntry{
		URL: p.URL,
		ID:  p.ID,
	}
}

// PackToPolicyEntries converts all pack policies to workspace PolicyEntry
// values for use in generated example configs.
func PackToPolicyEntries(m *PackManifest) []PolicyEntry {
	entries := make([]PolicyEntry, len(m.Policies))
	for i, p := range m.Policies {
		entries[i] = p.ToPolicyEntry()
	}
	return entries
}

// PackPolicyIDs returns the policy IDs declared in the pack manifest.
func PackPolicyIDs(m *PackManifest) []string {
	ids := make([]string, len(m.Policies))
	for i, p := range m.Policies {
		ids[i] = p.ID
	}
	return ids
}
