package main

import (
	"log"
	"net/http"
	"os"
	"strings"

	httpadapter "github.com/viralforge/mesh/services/trust-compliance/M68-retention-service/internal/adapters/http"
	"github.com/viralforge/mesh/services/trust-compliance/M68-retention-service/internal/adapters/postgres"
	"github.com/viralforge/mesh/services/trust-compliance/M68-retention-service/internal/application"
)

func main() {
	repos := postgres.NewRepositories()
	svc := application.NewService(application.Dependencies{
		Policies:     repos.Policies,
		Previews:     repos.Previews,
		Holds:        repos.Holds,
		Restorations: repos.Restorations,
		Deletions:    repos.Deletions,
		Audit:        repos.Audit,
		Idempotency:  repos.Idempotency,
	})
	router := httpadapter.NewRouter(httpadapter.NewHandler(svc))

	addr := strings.TrimSpace(os.Getenv("PORT"))
	if addr == "" {
		addr = "8080"
	}
	if !strings.HasPrefix(addr, ":") {
		addr = ":" + addr
	}

	log.Printf("M68-Retention-Service listening on %s", addr)
	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatal(err)
	}
}
