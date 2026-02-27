package main

import (
	"context"
	"log"

	"github.com/viralforge/mesh/services/financial-rails/M44-resolution-center/internal/app/bootstrap"
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
