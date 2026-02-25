# Mesh Local Development

## Prerequisites
- Go 1.23+
- Docker + Docker Compose

## Start Infra
`docker compose -f environments/compose/docker-compose.base.yaml up -d`

## Run One Service
`cd services/<cluster>/<service>` then `go run ./cmd/api`

## Run Validation
`bash scripts/validate-mesh-structure.sh`
`bash scripts/generate-mesh-index.sh --check`

## Enforce Service Import Boundaries
- Per service:
  - `cd services/<cluster>/<service>`
  - `make lint`
- Whole repo:
  - `make services-depguard`
