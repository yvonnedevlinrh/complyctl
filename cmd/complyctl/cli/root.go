// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"os"
	"path/filepath"
	"sync"

	"github.com/hashicorp/go-hclog"
	"github.com/spf13/cobra"

	"github.com/complytime/complyctl/internal/complytime"
	"github.com/complytime/complyctl/pkg/log"
)

var logger hclog.Logger

// lazyLogWriter defers log file creation until something actually writes to it.
// See R57: log lives at {WorkspaceDir}/{LogFileName} (.complytime/complyctl.log).
type lazyLogWriter struct {
	once sync.Once
	file *os.File
}

func (w *lazyLogWriter) Write(p []byte) (int, error) {
	w.once.Do(func() {
		logDir := complytime.WorkspaceDir
		if err := os.MkdirAll(logDir, 0750); err != nil {
			return
		}
		logPath := filepath.Join(logDir, complytime.LogFileName)
		f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
		if err != nil {
			return
		}
		w.file = f
	})
	if w.file == nil {
		return len(p), nil
	}
	return w.file.Write(p)
}

func init() {
	lw := &lazyLogWriter{}
	logger = log.NewLogger(lw)
}

func Error(msg string) {
	logger.Error(msg)
}

func enableDebug(opts *Common) {
	if opts.Debug {
		logger.SetLevel(hclog.Debug)
	}
}

func New() *cobra.Command {

	cmd := &cobra.Command{
		Use:           "complyctl [command]",
		SilenceErrors: true,
		SilenceUsage:  true,
	}

	opts := Common{
		Output: Output{
			Out:    cmd.OutOrStdout(),
			ErrOut: cmd.ErrOrStderr(),
		},
	}
	opts.BindFlags(cmd.PersistentFlags())

	cmd.AddCommand(
		versionCmd(&opts),
		initCmd(&opts),
		getCmd(&opts),
		scanCmd(&opts),
		generateCmd(&opts),
		listCmd(&opts),
		providersCmd(&opts),
		doctorCmd(&opts),
	)
	cmd.PersistentPreRun = func(_ *cobra.Command, _ []string) {
		enableDebug(&opts)
	}

	return cmd
}
