package main

import (
	"context"
	"log"

	"github.com/viralforge/mesh/services/financial-rails/M41-reward-engine/internal/app/bootstrap"
)

func main() {
	ctx := context.Background()
	runtime, err := bootstrap.NewRuntime(ctx, "configs/default.yaml")
	if err != nil {
		log.Fatalf("bootstrap api runtime: %v", err)
	}
	if err := runtime.RunAPI(ctx); err != nil {
		log.Fatalf("run api: %v", err)
	}
}
