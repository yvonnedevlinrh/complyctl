// SPDX-License-Identifier: Apache-2.0

package output_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/complytime/complyctl/internal/output"
)

func TestMarkdown_Write(t *testing.T) {
	outDir := t.TempDir()
	log := mockGemaraEvalLog()
	md := output.NewMarkdown("test-policy", log)

	path, err := md.Write(outDir)
	require.NoError(t, err)
	assert.FileExists(t, path)

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	content := string(data)

	assert.Contains(t, content, "# Compliance Scan Report: test-policy")
	assert.Contains(t, content, "## Control: ctrl-1")
	assert.Contains(t, content, "req-1")
	assert.Contains(t, content, "req-2")
}

func TestMarkdown_SkippedDueToTailoring(t *testing.T) {
	outDir := t.TempDir()
	log := mockGemaraEvalLog()
	md := output.NewMarkdown("test-policy", log)

	path, err := md.Write(outDir)
	require.NoError(t, err)

	data, _ := os.ReadFile(path)
	content := string(data)

	assert.True(t, strings.Contains(content, "Not Applicable"),
		"expected 'Not Applicable' in markdown output for skipped assessments")
}

func TestMarkdown_OutputFileNaming(t *testing.T) {
	outDir := t.TempDir()
	log := mockGemaraEvalLog()
	md := output.NewMarkdown("test-policy", log)

	path, err := md.Write(outDir)
	require.NoError(t, err)

	filename := filepath.Base(path)
	assert.Contains(t, filename, "report-test-policy-")
	assert.Contains(t, filename, ".md")
}

func TestMarkdown_EmbedEvaluationLog(t *testing.T) {
	outDir := t.TempDir()
	log := mockGemaraEvalLog()

	evalPath := filepath.Join(outDir, "eval.yaml")
	require.NoError(t, os.WriteFile(evalPath, []byte("test: true\n"), 0600))

	md := output.NewMarkdown("test-policy", log)
	md.SetEmbedEvaluationLog(evalPath)

	path, err := md.Write(outDir)
	require.NoError(t, err)

	data, _ := os.ReadFile(path)
	content := string(data)

	assert.Contains(t, content, "## Evaluation Log")
	assert.Contains(t, content, "test: true")
}
