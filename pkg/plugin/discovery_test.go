// SPDX-License-Identifier: Apache-2.0

package plugin

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/complytime/complyctl/internal/complytime"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- T157: DiscoverPlugins tests ---

func TestDiscoverPlugins_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	d := NewDiscovery(dir)
	plugins, err := d.DiscoverPlugins()
	require.NoError(t, err)
	assert.Empty(t, plugins)
}

func TestDiscoverPlugins_NonPrefixedExecutables(t *testing.T) {
	dir := t.TempDir()
	createExecutable(t, dir, "some-other-tool")

	d := NewDiscovery(dir)
	plugins, err := d.DiscoverPlugins()
	require.NoError(t, err)
	assert.Empty(t, plugins)
}

func TestDiscoverPlugins_ValidPlugin(t *testing.T) {
	dir := t.TempDir()
	createExecutable(t, dir, complytime.PluginExecutablePrefix+"mock")

	d := NewDiscovery(dir)
	plugins, err := d.DiscoverPlugins()
	require.NoError(t, err)
	require.Len(t, plugins, 1)
	assert.Equal(t, "mock", plugins[0].EvaluatorID)
	assert.Equal(t, filepath.Join(dir, complytime.PluginExecutablePrefix+"mock"), plugins[0].ExecutablePath)
}

// --- T158: scanDir tests ---

func TestScanDir_NonexistentDirectory(t *testing.T) {
	plugins, err := scanDir("/nonexistent/path/that/does/not/exist")
	assert.NoError(t, err)
	assert.Nil(t, plugins)
}

func TestScanDir_NonExecutableFile(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("permission bits not meaningful on Windows")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, complytime.PluginExecutablePrefix+"noexec")
	require.NoError(t, os.WriteFile(path, []byte("#!/bin/sh"), 0644)) // #nosec

	plugins, err := scanDir(dir)
	require.NoError(t, err)
	assert.Empty(t, plugins)
}

func TestScanDir_DirectoryEntriesSkipped(t *testing.T) {
	dir := t.TempDir()
	subdir := filepath.Join(dir, complytime.PluginExecutablePrefix+"subdir")
	require.NoError(t, os.Mkdir(subdir, 0755)) // #nosec

	plugins, err := scanDir(dir)
	require.NoError(t, err)
	assert.Empty(t, plugins)
}

func TestScanDir_MultipleValidExecutables(t *testing.T) {
	dir := t.TempDir()
	createExecutable(t, dir, complytime.PluginExecutablePrefix+"openscap")
	createExecutable(t, dir, complytime.PluginExecutablePrefix+"kube-eval")
	createExecutable(t, dir, complytime.PluginExecutablePrefix+"trivy")

	plugins, err := scanDir(dir)
	require.NoError(t, err)
	require.Len(t, plugins, 3)

	ids := make(map[string]bool)
	for _, p := range plugins {
		ids[p.EvaluatorID] = true
	}
	assert.True(t, ids["openscap"])
	assert.True(t, ids["kube-eval"])
	assert.True(t, ids["trivy"])
}

// --- T159: expandPath tests ---

func TestExpandPath_TildePrefix(t *testing.T) {
	home, err := os.UserHomeDir()
	require.NoError(t, err)

	result := expandPath("~/foo")
	assert.Equal(t, filepath.Join(home, "foo"), result)
}

func TestExpandPath_AbsolutePath(t *testing.T) {
	assert.Equal(t, "/absolute/path", expandPath("/absolute/path"))
}

func TestExpandPath_RelativePath(t *testing.T) {
	assert.Equal(t, "relative/path", expandPath("relative/path"))
}

// --- T160: user-dir precedence test ---

func TestScanDir_UserDirPrecedence(t *testing.T) {
	userDir := t.TempDir()
	sysDir := t.TempDir()
	createExecutable(t, userDir, complytime.PluginExecutablePrefix+"openscap")
	createExecutable(t, sysDir, complytime.PluginExecutablePrefix+"openscap")
	createExecutable(t, sysDir, complytime.PluginExecutablePrefix+"sys-only")

	seen := make(map[string]bool)
	var plugins []PluginInfo

	userPlugins, err := scanDir(userDir)
	require.NoError(t, err)
	for _, p := range userPlugins {
		seen[p.EvaluatorID] = true
		plugins = append(plugins, p)
	}

	sysPlugins, err := scanDir(sysDir)
	require.NoError(t, err)
	for _, p := range sysPlugins {
		if !seen[p.EvaluatorID] {
			plugins = append(plugins, p)
		}
	}

	require.Len(t, plugins, 2)

	idPaths := make(map[string]string)
	for _, p := range plugins {
		idPaths[p.EvaluatorID] = p.ExecutablePath
	}

	assert.Equal(t, filepath.Join(userDir, complytime.PluginExecutablePrefix+"openscap"), idPaths["openscap"],
		"user-dir openscap should take precedence over system-dir")
	assert.Equal(t, filepath.Join(sysDir, complytime.PluginExecutablePrefix+"sys-only"), idPaths["sys-only"])
}

func createExecutable(t *testing.T, dir, name string) {
	t.Helper()
	path := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(path, []byte("#!/bin/sh\n"), 0755)) // #nosec
}
