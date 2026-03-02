package main

import (
	"context"
	"errors"
	"log"

	"github.com/viralforge/mesh/services/data-ai/M25-auto-clipping-ai/internal/app/bootstrap"
)

func main() {
	runtime, err := bootstrap.NewRuntime(context.Background(), "configs/default.yaml")
	if err != nil {
		log.Fatalf("bootstrap runtime: %v", err)
	}
	log.Printf("%s worker initialized", runtime.String())
	if err := runtime.RunWorker(context.Background()); err != nil && !errors.Is(err, context.Canceled) {
		log.Fatalf("run worker: %v", err)
	}
}
