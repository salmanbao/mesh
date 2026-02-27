package main

import (
	"context"
	"log"

	"github.com/viralforge/mesh/services/integrations/M89-affiliate-service/internal/app/bootstrap"
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
