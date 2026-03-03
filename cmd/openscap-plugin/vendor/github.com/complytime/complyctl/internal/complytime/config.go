// SPDX-License-Identifier: Apache-2.0

package complytime

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/goccy/go-yaml"
)

var envVarPattern = regexp.MustCompile(`\$\{([^}]+)\}`)

// WorkspaceConfig is the top-level YAML configuration for a complytime workspace.
// See R48, R49: three-tier variable model.
//
// Variables holds workspace-scoped constants (e.g., output directory paths).
// These are NOT expanded for ${...} environment references — only
// target-level variables support ${VAR} substitution (for secrets and
// per-target credentials). Place environment-dependent values in
// targets[].variables instead.
type WorkspaceConfig struct {
	Version   int               `yaml:"version,omitempty"`
	Policies  []PolicyEntry     `yaml:"policies"`
	Targets   []TargetConfig    `yaml:"targets"`
	Variables map[string]string `yaml:"variables,omitempty"`
}

// PolicyEntry pairs a full OCI reference with an optional user-chosen shortname.
// If ID is empty, EffectiveID derives one from the last path segment of URL.
type PolicyEntry struct {
	URL string `yaml:"url"`
	ID  string `yaml:"id,omitempty"`
}

// EffectiveID returns the explicit ID if set, otherwise derives one from the
// last path segment of the URL (e.g. "registry.com/policies/nist-r5@v1" → "nist-r5").
func (p PolicyEntry) EffectiveID() string {
	if p.ID != "" {
		return p.ID
	}
	ref := ParsePolicyRef(p.URL)
	segments := strings.Split(ref.Repository, "/")
	return segments[len(segments)-1]
}

// TargetConfig binds a scan target to one or more policies with optional variables.
// Policies are referenced by their effective ID (explicit or derived).
type TargetConfig struct {
	ID        string            `yaml:"id"`
	Policies  []string          `yaml:"policies"`
	Variables map[string]string `yaml:"variables,omitempty"`
}

// PolicyRef represents a parsed OCI policy reference.
type PolicyRef struct {
	Raw        string
	Registry   string
	Repository string
	Version    string
}

// ParsePolicyRef parses a full OCI reference into its components.
// Handles optional scheme (http://, https://), registry host detection,
// and @version suffix.
func ParsePolicyRef(raw string) PolicyRef {
	ref := PolicyRef{Raw: raw}
	s := strings.TrimSpace(raw)

	var scheme string
	if strings.HasPrefix(s, "http://") {
		scheme = "http://"
		s = strings.TrimPrefix(s, "http://")
	} else if strings.HasPrefix(s, "https://") {
		scheme = "https://"
		s = strings.TrimPrefix(s, "https://")
	}

	if idx := strings.LastIndex(s, "@"); idx > 0 && idx < len(s)-1 {
		ref.Version = s[idx+1:]
		s = s[:idx]
	}

	parts := strings.SplitN(s, "/", 2)
	if len(parts) == 2 && (strings.Contains(parts[0], ".") || strings.Contains(parts[0], ":")) {
		ref.Registry = scheme + parts[0]
		ref.Repository = parts[1]
	} else {
		ref.Repository = s
	}

	return ref
}

// FindPolicy matches a policy identifier against the policies list.
// Tries: effective ID, full URL, repository path.
// Returns the matching entry and true if found.
func FindPolicy(policies []PolicyEntry, policyID string) (*PolicyEntry, bool) {
	policyID = strings.TrimSpace(policyID)

	for i, p := range policies {
		if p.EffectiveID() == policyID {
			return &policies[i], true
		}
	}

	for i, p := range policies {
		if p.URL == policyID {
			return &policies[i], true
		}
	}

	for i, p := range policies {
		ref := ParsePolicyRef(p.URL)
		if ref.Repository == policyID {
			return &policies[i], true
		}
	}

	return nil, false
}

// PolicyIDs returns the effective IDs for all policies in the config.
func PolicyIDs(policies []PolicyEntry) map[string]*PolicyEntry {
	m := make(map[string]*PolicyEntry, len(policies))
	for i := range policies {
		m[policies[i].EffectiveID()] = &policies[i]
	}
	return m
}

// Load reads, parses, and validates the complytime configuration from the
// default complytime.yaml path.
func Load() (*WorkspaceConfig, error) {
	return LoadFrom(WorkspaceConfigFile)
}

