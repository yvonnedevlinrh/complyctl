// SPDX-License-Identifier: Apache-2.0

package policy

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/complytime/complyctl/internal/complytime"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- T152: SaveGenerationState + LoadGenerationState round-trip ---

func TestGenerationState_SaveLoadRoundTrip(t *testing.T) {
	baseDir := t.TempDir()
	state := &GenerationState{
		PolicyID:     "nist-800-53-r5",
		PolicyDigest: "sha256:abc123",
		GeneratedAt:  time.Now().UTC().Format(time.RFC3339),
		EvaluatorIDs: []string{"openscap", "kube-eval"},
	}

	err := SaveGenerationState(baseDir, "nist-800-53-r5", state)
	require.NoError(t, err)

	loaded, err := LoadGenerationState(baseDir, "nist-800-53-r5")
	require.NoError(t, err)
	require.NotNil(t, loaded)
	assert.Equal(t, state.PolicyID, loaded.PolicyID)
	assert.Equal(t, state.PolicyDigest, loaded.PolicyDigest)
	assert.Equal(t, state.GeneratedAt, loaded.GeneratedAt)
	assert.Equal(t, state.EvaluatorIDs, loaded.EvaluatorIDs)
}

func TestGenerationState_SaveCreatesNestedDirs(t *testing.T) {
	baseDir := t.TempDir()
	state := &GenerationState{PolicyID: "policies/nested/deep"}

	err := SaveGenerationState(baseDir, "policies/nested/deep", state)
	require.NoError(t, err)

	path := filepath.Join(baseDir, complytime.WorkspaceDir, "generation", "policies/nested/deep.json")
	_, err = os.Stat(path)
	assert.NoError(t, err)
}

func TestLoadGenerationState_MissingFile(t *testing.T) {
	baseDir := t.TempDir()
	loaded, err := LoadGenerationState(baseDir, "nonexistent")
	assert.NoError(t, err)
	assert.Nil(t, loaded)
}

func TestLoadGenerationState_CorruptJSON(t *testing.T) {
	baseDir := t.TempDir()
	dir := filepath.Join(baseDir, complytime.WorkspaceDir, "generation")
	require.NoError(t, os.MkdirAll(dir, 0750))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "corrupt.json"), []byte("{not valid json"), 0600))

	_, err := LoadGenerationState(baseDir, "corrupt")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse generation state")
}

// --- T153: IsFresh tests ---

func TestIsFresh_MatchingDigest(t *testing.T) {
	s := &GenerationState{PolicyDigest: "sha256:abc123"}
	assert.True(t, s.IsFresh("sha256:abc123"))
}

func TestIsFresh_MismatchedDigest(t *testing.T) {
	s := &GenerationState{PolicyDigest: "sha256:abc123"}
	assert.False(t, s.IsFresh("sha256:def456"))
}

func TestIsFresh_EmptyDigest(t *testing.T) {
	s := &GenerationState{PolicyDigest: ""}
	assert.False(t, s.IsFresh("sha256:abc123"))
}

func TestIsFresh_BothEmpty(t *testing.T) {
	s := &GenerationState{PolicyDigest: ""}
	assert.True(t, s.IsFresh(""))
}

// --- T154: NewGenerationState tests ---

func TestNewGenerationState(t *testing.T) {
	evalIDs := []string{"openscap", "kube-eval"}
	state := NewGenerationState("test-policy", "sha256:abc", evalIDs)

	assert.Equal(t, "test-policy", state.PolicyID)
	assert.Equal(t, "sha256:abc", state.PolicyDigest)
	assert.Equal(t, evalIDs, state.EvaluatorIDs)

	_, err := time.Parse(time.RFC3339, state.GeneratedAt)
	assert.NoError(t, err, "GeneratedAt should be valid RFC3339")
}

func TestNewGenerationState_NilEvaluatorIDs(t *testing.T) {
	state := NewGenerationState("test", "sha256:xyz", nil)
	assert.Nil(t, state.EvaluatorIDs)
}
