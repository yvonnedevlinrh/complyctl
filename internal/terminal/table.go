// SPDX-License-Identifier: Apache-2.0

package terminal

import (
	"fmt"
	"io"
	"os"

	"github.com/charmbracelet/x/term"
)

// IsTTY returns true when stdout is an interactive terminal.
func IsTTY() bool {
	return term.IsTerminal(os.Stdout.Fd())
}

// ShowPlainTable renders a plain text formatted table to writer.
func ShowPlainTable(w io.Writer, headers []string, rows [][]string) {
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h) + 2
	}
	for _, row := range rows {
		for i, cell := range row {
			if i < len(widths) {
				if cw := len(cell) + 2; cw > widths[i] {
					widths[i] = cw
				}
			}
		}
	}
	for i, h := range headers {
		_, _ = fmt.Fprintf(w, "%-*s", widths[i], h)
	}
	_, _ = fmt.Fprintln(w)
	for _, row := range rows {
		for i, cell := range row {
			if i < len(widths) {
				_, _ = fmt.Fprintf(w, "%-*s", widths[i], cell)
			}
		}
		_, _ = fmt.Fprintln(w)
	}
}
