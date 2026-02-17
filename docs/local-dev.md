# Mesh Local Development

## Prerequisites
- Go 1.22+
- Docker + Docker Compose

## Start Infra
`docker compose -f environments/compose/docker-compose.base.yaml up -d`

## Run One Service
`cd services/<cluster>/<service>` then `go run ./cmd/api`

## Run Validation
`bash scripts/validate-mesh-structure.sh`
`bash scripts/generate-mesh-index.sh --check`
