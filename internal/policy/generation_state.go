// SPDX-License-Identifier: Apache-2.0

package policy

import (
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"time"

	"github.com/complytime/complyctl/internal/complytime"
)

// GenerationState tracks the policy cache digest at generation time for
// freshness detection. Persisted per policy at
// {workspace}/{WorkspaceDir}/generation/{policy-id}.json
// See R37: specs/001-gemara-native-workflow/research.md
type GenerationState struct {
	PolicyID          string            `json:"policy_id"`
	PolicyDigest      string            `json:"policy_digest"`
	ComplypackDigests map[string]string `json:"complypack_digests,omitempty"`
	GeneratedAt       string            `json:"generated_at"`
	EvaluatorIDs      []string          `json:"evaluator_ids"`
}

// SaveGenerationState persists a GenerationState to the generation directory.
// Creates the full directory path, including any subdirectories from nested
// policy IDs (e.g. "policies/cis-fedora-l1-workstation").
func SaveGenerationState(baseDir, policyID string, state *GenerationState) error {
	path := filepath.Join(baseDir, complytime.WorkspaceDir, "generation", policyID+".json")
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return fmt.Errorf("failed to create generation state directory: %w", err)
	}

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal generation state: %w", err)
	}

	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("failed to write generation state: %w", err)
	}
	return nil
}

// LoadGenerationState reads a persisted GenerationState for the given policy.
// Returns nil (no error) when no state file exists.
func LoadGenerationState(baseDir, policyID string) (*GenerationState, error) {
	path := filepath.Join(baseDir, complytime.WorkspaceDir, "generation", policyID+".json")

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to read generation state: %w", err)
	}

	var state GenerationState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to parse generation state: %w", err)
	}
	return &state, nil
}

// IsFresh returns true when the persisted policy digest and complypack digests
// both match their current values from the cache.
func (s *GenerationState) IsFresh(currentDigest string, currentComplypackDigests map[string]string) bool {
	if s.PolicyDigest != currentDigest {
		return false
	}
	return maps.Equal(normalizeNilMap(s.ComplypackDigests), normalizeNilMap(currentComplypackDigests))
}

func normalizeNilMap(m map[string]string) map[string]string {
	if m == nil {
		return map[string]string{}
	}
	return m
}

// NewGenerationState creates a GenerationState with the current timestamp.
func NewGenerationState(policyID, digest string, evaluatorIDs []string, complypackDigests map[string]string) *GenerationState {
	return &GenerationState{
		PolicyID:          policyID,
		PolicyDigest:      digest,
		ComplypackDigests: complypackDigests,
		GeneratedAt:       time.Now().UTC().Format(time.RFC3339),
		EvaluatorIDs:      evaluatorIDs,
	}
}
