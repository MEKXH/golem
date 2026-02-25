package main

import (
	"os"

	"github.com/MEKXH/golem/cmd/golem/commands"
)

func main() {
	if err := commands.NewRootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}
