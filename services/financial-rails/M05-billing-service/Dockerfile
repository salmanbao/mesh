FROM golang:1.22-alpine
WORKDIR /app
COPY . .
RUN go build -o /out/api ./cmd/api
CMD ["/out/api"]
