// SPDX-License-Identifier: Apache-2.0

package behavioral

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/gemaraproj/go-gemara"

	"github.com/complytime/complyctl/internal/complytime"
)

// EvaluationLogProduced verifies that scan produces a Gemara evaluation
// log YAML file in the .complytime/scan output directory (CTRL06.AR01).
func EvaluationLogProduced(payload any) (gemara.Result, string, gemara.ConfidenceLevel) {
	ctx, result, msg, conf := verifyContext(payload)
	if ctx == nil {
		return result, msg, conf
	}

	output, err := ctx.RunBinary("scan", "--policy-id", ctx.PolicyID)
	if err != nil {
		return gemara.Failed, "scan failed: " + output, gemara.High
	}

	outDir := filepath.Join(ctx.WorkDir, complytime.WorkspaceDir, complytime.ScanOutputDir)
	evalLog := findFile(outDir, "evaluation-log-", ".yaml")
	if evalLog == "" {
		return gemara.Failed, "no evaluation-log-*.yaml found in " + outDir, gemara.High
	}

	data, err := os.ReadFile(evalLog)
	if err != nil {
		return gemara.Failed, "could not read evaluation log: " + err.Error(), gemara.High
	}

	if len(data) == 0 {
		return gemara.Failed, "evaluation log is empty", gemara.High
	}

	return gemara.Passed, "scan produced evaluation log", gemara.High
}

// OSCALResultProduced verifies that scan with --format oscal produces
// an OSCAL assessment result JSON file (CTRL06.AR02).
func OSCALResultProduced(payload any) (gemara.Result, string, gemara.ConfidenceLevel) {
	ctx, result, msg, conf := verifyContext(payload)
	if ctx == nil {
		return result, msg, conf
	}

	output, err := ctx.RunBinary("scan", "--policy-id", ctx.PolicyID, "--format", "oscal")
	if err != nil {
		return gemara.Failed, "scan --format oscal failed: " + output, gemara.High
	}

	outDir := filepath.Join(ctx.WorkDir, complytime.WorkspaceDir, complytime.ScanOutputDir)
	oscalFile := findFile(outDir, "assessment-results-", ".json")
	if oscalFile == "" {
		return gemara.Failed, "no assessment-results-*.json found in " + outDir, gemara.High
	}

	data, err := os.ReadFile(oscalFile)
	if err != nil {
		return gemara.Failed, "could not read OSCAL result: " + err.Error(), gemara.High
	}

	if strings.Contains(string(data), "assessment-results") || strings.Contains(string(data), "results") {
		return gemara.Passed, "scan produced OSCAL assessment result", gemara.High
	}
	return gemara.Failed, "OSCAL file missing expected structure", gemara.High
}

func findFile(dir, prefix, suffix string) string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return ""
	}
	for _, e := range entries {
		name := e.Name()
		if strings.HasPrefix(name, prefix) && strings.HasSuffix(name, suffix) {
			return filepath.Join(dir, name)
		}
	}
	return ""
}
