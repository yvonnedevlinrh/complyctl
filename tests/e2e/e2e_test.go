//go:build e2e

package e2e

import (
	"os/exec"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestComplyctlHelp(t *testing.T) {
	// Run the "complyctl --help" command
	cmd := exec.Command("complyctl", "--help")
	output, err := cmd.CombinedOutput()

	// Ensure there is no error when running the command
	if err != nil {
		t.Fatalf("Error running complyctl --help: %v\nOutput: %s", err, string(output))
	}

	// Convert the output to a string and check if expected text is present
	outputStr := string(output)

	// Assert that "Usage" or the expected help message is part of the output
	// Use a table-driven test for clear, maintainable assertions
	expectedSubstrings := []string{
		"Usage:",
		"Aliases:",
		"Available Commands:",
		"Flags:",
		"complyctl [command]",
	}

	for _, expected := range expectedSubstrings {
		t.Run("Contains "+expected, func(t *testing.T) {
			assert.True(t, strings.Contains(outputStr, expected), "Help output should contain '%s'", expected)
		})
	}
}

func TestComplyctlList(t *testing.T) {
	// Run the "complyctl list" command
	cmd := exec.Command("complyctl", "list", "--plain")
	output, err := cmd.CombinedOutput()

	// Ensure there is no error when running the command
	if err != nil {
		t.Fatalf("Error running complyctl list: %v\nOutput: %s", err, string(output))
	}

	// Convert the output to a string and check if expected content is returned
	outputStr := string(output)

	// Check if the output contains expected text
	assert.True(t, len(outputStr) > 0, "Output from 'complyctl list' should not be empty")
	assert.True(t, strings.Contains(outputStr, "cusp_fedora"), "The output should contain 'cusp_fedora'")
}
