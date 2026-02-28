package main

import (
	"log/slog"
	"os"
	"time"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger.With("service", "M51-Data-Portability-Service"))
	slog.Info("worker started", "mode", "phase0-foundation", "note", "background export pipelines are not enabled in this baseline")
	time.Sleep(100 * time.Millisecond)
}
