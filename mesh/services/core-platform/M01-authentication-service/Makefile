.PHONY: run-api run-worker test lint

run-api:
	go run ./cmd/api

run-worker:
	go run ./cmd/worker

test:
	go test ./...

lint:
	golangci-lint run --disable-all -E depguard ./...
