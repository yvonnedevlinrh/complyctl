// SPDX-License-Identifier: Apache-2.0

package complytime

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveWorkspaceDir_FlagPrecedence(t *testing.T) {
	flagDir := t.TempDir()
	envDir := t.TempDir()
	t.Setenv(WorkspaceEnvVar, envDir)

	result, err := ResolveWorkspaceDir(flagDir)

	require.NoError(t, err)
	assert.Equal(t, flagDir, result)
}

func TestResolveWorkspaceDir_EnvVarFallback(t *testing.T) {
	envDir := t.TempDir()
	t.Setenv(WorkspaceEnvVar, envDir)

	result, err := ResolveWorkspaceDir("")

	require.NoError(t, err)
	assert.Equal(t, envDir, result)
}

func TestResolveWorkspaceDir_DefaultToCwd(t *testing.T) {
	t.Setenv(WorkspaceEnvVar, "")

	result, err := ResolveWorkspaceDir("")

	require.NoError(t, err)
	cwd, _ := os.Getwd()
	assert.Equal(t, cwd, result)
}

func TestResolveWorkspaceDir_TildeExpansion(t *testing.T) {
	home, err := os.UserHomeDir()
	require.NoError(t, err)

	result, err := ResolveWorkspaceDir("~")

	require.NoError(t, err)
	assert.Equal(t, home, result)
	assert.NotContains(t, result, "~")
}

func TestResolveWorkspaceDir_TildeSubpathExpansion(t *testing.T) {
	home, err := os.UserHomeDir()
	require.NoError(t, err)
	subdir := filepath.Join(home, "complytime-test-tilde-subpath")
	require.NoError(t, os.MkdirAll(subdir, 0o755))
	t.Cleanup(func() { os.RemoveAll(subdir) })

	result, err := ResolveWorkspaceDir("~/complytime-test-tilde-subpath")

	require.NoError(t, err)
	assert.Equal(t, subdir, result)
	assert.NotContains(t, result, "~")
}

func TestResolveWorkspaceDir_RelativeToAbsolute(t *testing.T) {
	tmpDir := t.TempDir()
	subdir := filepath.Join(tmpDir, "subdir")
	err := os.Mkdir(subdir, 0o755)
	require.NoError(t, err)

	cwd, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(cwd) })

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	result, err := ResolveWorkspaceDir("./subdir")

	require.NoError(t, err)
	assert.True(t, filepath.IsAbs(result))
	assert.Equal(t, subdir, result)
}

func TestResolveWorkspaceDir_InvalidPath(t *testing.T) {
	result, err := ResolveWorkspaceDir("/nonexistent/path/xyz123")

	require.Error(t, err)
	assert.Empty(t, result)
	assert.Contains(t, err.Error(), "does not exist")
}

func TestResolveWorkspaceDir_FileNotDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "file.txt")
	err := os.WriteFile(tmpFile, []byte("test"), 0o644) // #nosec G306 — test file
	require.NoError(t, err)

	result, err := ResolveWorkspaceDir(tmpFile)

	require.Error(t, err)
	assert.Empty(t, result)
	assert.Contains(t, err.Error(), "not a directory")
}

func TestDetectConfigPath_NewLocation(t *testing.T) {
	baseDir := t.TempDir()
	newConfigDir := filepath.Join(baseDir, WorkspaceDir)
	err := os.MkdirAll(newConfigDir, 0o750)
	require.NoError(t, err)
	newConfigPath := filepath.Join(newConfigDir, WorkspaceConfigFile)
	err = os.WriteFile(newConfigPath, []byte("version: 1"), 0o600)
	require.NoError(t, err)

	configPath, isLegacy, err := DetectConfigPath(baseDir)

	require.NoError(t, err)
	assert.Equal(t, newConfigPath, configPath)
	assert.False(t, isLegacy)
}

func TestDetectConfigPath_LegacyLocation(t *testing.T) {
	baseDir := t.TempDir()
	legacyConfigPath := filepath.Join(baseDir, WorkspaceConfigFile)
	err := os.WriteFile(legacyConfigPath, []byte("version: 1"), 0o600)
	require.NoError(t, err)

	configPath, isLegacy, err := DetectConfigPath(baseDir)

	require.NoError(t, err)
	assert.Equal(t, legacyConfigPath, configPath)
	assert.True(t, isLegacy)
}

func TestDetectConfigPath_NewLocationTakesPrecedence(t *testing.T) {
	baseDir := t.TempDir()
	newConfigDir := filepath.Join(baseDir, WorkspaceDir)
	err := os.MkdirAll(newConfigDir, 0o750)
	require.NoError(t, err)
	newConfigPath := filepath.Join(newConfigDir, WorkspaceConfigFile)
	err = os.WriteFile(newConfigPath, []byte("version: 1"), 0o600)
	require.NoError(t, err)
	legacyConfigPath := filepath.Join(baseDir, WorkspaceConfigFile)
	err = os.WriteFile(legacyConfigPath, []byte("version: 1"), 0o600)
	require.NoError(t, err)

	configPath, isLegacy, err := DetectConfigPath(baseDir)

	require.NoError(t, err)
	assert.Equal(t, newConfigPath, configPath)
	assert.False(t, isLegacy)
}

