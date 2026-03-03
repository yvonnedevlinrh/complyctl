// SPDX-License-Identifier: Apache-2.0

package main

import (
	"github.com/complytime/complyctl/cmd/openscap-plugin/server"
	"github.com/complytime/complyctl/pkg/plugin"
)

func main() {
	openSCAPPlugin := server.New()
	plugin.Serve(openSCAPPlugin)
}