// LoadFrom reads, parses, and validates the complytime configuration from the
// given path.
func LoadFrom(configPath string) (*WorkspaceConfig, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf(
				"complytime config not found: %s (run 'complyctl init' to create)",
				configPath,
			)
		}
		return nil, fmt.Errorf("failed to read config file %s: %w", configPath, err)
	}

	var config WorkspaceConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf(
			"corrupted complytime file %s: invalid YAML: %w",
			configPath, err,
		)
	}

	if err := resolveEnvVars(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

// resolveEnvVars expands ${VAR} references in target variable values from the
// process environment. Returns an error if a referenced variable is not set.
func resolveEnvVars(config *WorkspaceConfig) error {
	for i, target := range config.Targets {
		for key, val := range target.Variables {
			resolved, err := expandEnvRef(val)
			if err != nil {
				return fmt.Errorf("targets[%s].variables.%s: %w", target.ID, key, err)
			}
			config.Targets[i].Variables[key] = resolved
		}
	}
	return nil
}

// expandEnvRef replaces all ${VAR} occurrences in s with their environment
// values. Returns an error if any referenced variable is unset.
func expandEnvRef(s string) (string, error) {
	var missing []string
	result := envVarPattern.ReplaceAllStringFunc(s, func(match string) string {
		varName := envVarPattern.FindStringSubmatch(match)[1]
		if val, ok := os.LookupEnv(varName); ok {
			return val
		}
		missing = append(missing, varName)
		return match
	})
	if len(missing) > 0 {
		return "", fmt.Errorf("unset environment variable(s): %s", strings.Join(missing, ", "))
	}
	return result, nil
}

// Save writes complytime configuration to the default complytime.yaml path.
func Save(config *WorkspaceConfig) error {
	return SaveTo(config, WorkspaceConfigFile)
}

// SaveTo writes complytime configuration to the given path.
func SaveTo(config *WorkspaceConfig, configPath string) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal workspace config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write config file %s: %w", configPath, err)
	}

	return nil
}

// Validate checks required fields, uniqueness constraints, and policy format.
func Validate(config *WorkspaceConfig) error {
	if config.Version != 0 && config.Version != CurrentWorkspaceVersion {
		return fmt.Errorf(
			"unsupported config version %d (expected %d)",
			config.Version, CurrentWorkspaceVersion,
		)
	}

	if len(config.Policies) == 0 {
		return fmt.Errorf("policies: at least one policy is required")
	}

	seenURL := make(map[string]bool)
	seenID := make(map[string]bool)
	for _, p := range config.Policies {
		if strings.TrimSpace(p.URL) == "" {
			return fmt.Errorf("policies[].url cannot be empty")
		}
		if seenURL[p.URL] {
			return fmt.Errorf("policies: duplicate url %s", p.URL)
		}
		seenURL[p.URL] = true

		eid := p.EffectiveID()
		if seenID[eid] {
			return fmt.Errorf("policies: duplicate id %s", eid)
		}
		seenID[eid] = true
	}

	policyLookup := PolicyIDs(config.Policies)

	targetIDs := make(map[string]bool)
	for _, target := range config.Targets {
		if target.ID == "" {
			return fmt.Errorf("targets[].id cannot be empty")
		}
		if targetIDs[target.ID] {
			return fmt.Errorf("targets[].id: duplicate %s", target.ID)
		}
		targetIDs[target.ID] = true
		if len(target.Policies) == 0 {
			return fmt.Errorf("targets[%s].policies: at least one required", target.ID)
		}
		for _, pid := range target.Policies {
			if _, ok := policyLookup[pid]; !ok {
				return fmt.Errorf("targets[%s]: policy %q not in policies list", target.ID, pid)
			}
		}
	}
	return nil
}

// UniqueRegistries extracts the distinct registry URLs from all policy entries.
func UniqueRegistries(policies []PolicyEntry) []string {
	seen := make(map[string]bool)
	var registries []string
	for _, p := range policies {
		ref := ParsePolicyRef(p.URL)
		if ref.Registry != "" && !seen[ref.Registry] {
			seen[ref.Registry] = true
			registries = append(registries, ref.Registry)
		}
	}
	return registries
}

// ValidateTargetPolicyVersions ensures every target references policies that
// exist in the workspace policies list (by effective ID) with no duplicates.
func ValidateTargetPolicyVersions(config *WorkspaceConfig) error {
	lookup := PolicyIDs(config.Policies)
	for _, target := range config.Targets {
		seen := make(map[string]bool)
		for _, pid := range target.Policies {
			if _, ok := lookup[pid]; !ok {
				return fmt.Errorf("target %s: policy %q not in policies list", target.ID, pid)
			}
			if seen[pid] {
				return fmt.Errorf("target %s: duplicate policy %s", target.ID, pid)
			}
			seen[pid] = true
		}
	}
	return nil
}

// ResolvePolicyForTarget looks up a target's policy reference against the
// workspace policies and returns the parsed OCI reference.
func ResolvePolicyForTarget(policies []PolicyEntry, targetPolicyID string) (*PolicyEntry, PolicyRef, bool) {
	entry, found := FindPolicy(policies, targetPolicyID)
	if !found {
		return nil, PolicyRef{}, false
	}
	return entry, ParsePolicyRef(entry.URL), true
}
