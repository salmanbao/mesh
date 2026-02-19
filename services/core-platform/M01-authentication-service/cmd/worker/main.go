package main

import (
	"context"
	"log"

	"github.com/viralforge/mesh/services/core-platform/M01-authentication-service/internal/app/bootstrap"
)

func main() {
	ctx := context.Background()
	runtime, err := bootstrap.NewRuntime(ctx, "configs/default.yaml")
	if err != nil {
		log.Fatalf("bootstrap worker runtime: %v", err)
	}
	if err := runtime.RunWorker(ctx); err != nil {
		log.Fatalf("run worker: %v", err)
	}
}
