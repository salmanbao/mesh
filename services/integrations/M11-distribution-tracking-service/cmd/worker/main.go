package main

import (
	"context"
	"log"

	"github.com/viralforge/mesh/services/integrations/M11-distribution-tracking-service/internal/app/bootstrap"
)

func main() {
	r, err := bootstrap.NewRuntime(context.Background(), "configs/default.yaml")
	if err != nil {
		log.Fatal(err)
	}
	if err := r.RunWorker(context.Background()); err != nil {
		log.Fatal(err)
	}
}
