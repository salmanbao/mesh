package main

import (
	"log/slog"
	"os"
	"time"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger.With("service", "M38-Content-License-Verification"))
	slog.Info("worker started", "mode", "phase0-foundation", "note", "no background pipelines configured")
	time.Sleep(100 * time.Millisecond)
}
