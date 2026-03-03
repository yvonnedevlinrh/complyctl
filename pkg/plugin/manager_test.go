// SPDX-License-Identifier: Apache-2.0

package plugin_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/complytime/complyctl/pkg/plugin"
)

func TestManager_GetPlugin(t *testing.T) {
	mgr, err := plugin.NewManager("/nonexistent", nil)
	require.NoError(t, err)

	mgr.RegisterPluginForTest("mock", &plugin.LoadedPlugin{
		Info: plugin.PluginInfo{
			PluginID:       "mock",
			EvaluatorID:    "mock",
			ExecutablePath: "(test)",
		},
	})

	p, err := mgr.GetPlugin("mock")
	require.NoError(t, err)
	assert.Equal(t, "mock", p.Info.PluginID)
}

func TestManager_GetPlugin_UnknownID(t *testing.T) {
	mgr, err := plugin.NewManager("/nonexistent", nil)
	require.NoError(t, err)

	mgr.RegisterPluginForTest("mock", &plugin.LoadedPlugin{
		Info: plugin.PluginInfo{
			PluginID:       "mock",
			EvaluatorID:    "mock",
			ExecutablePath: "(test)",
		},
	})

	_, err = mgr.GetPlugin("unknown-evaluator")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "plugin not found for evaluator ID")
}

func TestManager_ListPlugins(t *testing.T) {
	mgr, err := plugin.NewManager("/nonexistent", nil)
	require.NoError(t, err)

	mgr.RegisterPluginForTest("mock", &plugin.LoadedPlugin{
		Info: plugin.PluginInfo{
			PluginID:       "mock",
			EvaluatorID:    "mock",
			ExecutablePath: "(test)",
		},
	})

	plugins := mgr.ListPlugins()
	assert.Len(t, plugins, 1)
	assert.Equal(t, "mock", plugins[0].Info.PluginID)
}

func TestManager_EmptyPluginDir(t *testing.T) {
	tmpDir := t.TempDir()

	mgr, err := plugin.NewManager(tmpDir, nil)
	require.NoError(t, err)
	require.NoError(t, mgr.LoadPlugins())

	plugins := mgr.ListPlugins()
	assert.Empty(t, plugins)
}
