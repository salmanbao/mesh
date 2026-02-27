package bootstrap

import "context"

// Build wires runtime dependencies for this service.
func Build() error {
	runtime, err := NewRuntime(context.Background(), "configs/default.yaml")
	if err != nil {
		return err
	}
	return runtime.RunAPI(context.Background())
}
