package main

import (
	"log"
	"net/http"
	"os"
	"strings"

	httpadapter "github.com/viralforge/mesh/services/trust-compliance/M50-consent-service/internal/adapters/http"
	"github.com/viralforge/mesh/services/trust-compliance/M50-consent-service/internal/adapters/postgres"
	"github.com/viralforge/mesh/services/trust-compliance/M50-consent-service/internal/application"
)

func main() {
	repos := postgres.NewRepositories()
	svc := application.NewService(application.Dependencies{
		Consents:    repos.Consents,
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

	log.Printf("M50-Consent-Service listening on %s", addr)
	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatal(err)
	}
}
