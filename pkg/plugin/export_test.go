// SPDX-License-Identifier: Apache-2.0

package plugin

// RegisterPluginForTest allows test code to register a plugin directly
// into the Manager's plugin map. This is only available during testing.
func (m *Manager) RegisterPluginForTest(evaluatorID string, p *LoadedPlugin) {
	m.plugins[evaluatorID] = p
}
