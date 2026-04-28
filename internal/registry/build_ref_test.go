// SPDX-License-Identifier: Apache-2.0

package registry

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBuildRef_AppendsLatest(t *testing.T) {
	got := buildRef("registry.example.com", "policies/nist")
	assert.Equal(t, "registry.example.com/policies/nist:latest", got)
}

func TestBuildRef_PreservesExplicitTag(t *testing.T) {
	got := buildRef("registry.example.com", "policies/nist:v1.0.0")
	assert.Equal(t, "registry.example.com/policies/nist:v1.0.0", got)
}

func TestBuildRef_PreservesDigestRef(t *testing.T) {
	got := buildRef("registry.example.com", "policies/nist@sha256:abc123")
	assert.Equal(t, "registry.example.com/policies/nist@sha256:abc123", got)
}
