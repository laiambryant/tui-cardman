package main

import (
	"log/slog"
	"os"

	"gihtub.com/laiambryant/tui-cardman/internal/config"
)

func main() {
	config.LoadConfig()
	log, err := os.Create("output.log")
	if err != nil {
		panic(err)
	}
	defer log.Close()
	slog.SetDefault(slog.New(slog.NewTextHandler(log, &slog.HandlerOptions{})))
}
