// SPDX-License-Identifier: Apache-2.0

package registry_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/complytime/complyctl/internal/registry"
)

func TestClient_DefinitionVersion_WithMockFetcher(t *testing.T) {
	mock := registry.NewMockFetcher()
	mock.SeedTestPolicy("test-policy")

	client := registry.NewClientWithFetcher("mock-registry", nil, mock)

	digest, version, err := client.DefinitionVersion(context.Background(), "test-policy")
	require.NoError(t, err)
	assert.NotEmpty(t, digest)
	assert.NotEmpty(t, version)
}

func TestClient_DefinitionVersion_EmptyPath(t *testing.T) {
	client := registry.NewClientWithFetcher("mock-registry", nil, registry.NewMockFetcher())

	_, _, err := client.DefinitionVersion(context.Background(), "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "module path cannot be empty")
}

func TestClient_DefinitionVersion_NotFound(t *testing.T) {
	mock := registry.NewMockFetcher()
	client := registry.NewClientWithFetcher("mock-registry", nil, mock)

	_, _, err := client.DefinitionVersion(context.Background(), "nonexistent")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}
