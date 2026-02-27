package bootstrap

import "context"

// Build wires runtime dependencies for this service.
func Build() error {
	_, err := NewRuntime(context.Background(), "configs/default.yaml")
	return err
}
