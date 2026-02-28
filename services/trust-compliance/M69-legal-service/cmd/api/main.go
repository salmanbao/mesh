package main

import (
	"log"
	"net/http"
	"os"
	"strings"

	httpadapter "github.com/viralforge/mesh/services/trust-compliance/M69-legal-service/internal/adapters/http"
	"github.com/viralforge/mesh/services/trust-compliance/M69-legal-service/internal/adapters/postgres"
	"github.com/viralforge/mesh/services/trust-compliance/M69-legal-service/internal/application"
)

func main() {
	repos := postgres.NewRepositories()
	svc := application.NewService(application.Dependencies{
		Documents:   repos.Documents,
		Signatures:  repos.Signatures,
		Holds:       repos.Holds,
		Compliance:  repos.Compliance,
		Disputes:    repos.Disputes,
		DMCA:        repos.DMCANotices,
		Filings:     repos.Filings,
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

	log.Printf("M69-Legal-Service listening on %s", addr)
	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatal(err)
	}
}
