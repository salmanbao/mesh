package main

import (
	"context"
	"log"

	"github.com/viralforge/mesh/services/integrations/M66-embed-service/internal/app/bootstrap"
)

func main() {
	r, err := bootstrap.NewRuntime(context.Background(), "configs/default.yaml")
	if err != nil {
		log.Fatal(err)
	}
	if err := r.RunAPI(context.Background()); err != nil {
		log.Fatal(err)
	}
}
