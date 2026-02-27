package bootstrap

import "context"

func Build() error {
	_, err := NewRuntime(context.Background(), "configs/default.yaml")
	return err
}
