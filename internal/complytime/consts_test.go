// SPDX-License-Identifier: Apache-2.0

package complytime

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWithWorkspaceVar_NilMap(t *testing.T) {
	result := WithWorkspaceVar(nil, "/home/user/project")
	assert.Equal(t, map[string]string{WorkspaceVarKey: "/home/user/project"}, result)
}

func TestWithWorkspaceVar_EmptyMap(t *testing.T) {
	result := WithWorkspaceVar(map[string]string{}, "/home/user/project")
	assert.Equal(t, map[string]string{WorkspaceVarKey: "/home/user/project"}, result)
}

func TestWithWorkspaceVar_PreservesExistingVars(t *testing.T) {
	input := map[string]string{"output_dir": "/tmp", "scan_target": "host"}
	result := WithWorkspaceVar(input, "/workspace")

	assert.Equal(t, "/workspace", result[WorkspaceVarKey])
	assert.Equal(t, "/tmp", result["output_dir"])
	assert.Equal(t, "host", result["scan_target"])
	assert.Len(t, result, 3)
}

func TestWithWorkspaceVar_OverridesUserDefined(t *testing.T) {
	input := map[string]string{WorkspaceVarKey: "."}
	result := WithWorkspaceVar(input, "/resolved/path")

	assert.Equal(t, "/resolved/path", result[WorkspaceVarKey])
}

func TestWithWorkspaceVar_DoesNotMutateOriginal(t *testing.T) {
	input := map[string]string{"key": "value"}
	_ = WithWorkspaceVar(input, "/workspace")

	_, hasWorkspace := input[WorkspaceVarKey]
	assert.False(t, hasWorkspace, "original map must not be mutated")
	assert.Len(t, input, 1)
}
