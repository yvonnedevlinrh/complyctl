// SPDX-License-Identifier: Apache-2.0

package cache

import (
	"context"
	"fmt"

	ocispec "github.com/opencontainers/image-spec/specs-go/v1"
	"oras.land/oras-go/v2"
	ocistore "oras.land/oras-go/v2/content/oci"

	"github.com/complytime/complyctl/internal/registry"
)

// PolicySource abstracts remote policy access for sync operations.
type PolicySource interface {
	DefinitionVersion(ctx context.Context, policyID string) (digest string, version string, err error)
	CopyPolicy(ctx context.Context, policyID, tag string, dst *ocistore.Store) (ocispec.Descriptor, error)
}

// RegistrySource wraps a registry.Client to implement PolicySource.
// Uses oras.Copy() for atomic remote-to-local transfer with digest verification.
type RegistrySource struct {
	client *registry.Client
}

func NewRegistrySource(client *registry.Client) *RegistrySource {
	return &RegistrySource{client: client}
}

func (s *RegistrySource) DefinitionVersion(ctx context.Context, policyID string) (string, string, error) {
	return s.client.DefinitionVersion(ctx, policyID)
}

func (s *RegistrySource) CopyPolicy(ctx context.Context, policyID, tag string, dst *ocistore.Store) (ocispec.Descriptor, error) {
	repo, err := s.client.NewRemoteRepository(ctx, policyID)
	if err != nil {
		return ocispec.Descriptor{}, fmt.Errorf("failed to create remote repository: %w", err)
	}
	return oras.Copy(ctx, repo, tag, dst, tag, oras.CopyOptions{})
}
