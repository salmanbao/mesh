package bootstrap

import "context"

// Build validates runtime bootstrap wiring for this service.
func Build() error {
	_, err := NewRuntime(context.Background(), "configs/default.yaml")
	return err
}
