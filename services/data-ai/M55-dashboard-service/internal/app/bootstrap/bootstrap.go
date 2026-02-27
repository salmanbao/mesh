package bootstrap

import "context"

// Build validates that runtime dependencies can be wired with default config.
func Build() error {
	_, err := NewRuntime(context.Background(), "configs/default.yaml")
	return err
}
