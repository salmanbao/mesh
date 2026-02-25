FROM golang:1.23-alpine AS builder
WORKDIR /src

# Build context is expected to be mesh root so local replace paths resolve.
COPY . .
WORKDIR /src/services/core-platform/M01-authentication-service
RUN go build -o /out/api ./cmd/api && \
    go build -o /out/worker ./cmd/worker

FROM alpine:3.20
RUN apk add --no-cache ca-certificates
WORKDIR /app

COPY --from=builder /out/api /out/api
COPY --from=builder /out/worker /out/worker
COPY --from=builder /src/services/core-platform/M01-authentication-service/configs /app/configs

# Default mode runs API; worker uses command override in compose/k8s.
CMD ["/out/api"]
