package bootstrap

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	HTTPPort string
}

func loadConfig() Config {
	port := strings.TrimSpace(os.Getenv("HTTP_PORT"))
	if port == "" {
		port = "8080"
	}
	if _, err := strconv.Atoi(strings.TrimPrefix(port, ":")); err != nil {
		port = "8080"
	}
	return Config{HTTPPort: port}
}
