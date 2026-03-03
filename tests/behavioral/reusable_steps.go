// SPDX-License-Identifier: Apache-2.0

package behavioral

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gemaraproj/go-gemara"
)

// WriteConfig writes a standard complytime.yaml to the context's WorkDir.
func WriteConfig(payload any) (gemara.Result, string, gemara.ConfidenceLevel) {
	ctx, result, msg, conf := verifyContext(payload)
	if ctx == nil {
		return result, msg, conf
	}

	configYAML := fmt.Sprintf(`policies:
  - url: %s/%s
    id: %s
targets:
  - id: e2e-target
    policies:
      - %s
    variables:
      env: test
`, ctx.RegistryURL, ctx.PolicyID, ctx.PolicyID, ctx.PolicyID)

	path := filepath.Join(ctx.WorkDir, "complytime.yaml")
	if err := os.WriteFile(path, []byte(configYAML), 0600); err != nil {
		return gemara.Unknown, "failed to write config: " + err.Error(), gemara.Undetermined
	}
	return gemara.Passed, "workspace config written", gemara.High
}

// SyncPolicy runs `complyctl get` to pull the policy from the registry.
func SyncPolicy(payload any) (gemara.Result, string, gemara.ConfidenceLevel) {
	ctx, result, msg, conf := verifyContext(payload)
	if ctx == nil {
		return result, msg, conf
	}

	output, err := ctx.RunBinary("get")
	if err != nil {
		return gemara.Failed, "get failed: " + output, gemara.High
	}
	if strings.Contains(output, "Synchronization completed.") {
		return gemara.Passed, "policy synced", gemara.High
	}
	return gemara.Passed, "get succeeded: " + output, gemara.High
}

// InstallTestPlugin copies the test plugin binary into the plugin directory.
func InstallTestPlugin(payload any) (gemara.Result, string, gemara.ConfidenceLevel) {
	ctx, result, msg, conf := verifyContext(payload)
	if ctx == nil {
		return result, msg, conf
	}

	if ctx.TestPluginBinary == "" {
		return gemara.Unknown, "test plugin binary path not set", gemara.Undetermined
	}
	if _, err := os.Stat(ctx.TestPluginBinary); err != nil {
		return gemara.Unknown,
			fmt.Sprintf("test plugin not found at %s — run 'make build-test-plugin' first", ctx.TestPluginBinary),
			gemara.Undetermined
	}

	pluginDir := filepath.Join(ctx.HomeDir, ".complytime", "providers")
	if err := os.MkdirAll(pluginDir, 0755); err != nil {
		return gemara.Unknown, "failed to create plugin dir: " + err.Error(), gemara.Undetermined
	}

	data, err := os.ReadFile(ctx.TestPluginBinary)
	if err != nil {
		return gemara.Unknown, "failed to read test plugin: " + err.Error(), gemara.Undetermined
	}

	dst := filepath.Join(pluginDir, "complyctl-provider-test")
	if err := os.WriteFile(dst, data, 0700); err != nil { //nolint:gosec // G306 - plugin binary needs execute permission
		return gemara.Unknown, "failed to install test plugin: " + err.Error(), gemara.Undetermined
	}
	return gemara.Passed, "test plugin installed", gemara.High
}
