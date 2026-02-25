package main

import (
	"context"
	"log"

	"github.com/viralforge/mesh/services/core-platform/M02-profile-service/internal/app/bootstrap"
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
