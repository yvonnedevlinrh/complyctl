// SPDX-License-Identifier: Apache-2.0

package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/complytime/complyctl/internal/complytime"
)

// State tracks sync metadata for all cached policies, persisted as state.json.
type State struct {
	LastSync time.Time              `json:"last_sync"`
	Policies map[string]PolicyState `json:"policies"`
}

// PolicyState holds version, digest, and timestamp for a single cached policy.
type PolicyState struct {
	Version     string    `json:"version"`
	Digest      string    `json:"digest"`
	LastUpdated time.Time `json:"last_updated"`
}

func LoadState(cacheDir string) (*State, error) {
	statePath := filepath.Join(cacheDir, complytime.StateFileName)

	data, err := os.ReadFile(statePath)
	if err != nil {
		if os.IsNotExist(err) {
			return &State{
				LastSync: time.Time{},
				Policies: make(map[string]PolicyState),
			}, nil
		}
		return nil, fmt.Errorf("failed to read state file %s: %w", statePath, err)
	}

	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("failed to parse state file %s: %w", statePath, err)
	}

	if state.Policies == nil {
		state.Policies = make(map[string]PolicyState)
	}

	return &state, nil
}

func SaveState(state *State, cacheDir string) error {
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	statePath := filepath.Join(cacheDir, complytime.StateFileName)

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	if err := os.WriteFile(statePath, data, 0600); err != nil {
		return fmt.Errorf("failed to write state file %s: %w", statePath, err)
	}

	return nil
}

func (s *State) UpdatePolicyState(policyID, version, digest string) {
	if s.Policies == nil {
		s.Policies = make(map[string]PolicyState)
	}
	s.Policies[policyID] = PolicyState{
		Version:     version,
		Digest:      digest,
		LastUpdated: time.Now(),
	}
	s.LastSync = time.Now()
}

func (s *State) GetPolicyState(policyID string) (PolicyState, bool) {
	if s.Policies == nil {
		return PolicyState{}, false
	}
	state, exists := s.Policies[policyID]
	return state, exists
}
