// SPDX-License-Identifier: Apache-2.0

package output

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gemaraproj/go-gemara"

	"github.com/complytime/complyctl/internal/complytime"
)

// Markdown generates a human-readable compliance report in Markdown format.
type Markdown struct {
	policyID      string
	evalLog       *gemara.EvaluationLog
	embedLog      bool
	evaluationLog string
}

func NewMarkdown(policyID string, evalLog *gemara.EvaluationLog) *Markdown {
	return &Markdown{
		policyID: policyID,
		evalLog:  evalLog,
		embedLog: false,
	}
}

func (m *Markdown) SetEmbedEvaluationLog(path string) {
	m.embedLog = true
	m.evaluationLog = path
}

func (m *Markdown) Write(outDir string) (string, error) {
	var sb strings.Builder

	fmt.Fprintf(&sb, "# Compliance Scan Report: %s\n\n", m.policyID)
	fmt.Fprintf(&sb, "**Generated**: %s\n\n", time.Now().Format(time.RFC3339))
	sb.WriteString("---\n\n")

	for _, ce := range m.evalLog.Evaluations {
		fmt.Fprintf(&sb, "## Control: %s\n\n", ce.Name)
		fmt.Fprintf(&sb, "- **Result**: %s\n", ce.Result.String())
		fmt.Fprintf(&sb, "- **Message**: %s\n\n", ce.Message)

		for _, al := range ce.AssessmentLogs {
			fmt.Fprintf(&sb, "### %s\n", al.Requirement.EntryId)
			fmt.Fprintf(&sb, "- **Confidence**: %s\n", al.ConfidenceLevel.String())
			fmt.Fprintf(&sb, "- **Result**: %s\n", al.Result.String())
			fmt.Fprintf(&sb, "- **Message**: %s\n", al.Message)
			fmt.Fprintf(&sb, "- **Steps Executed**: %d\n", al.StepsExecuted)
			sb.WriteString("\n")
		}
	}

	if m.embedLog && m.evaluationLog != "" {
		data, err := os.ReadFile(m.evaluationLog)
		if err == nil {
			sb.WriteString("---\n\n## Evaluation Log\n\n```yaml\n")
			sb.WriteString(string(data))
			sb.WriteString("\n```\n")
		}
	}

	if err := os.MkdirAll(outDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create output directory: %w", err)
	}

	filename := fmt.Sprintf("report-%s-%s.md",
		complytime.FilenameSafe(m.policyID), time.Now().Format("20060102-150405"))
	path := filepath.Join(outDir, filename)
	if err := os.WriteFile(path, []byte(sb.String()), 0600); err != nil {
		return "", fmt.Errorf("failed to write markdown report: %w", err)
	}

	return path, nil
}
