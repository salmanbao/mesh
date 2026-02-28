package main

import (
	"log"
	"net/http"
	"os"
	"strings"

	httpadapter "github.com/viralforge/mesh/services/integrations/M70-developer-portal/internal/adapters/http"
	"github.com/viralforge/mesh/services/integrations/M70-developer-portal/internal/adapters/postgres"
	"github.com/viralforge/mesh/services/integrations/M70-developer-portal/internal/application"
)

func main() {
	repos := postgres.NewRepositories()
	svc := application.NewService(application.Dependencies{
		Developers:  repos.Developers,
		Sessions:    repos.Sessions,
		APIKeys:     repos.APIKeys,
		Rotations:   repos.Rotations,
		Webhooks:    repos.Webhooks,
		Deliveries:  repos.Deliveries,
		Usage:       repos.Usage,
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
	log.Printf("M70-Developer-Portal listening on %s", addr)
	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatal(err)
	}
}
