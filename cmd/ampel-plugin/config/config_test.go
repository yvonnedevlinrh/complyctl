package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/complytime/complyctl/internal/complytime"
)

func TestNewConfig(t *testing.T) {
	c := NewConfig()
	require.NotNil(t, c)
}

func TestPolicyDirPath(t *testing.T) {
	expected := filepath.Join(complytime.WorkspaceDir, PluginDir, DefaultPolicyDir)
	require.Equal(t, expected, PolicyDirPath())
}

func TestResultsDirPath(t *testing.T) {
	expected := filepath.Join(complytime.WorkspaceDir, PluginDir, DefaultResultsDir)
	require.Equal(t, expected, ResultsDirPath())
}

func TestTargetsFilePath(t *testing.T) {
	expected := filepath.Join(complytime.WorkspaceDir, PluginDir, DefaultTargetsFile)
	require.Equal(t, expected, TargetsFilePath())
}

func TestGeneratedPolicyDirPath(t *testing.T) {
	expected := filepath.Join(complytime.WorkspaceDir, PluginDir, GeneratedPolicyDir)
	require.Equal(t, expected, GeneratedPolicyDirPath())
}

func TestSpecDirPath(t *testing.T) {
	expected := filepath.Join(complytime.WorkspaceDir, PluginDir, "specs")
	require.Equal(t, expected, SpecDirPath())
}

func TestEnsureDirectories(t *testing.T) {
	dir := t.TempDir()
	origWd, err := os.Getwd()
	require.NoError(t, err)
	require.NoError(t, os.Chdir(dir))
	t.Cleanup(func() { _ = os.Chdir(origWd) })

	err = EnsureDirectories()
	require.NoError(t, err)

	require.DirExists(t, filepath.Join(dir, complytime.WorkspaceDir, PluginDir))
	require.DirExists(t, filepath.Join(dir, complytime.WorkspaceDir, PluginDir, DefaultPolicyDir))
	require.DirExists(t, filepath.Join(dir, complytime.WorkspaceDir, PluginDir, DefaultResultsDir))
	require.DirExists(t, filepath.Join(dir, complytime.WorkspaceDir, PluginDir, "specs"))
}
