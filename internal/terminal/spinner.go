// SPDX-License-Identifier: Apache-2.0

package terminal

import (
	"fmt"
	"io"
	"os"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
)

type stopMsg struct{}

type spinnerModel struct {
	spinner spinner.Model
	message string
}

func (m spinnerModel) Init() tea.Cmd {
	return m.spinner.Tick
}

func (m spinnerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if _, ok := msg.(stopMsg); ok {
		return m, tea.Quit
	}
	var cmd tea.Cmd
	m.spinner, cmd = m.spinner.Update(msg)
	return m, cmd
}

func (m spinnerModel) View() string {
	return fmt.Sprintf("%s %s", m.spinner.View(), m.message)
}

// Spinner renders an animated braille spinner using charmbracelet/bubbles.
// Start() launches the animation; Stop() halts it and cleans up.
type Spinner struct {
	program *tea.Program
}

func NewSpinner(message string) *Spinner {
	return NewSpinnerWriter(message, os.Stderr)
}

// NewSpinnerWriter creates a spinner that writes to w instead of stderr.
func NewSpinnerWriter(message string, w io.Writer) *Spinner {
	s := spinner.New()
	s.Spinner = spinner.MiniDot
	m := spinnerModel{spinner: s, message: message}
	p := tea.NewProgram(m,
		tea.WithOutput(w),
		tea.WithInput(nil),
		tea.WithoutSignalHandler(),
	)
	return &Spinner{program: p}
}

func (s *Spinner) Start() {
	go func() {
		_, _ = s.program.Run()
	}()
}

func (s *Spinner) Stop() {
	s.program.Send(stopMsg{})
	s.program.Wait()
}
