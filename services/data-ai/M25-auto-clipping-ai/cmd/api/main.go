package main

import (
	"context"
	"log"

	"github.com/viralforge/mesh/services/data-ai/M25-auto-clipping-ai/internal/app/bootstrap"
)

func main() {
	runtime, err := bootstrap.NewRuntime(context.Background(), "configs/default.yaml")
	if err != nil {
		log.Fatalf("bootstrap runtime: %v", err)
	}
	log.Printf("%s API initialized", runtime.String())
	if err := runtime.RunAPI(context.Background()); err != nil {
		log.Fatalf("run api: %v", err)
	}
}
