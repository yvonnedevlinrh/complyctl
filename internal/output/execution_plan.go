// SPDX-License-Identifier: Apache-2.0

package output

import (
	"fmt"
	"strings"

	"github.com/complytime/complyctl/internal/complytime"
)

// ExecutionPlanRow describes one target-provider combination in the plan.
type ExecutionPlanRow struct {
	TargetID         string
	ProviderID       string
	RequirementCount int
	Status           string
}

// FormatExecutionPlan produces a structured plain-text execution plan.
// See FR-007, Session 2026-02-26e: specs/001-gemara-native-workflow/spec.md
func FormatExecutionPlan(effectiveID string, rows []ExecutionPlanRow) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Execution Plan: %s\n", effectiveID)
	for _, r := range rows {
		emoji := statusEmoji(r.Status)
		fmt.Fprintf(&b, "\n  Target: %s\n", r.TargetID)
		fmt.Fprintf(&b, "    Provider: %s (%s %s)\n", r.ProviderID, emoji, r.Status)
		fmt.Fprintf(&b, "    Requirements: %d\n", r.RequirementCount)
	}
	fmt.Fprintln(&b, "\nGeneration completed.")
	return b.String()
}

func statusEmoji(status string) string {
	switch status {
	case "healthy":
		return complytime.StatusPassed
	default:
		return complytime.StatusFailed
	}
}

// FormatPreScanSummary produces a brief one-line summary for normal scan mode.
// See FR-034: specs/001-gemara-native-workflow/spec.md
func FormatPreScanSummary(requirementCount int, providerIDs []string, targetIDs []string) string {
	providers := strings.Join(providerIDs, ", ")
	targets := strings.Join(targetIDs, ", ")
	return fmt.Sprintf("Scanning %d requirements via %s for target(s): %s...",
		requirementCount, providers, targets)
}
