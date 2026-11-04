package main

import (
	"context"
	"fmt"
	"os"

	"ssh-keyflow/internal/app"
)

var version = "dev"

func main() {
	if len(os.Args) == 2 && os.Args[1] == "--version" {
		fmt.Fprintln(os.Stdout, "SSH KeyFlow", version)
		return
	}
	if len(os.Args) != 1 {
		fmt.Fprintln(os.Stderr, "Usage: ssh-keyflow [--version]")
		os.Exit(2)
	}
	home, err := os.UserHomeDir()
	if err == nil {
		err = app.Run(context.Background(), app.Options{Home: home})
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
