package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/viralforge/mesh/services/platform-ops/M83-cdn-management-service/internal/app/bootstrap"
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
