FROM golang:1.23-alpine AS build
WORKDIR /src
COPY . .
RUN go build -o /out/api ./cmd/api && go build -o /out/worker ./cmd/worker

FROM alpine:3.20
WORKDIR /app
COPY --from=build /out/api /out/api
COPY --from=build /out/worker /out/worker
COPY configs ./configs
CMD ["/out/api"]
