package bootstrap

import "context"

// Build validates runtime wiring with default config.
func Build() error {
	_, err := NewRuntime(context.Background(), "configs/default.yaml")
	return err
}
