package main

import (
	"context"
	"log"

	"github.com/viralforge/mesh/services/trust-compliance/M36-risk-service/internal/app/bootstrap"
)

func main() {
	runtime, err := bootstrap.NewRuntime(context.Background(), "configs/default.yaml")
	if err != nil {
		log.Fatalf("bootstrap runtime: %v", err)
	}
	if err := runtime.RunAPI(context.Background()); err != nil {
		log.Fatalf("run api: %v", err)
	}
}
