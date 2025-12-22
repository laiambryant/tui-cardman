package main

import (
	"os"

	"gihtub.com/laiambryant/tui-cardman/cmd/command"
)

func main() {
	if err := command.Execute(); err != nil {
		os.Exit(1)
	}
}
