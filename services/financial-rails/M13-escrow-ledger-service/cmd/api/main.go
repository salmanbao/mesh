package main

import (
	"context"
	"log"

	"github.com/viralforge/mesh/services/financial-rails/M13-escrow-ledger-service/internal/app/bootstrap"
)

func main() {
	r, err := bootstrap.NewRuntime(context.Background(), "configs/default.yaml")
	if err != nil {
		log.Fatalf("bootstrap runtime: %v", err)
	}
	if err := r.RunAPI(context.Background()); err != nil {
		log.Fatalf("run api: %v", err)
	}
}
