module github.com/viralforge/mesh/services/data-ai/M06-media-processing-pipeline

go 1.23

toolchain go1.24.5

require (
	github.com/go-chi/chi/v5 v5.2.3
	github.com/google/uuid v1.6.0
	github.com/viralforge/mesh/contracts v0.0.0
	google.golang.org/grpc v1.75.1
	gopkg.in/yaml.v3 v3.0.1
)

require (
	golang.org/x/net v0.41.0 // indirect
	golang.org/x/sys v0.33.0 // indirect
	golang.org/x/text v0.26.0 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20250707201910-8d1bb00bc6a7 // indirect
	google.golang.org/protobuf v1.36.6 // indirect
)

replace github.com/viralforge/mesh/contracts => ../../../contracts
