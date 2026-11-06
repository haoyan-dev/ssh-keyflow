package main

import (
	"context"
	"os"

	"ssh-keyflow/internal/cli"
	"ssh-keyflow/internal/tui"
)

var version = "dev"

func main() {
	os.Exit(cli.Run(context.Background(), os.Args[1:], cli.Options{Version: version, RunTUI: tui.Run}))
}
