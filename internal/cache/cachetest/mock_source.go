// SPDX-License-Identifier: Apache-2.0

// Package cachetest provides test doubles for the cache package.
// This package is intended for use in tests only.
package cachetest

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/opencontainers/go-digest"
	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	ocistore "oras.land/oras-go/v2/content/oci"
)

// MockPolicySource provides an in-memory mock for testing sync operations.
// Implements cache.PolicySource by pushing OCI content directly into the
// destination store.
type MockPolicySource struct {
	mu            sync.RWMutex
	policies      map[string]*mockPolicyData
	LastLookupRef string
}

type mockPolicyData struct {
	digest  string
	version string
}

// NewMockPolicySource creates a mock policy source for testing
func NewMockPolicySource() *MockPolicySource {
	return &MockPolicySource{
		policies: make(map[string]*mockPolicyData),
	}
}

// SeedPolicy adds a mock policy for testing. The policy is registered under
// both the bare policyID and the versioned key (policyID:version) so that
// lookups with an explicit version tag also resolve.
func (m *MockPolicySource) SeedPolicy(policyID, version, digestStr string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	data := &mockPolicyData{
		digest:  digestStr,
		version: version,
	}
	m.policies[policyID] = data
	m.policies[policyID+":"+version] = data
}

// DefinitionVersion returns digest and version for a mock policy.
// The lookupRef is recorded in LastLookupRef for test assertions.
func (m *MockPolicySource) DefinitionVersion(_ context.Context, lookupRef string) (string, string, error) {
	m.mu.Lock()
	m.LastLookupRef = lookupRef
	m.mu.Unlock()

	m.mu.RLock()
	defer m.mu.RUnlock()
	p, ok := m.policies[lookupRef]
	if !ok {
		return "", "", fmt.Errorf("policy %s not found", lookupRef)
	}
	return p.digest, p.version, nil
}

// CopyPolicy pushes a minimal OCI manifest into the destination store for testing.
// Simulates what oras.Copy() does in production without needing a remote registry.
func (m *MockPolicySource) CopyPolicy(ctx context.Context, policyID, tag string, dst *ocistore.Store) (ocispec.Descriptor, error) {
	m.mu.RLock()
	_, ok := m.policies[policyID]
	m.mu.RUnlock()
	if !ok {
		return ocispec.Descriptor{}, fmt.Errorf("policy %s not found in mock source", policyID)
	}

	configData := []byte("{}")
	configDesc := ocispec.Descriptor{
		MediaType: ocispec.MediaTypeEmptyJSON,
		Digest:    digest.FromBytes(configData),
		Size:      int64(len(configData)),
	}
	if err := dst.Push(ctx, configDesc, bytes.NewReader(configData)); err != nil {
		if !isDuplicateErr(err) {
			return ocispec.Descriptor{}, fmt.Errorf("failed to push complytime: %w", err)
		}
	}

	layerData := []byte(fmt.Sprintf(`{"policy_id": %q, "type": "test-layer"}`, policyID))
	layerDesc := ocispec.Descriptor{
		MediaType: "application/vnd.gemara.policy.v1+yaml",
		Digest:    digest.FromBytes(layerData),
		Size:      int64(len(layerData)),
		Annotations: map[string]string{
			ocispec.AnnotationTitle: "policies",
		},
	}
	if err := dst.Push(ctx, layerDesc, bytes.NewReader(layerData)); err != nil {
		if !isDuplicateErr(err) {
			return ocispec.Descriptor{}, fmt.Errorf("failed to push layer: %w", err)
		}
	}

	manifest := ocispec.Manifest{
		MediaType: ocispec.MediaTypeImageManifest,
		Config:    configDesc,
		Layers:    []ocispec.Descriptor{layerDesc},
	}
	manifestData, err := json.Marshal(manifest)
	if err != nil {
		return ocispec.Descriptor{}, fmt.Errorf("failed to marshal manifest: %w", err)
	}

	manifestDesc := ocispec.Descriptor{
		MediaType: ocispec.MediaTypeImageManifest,
		Digest:    digest.FromBytes(manifestData),
		Size:      int64(len(manifestData)),
	}
	if err := dst.Push(ctx, manifestDesc, bytes.NewReader(manifestData)); err != nil {
		if !isDuplicateErr(err) {
			return ocispec.Descriptor{}, fmt.Errorf("failed to push manifest: %w", err)
		}
	}

	if err := dst.Tag(ctx, manifestDesc, tag); err != nil {
		return ocispec.Descriptor{}, fmt.Errorf("failed to tag manifest: %w", err)
	}

	return manifestDesc, nil
}

func isDuplicateErr(err error) bool {
	if err == nil {
		return false
	}
	return err.Error() == "already exists"
}
