// SPDX-License-Identifier: Apache-2.0

package registry

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
)

// MockFetcher provides an in-memory mock registry for testing
type MockFetcher struct {
	mu       sync.RWMutex
	policies map[string]map[string]*MockPolicy
}

// MockPolicy represents a mock policy version
type MockPolicy struct {
	Digest   string
	Version  string
	Manifest []byte
}

// NewMockFetcher creates a mock fetcher with optional seed data
func NewMockFetcher() *MockFetcher {
	return &MockFetcher{
		policies: make(map[string]map[string]*MockPolicy),
	}
}

// AddPolicy registers a policy version for the mock
func (m *MockFetcher) AddPolicy(modulePath, version, digest string, manifest []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.policies[modulePath] == nil {
		m.policies[modulePath] = make(map[string]*MockPolicy)
	}
	m.policies[modulePath][version] = &MockPolicy{
		Digest:   digest,
		Version:  version,
		Manifest: manifest,
	}
	m.policies[modulePath]["latest"] = &MockPolicy{
		Digest:   digest,
		Version:  version,
		Manifest: manifest,
	}
}

// DefinitionVersion returns digest and version for modulePath
func (m *MockFetcher) DefinitionVersion(ctx context.Context, modulePath string) (string, string, error) {
	_ = ctx
	m.mu.RLock()
	defer m.mu.RUnlock()

	versions, ok := m.policies[modulePath]
	if !ok {
		return "", "", fmt.Errorf("policy %s not found", modulePath)
	}
	p := versions["latest"]
	if p == nil {
		return "", "", fmt.Errorf("policy %s has no latest version", modulePath)
	}
	return p.Digest, p.Version, nil
}

// SeedTestPolicy adds default test policy for integration tests
func (m *MockFetcher) SeedTestPolicy(modulePath string) {
	manifest := map[string]interface{}{
		"policy_id": modulePath,
		"version":   "v1.0.0",
		"layers":    []string{"controls", "guidelines", "assessments"},
	}
	data, _ := json.Marshal(manifest)
	m.AddPolicy(modulePath, "v1.0.0", "sha256:abc123", data)
}
