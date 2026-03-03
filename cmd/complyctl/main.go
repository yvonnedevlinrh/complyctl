// SPDX-License-Identifier: Apache-2.0

package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/complytime/complyctl/cmd/complyctl/cli"
)

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt)
	defer cancel()
	complyctl := cli.New()
	if err := complyctl.ExecuteContext(ctx); err != nil {
		cli.Error(fmt.Sprintf("error running complytime: %v", err))
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
