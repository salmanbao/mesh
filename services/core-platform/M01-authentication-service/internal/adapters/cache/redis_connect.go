package cache

import (
	"context"
	"fmt"
	"strings"

	"github.com/redis/go-redis/v9"
)

// Connect initializes a Redis client from URL or host:port input.
// Supporting both formats keeps local/dev and container config paths simple.
func Connect(_ context.Context, redisURL string) (*redis.Client, error) {
	if strings.HasPrefix(redisURL, "redis://") {
		opt, parseErr := redis.ParseURL(redisURL)
		if parseErr != nil {
			return nil, fmt.Errorf("parse redis url: %w", parseErr)
		}
		return redis.NewClient(opt), nil
	}
	return redis.NewClient(&redis.Options{Addr: redisURL}), nil
}
