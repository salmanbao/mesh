package main

import (
	"context"
	"log"

	"github.com/viralforge/mesh/services/data-ai/M54-analytics-service/internal/app/bootstrap"
)

func main() {
	runtime, err := bootstrap.NewRuntime(context.Background(), "configs/default.yaml")
	if err != nil {
		log.Fatalf("bootstrap runtime: %v", err)
	}
	if err := runtime.RunWorker(context.Background()); err != nil {
		log.Fatalf("run worker: %v", err)
	}
}
