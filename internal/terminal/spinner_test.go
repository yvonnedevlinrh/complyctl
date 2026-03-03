// SPDX-License-Identifier: Apache-2.0

package terminal

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSpinnerModelView(t *testing.T) {
	m := spinnerModel{}
	m.spinner.Spinner.Frames = []string{"⠋"}
	m.message = "loading..."

	view := m.View()
	assert.True(t, strings.Contains(view, "loading..."),
		"expected spinner view to contain message, got: %q", view)
}

func TestSpinnerModelStopMsg(t *testing.T) {
	m := spinnerModel{}
	m.message = "test"

	updated, cmd := m.Update(stopMsg{})
	require.NotNil(t, updated)
	require.NotNil(t, cmd)

	msg := cmd()
	_, isQuit := msg.(tea.QuitMsg)
	assert.True(t, isQuit, "expected tea.Quit command on stopMsg")
}

func TestSpinnerModelTickAdvancesFrame(t *testing.T) {
	m := spinnerModel{}
	m.spinner.Spinner.Frames = []string{"A", "B", "C"}
	m.message = "working"

	view1 := m.View()
	updated, _ := m.Update(m.spinner.Tick())
	view2 := updated.(spinnerModel).View()

	assert.Contains(t, view1, "working")
	assert.Contains(t, view2, "working")
}

func TestNewSpinnerWriter(t *testing.T) {
	s := NewSpinnerWriter("test msg", nil)
	require.NotNil(t, s)
	require.NotNil(t, s.program)
}
