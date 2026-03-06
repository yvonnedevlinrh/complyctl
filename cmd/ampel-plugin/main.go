package main

import (
	"github.com/complytime/complyctl/cmd/ampel-plugin/server"
	"github.com/complytime/complyctl/pkg/plugin"
)

func main() {
	ampelPlugin := server.New()
	plugin.Serve(ampelPlugin)
}
