package main

import (
	"log"
	"net/http"
	"os"
	"strings"

	httpadapter "github.com/viralforge/mesh/services/integrations/M71-integration-hub/internal/adapters/http"
	"github.com/viralforge/mesh/services/integrations/M71-integration-hub/internal/adapters/postgres"
	"github.com/viralforge/mesh/services/integrations/M71-integration-hub/internal/application"
)

func main() {
	repos := postgres.NewRepositories()
	svc := application.NewService(application.Dependencies{
		Integrations: repos.Integrations,
		Credentials:  repos.Credentials,
		Workflows:    repos.Workflows,
		Executions:   repos.Executions,
		Webhooks:     repos.Webhooks,
		Deliveries:   repos.Deliveries,
		Analytics:    repos.Analytics,
		Logs:         repos.Logs,
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
	log.Printf("M71-Integration-Hub listening on %s", addr)
	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatal(err)
	}
}
