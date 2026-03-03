// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"io"

	"github.com/spf13/pflag"
)

type Common struct {
	Debug bool
	Output
}

type Output struct {
	Out    io.Writer
	ErrOut io.Writer
}

func (o *Common) BindFlags(fs *pflag.FlagSet) {
	fs.BoolVarP(&o.Debug, "debug", "d", false, "output debug logs")
}
