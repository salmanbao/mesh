package main

import (
	"context"
	"log"

	"github.com/viralforge/mesh/services/core-platform/M97-team-service/internal/app/bootstrap"
)

func main() {
	r, err := bootstrap.NewRuntime(context.Background(), "configs/default.yaml")
	if err != nil {
		log.Fatalf("bootstrap runtime: %v", err)
	}
	if err := r.RunWorker(context.Background()); err != nil {
		log.Fatalf("run worker: %v", err)
	}
}