func TestDetectConfigPath_NeitherLocationExists(t *testing.T) {
	baseDir := t.TempDir()

	configPath, isLegacy, err := DetectConfigPath(baseDir)

	require.Error(t, err)
	assert.Empty(t, configPath)
	assert.False(t, isLegacy)
	assert.Contains(t, err.Error(), "config file not found in "+baseDir)
	assert.Contains(t, err.Error(), "checked .complytime/complytime.yaml and complytime.yaml")
}

func TestNewWorkspace_WithBaseDir(t *testing.T) {
	baseDir := t.TempDir()
	newConfigDir := filepath.Join(baseDir, WorkspaceDir)
	err := os.MkdirAll(newConfigDir, 0750)
	require.NoError(t, err)
	newConfigPath := filepath.Join(newConfigDir, WorkspaceConfigFile)
	err = os.WriteFile(newConfigPath, []byte("version: 1\npolicies: []\ntargets: []"), 0600)
	require.NoError(t, err)

	ws := NewWorkspace(baseDir)

	require.NotNil(t, ws)
	assert.Equal(t, baseDir, ws.BaseDir())
	assert.Equal(t, newConfigPath, ws.Path())
}

func TestNewWorkspace_LegacyLocationWarning(t *testing.T) {
	baseDir := t.TempDir()
	legacyConfigPath := filepath.Join(baseDir, WorkspaceConfigFile)
	err := os.WriteFile(legacyConfigPath, []byte("version: 1\npolicies: []\ntargets: []"), 0600)
	require.NoError(t, err)

	// Capture stderr to verify warning is printed
	origStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w
	t.Cleanup(func() { os.Stderr = origStderr })

	ws := NewWorkspace(baseDir)

	w.Close()
	var buf [512]byte
	n, _ := r.Read(buf[:])
	stderr := string(buf[:n])

	require.NotNil(t, ws)
	assert.Equal(t, baseDir, ws.BaseDir())
	assert.Equal(t, legacyConfigPath, ws.Path())
	assert.Contains(t, stderr, "WARNING: complytime.yaml found at repository root (legacy location)")
	assert.Contains(t, stderr, "Please move it to .complytime/complytime.yaml")
}

func TestWorkspace_Load_ValidConfig(t *testing.T) {
	baseDir := t.TempDir()
	configDir := filepath.Join(baseDir, WorkspaceDir)
	require.NoError(t, os.MkdirAll(configDir, 0o750))
	require.NoError(t, os.WriteFile(
		filepath.Join(configDir, WorkspaceConfigFile),
		[]byte("policies:\n  - url: registry.example.com/p@v1\ntargets: []"), 0o600))

	ws := NewWorkspace(baseDir)
	require.NoError(t, ws.Load())
	require.NotNil(t, ws.Config())
	assert.Len(t, ws.Config().Policies, 1)
}

func TestWorkspace_Load_InvalidYAML(t *testing.T) {
	baseDir := t.TempDir()
	configDir := filepath.Join(baseDir, WorkspaceDir)
	require.NoError(t, os.MkdirAll(configDir, 0o750))
	require.NoError(t, os.WriteFile(
		filepath.Join(configDir, WorkspaceConfigFile),
		[]byte("not: [valid: yaml: {{"), 0o600))

	ws := NewWorkspace(baseDir)
	err := ws.Load()
	require.Error(t, err)
}

func TestWorkspace_Save_NilConfig(t *testing.T) {
	baseDir := t.TempDir()
	ws := NewWorkspace(baseDir)
	err := ws.Save()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no configuration to save")
}

func TestWorkspace_Exists_False(t *testing.T) {
	baseDir := t.TempDir()
	ws := NewWorkspace(baseDir)
	assert.False(t, ws.Exists(), "workspace should not exist when config is missing")
}

func TestNewWorkspace_EnsureDir(t *testing.T) {
	baseDir := t.TempDir()
	ws := NewWorkspace(baseDir)
	require.NoError(t, ws.EnsureDir())

	configDir := filepath.Join(baseDir, WorkspaceDir)
	info, err := os.Stat(configDir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestNewWorkspace_ConfigNotFound(t *testing.T) {
	baseDir := t.TempDir()

	ws := NewWorkspace(baseDir)

	require.NotNil(t, ws)
	assert.Equal(t, baseDir, ws.BaseDir())
	expectedPath := filepath.Join(baseDir, WorkspaceDir, WorkspaceConfigFile)
	assert.Equal(t, expectedPath, ws.Path())
}
