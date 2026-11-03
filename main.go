package main

import (
	"os"

	"github.com/julian-one/moria/cmd"
)

func main() {
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
