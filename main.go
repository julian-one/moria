package main

import (
	"log/slog"
	"os"

	"github.com/julian-one/moria/cmd"
)

func main() {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stderr, nil)))

	cmd.Execute()
}
