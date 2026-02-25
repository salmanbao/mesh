.PHONY: run-api run-worker test fmt tidy lint

run-api:
	go run ./cmd/api

run-worker:
	go run ./cmd/worker

test:
	go test ./...

fmt:
	gofmt -w ./cmd ./internal

tidy:
	go mod tidy

lint:
	golangci-lint run --disable-all -E depguard ./...
