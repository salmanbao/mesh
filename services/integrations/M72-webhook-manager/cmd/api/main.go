package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/viralforge/mesh/services/integrations/M72-webhook-manager/internal/app/bootstrap"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	rt, err := bootstrap.NewRuntime(ctx, "configs/default.yaml")
	if err != nil {
		log.Fatal(err)
	}
	if err := rt.Run(ctx); err != nil {
		log.Fatal(err)
	}
}
