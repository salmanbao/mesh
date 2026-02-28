package main

import (
	"log"
	"net/http"
	"os"
	"strings"

	httpadapter "github.com/viralforge/mesh/services/data-ai/M57-ai-service/internal/adapters/http"
	"github.com/viralforge/mesh/services/data-ai/M57-ai-service/internal/adapters/postgres"
	"github.com/viralforge/mesh/services/data-ai/M57-ai-service/internal/application"
)

func main() {
	repos := postgres.NewRepositories()
	svc := application.NewService(application.Dependencies{
		Predictions: repos.Predictions,
		BatchJobs:   repos.BatchJobs,
		Models:      repos.Models,
		Feedback:    repos.Feedback,
		Audit:       repos.Audit,
		Idempotency: repos.Idempotency,
	})
	router := httpadapter.NewRouter(httpadapter.NewHandler(svc))

	addr := strings.TrimSpace(os.Getenv("PORT"))
	if addr == "" {
		addr = "8080"
	}
	if !strings.HasPrefix(addr, ":") {
		addr = ":" + addr
	}

	log.Printf("M57-AI-Service listening on %s", addr)
	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatal(err)
	}
}
