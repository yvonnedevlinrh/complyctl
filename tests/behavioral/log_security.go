// SPDX-License-Identifier: Apache-2.0

package behavioral

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gemaraproj/go-gemara"
)

// LogCredentialRedaction verifies the log file does not contain plaintext
// values of target variables resolved from environment references (CTRL09.AR01).
func LogCredentialRedaction(payload any) (gemara.Result, string, gemara.ConfidenceLevel) {
	ctx, result, msg, conf := verifyContext(payload)
	if ctx == nil {
		return result, msg, conf
	}

	secretValue := "SUPER_SECRET_TOKEN_12345"
	ctx.Env = append(ctx.Env, "TEST_CREDENTIAL="+secretValue)

	configYAML := fmt.Sprintf(`policies:
  - url: %s/%s
    id: %s
targets:
  - id: log-test-target
    policies:
      - %s
    variables:
      auth_token: ${TEST_CREDENTIAL}
`, ctx.RegistryURL, ctx.PolicyID, ctx.PolicyID, ctx.PolicyID)

	if err := os.WriteFile(filepath.Join(ctx.WorkDir, "complytime.yaml"), []byte(configYAML), 0600); err != nil {
		return gemara.Unknown, "failed to write config: " + err.Error(), gemara.Undetermined
	}

	_, _ = ctx.RunBinary("get")
	_, _ = ctx.RunBinary("generate", "--policy-id", ctx.PolicyID)

	logPath := filepath.Join(ctx.WorkDir, "complytime.log")
	logData, err := os.ReadFile(logPath)
	if err != nil {
		return gemara.Unknown, "no log file found at " + logPath, gemara.Undetermined
	}

	if strings.Contains(string(logData), secretValue) {
		return gemara.Failed, "log file contains plaintext credential value", gemara.High
	}

	return gemara.Passed, "log file does not contain plaintext credential", gemara.High
}
