// SPDX-License-Identifier: Apache-2.0

package policy

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"

	"github.com/complytime/complyctl/internal/cache"
)

// Loader reads policy artifacts from OCI Layout cache stores.
type Loader struct {
	cacheMgr *cache.Cache
}

func NewLoader(cacheMgr *cache.Cache) *Loader {
	return &Loader{
		cacheMgr: cacheMgr,
	}
}

// ResolveVersion resolves a policy version against the cache. If the requested
// version is empty or "latest", it returns the last cached tag. If the exact
// version exists in cache, it is returned as-is.
func (l *Loader) ResolveVersion(policyID, configVersion string) (string, error) {
	version := configVersion
	if version == "" {
		version = "latest"
	}

	if l.PolicyExists(policyID, version) {
		return version, nil
	}

	if version == "latest" {
		cached, _ := l.GetCachedVersions(policyID)
		if len(cached) > 0 {
			return cached[len(cached)-1], nil
		}
	}

	cached, _ := l.GetCachedVersions(policyID)
	return "", fmt.Errorf(
		"policy %s@%s not in cache (run 'complyctl get' first); cached: %v",
		policyID, version, cached,
	)
}

// LoadLayerByMediaType loads a specific Gemara layer from the policy's OCI manifest
// by matching the layer descriptor's media type.
func (l *Loader) LoadLayerByMediaType(policyID, version, mediaType string) ([]byte, error) {
	if policyID == "" {
		return nil, fmt.Errorf("policy ID cannot be empty")
	}
	if version == "" {
		return nil, fmt.Errorf("version cannot be empty")
	}
	if mediaType == "" {
		return nil, fmt.Errorf("media type cannot be empty")
	}

	store, err := l.cacheMgr.NewPolicyStore(policyID)
	if err != nil {
		return nil, fmt.Errorf("policy not found in cache: %s: %w", policyID, err)
	}

	ctx := context.Background()

	manifestDesc, err := store.Resolve(ctx, version)
	if err != nil {
		return nil, fmt.Errorf("policy %s@%s not in cache: %w", policyID, version, err)
	}

	rc, err := store.Fetch(ctx, manifestDesc)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch manifest for %s@%s: %w", policyID, version, err)
	}
	manifestData, err := io.ReadAll(rc)
	rc.Close()
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest: %w", err)
	}

	var manifest ocispec.Manifest
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse manifest JSON: %w", err)
	}

	for _, layer := range manifest.Layers {
		if layer.MediaType == mediaType {
			layerRC, err := store.Fetch(ctx, layer)
			if err != nil {
				return nil, fmt.Errorf("failed to fetch layer %s: %w", mediaType, err)
			}
			data, err := io.ReadAll(layerRC)
			layerRC.Close()
			if err != nil {
				return nil, fmt.Errorf("failed to read layer %s: %w", mediaType, err)
			}
			return data, nil
		}
	}

	return nil, fmt.Errorf(
		"layer with media type %s not found for policy %s@%s",
		mediaType, policyID, version,
	)
}

func (l *Loader) PolicyExists(policyID, version string) bool {
	if policyID == "" || version == "" {
		return false
	}

	store, err := l.cacheMgr.NewPolicyStore(policyID)
	if err != nil {
		return false
	}

	ctx := context.Background()
	_, err = store.Resolve(ctx, version)
	return err == nil
}

func (l *Loader) GetCachedVersions(policyID string) ([]string, error) {
	if !l.cacheMgr.PolicyStoreExists(policyID) {
		return []string{}, nil
	}

	store, err := l.cacheMgr.NewPolicyStore(policyID)
	if err != nil {
		return nil, fmt.Errorf("failed to open store for policy %s: %w", policyID, err)
	}

	ctx := context.Background()
	var versions []string
	err = store.Tags(ctx, "", func(tags []string) error {
		versions = append(versions, tags...)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list tags for policy %s: %w", policyID, err)
	}

	return versions, nil
}

func (l *Loader) ListCachedPolicies() (map[string][]string, error) {
	policyIDs, err := l.cacheMgr.ListPolicies()
	if err != nil {
		return nil, fmt.Errorf("failed to list cached policies: %w", err)
	}

	result := make(map[string][]string)
	for _, policyID := range policyIDs {
		versions, err := l.GetCachedVersions(policyID)
		if err != nil {
			return nil, fmt.Errorf("failed to get versions for policy %s: %w", policyID, err)
		}
		if len(versions) > 0 {
			result[policyID] = versions
		}
	}
	return result, nil
}
