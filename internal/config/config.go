package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	HTTPPort        string
	Workers         int
	PoolSize        int
	ShutdownTimeout time.Duration
}

func New() Config {
	_ = godotenv.Load()

	cfg := Config{
		HTTPPort:        ":8080",
		Workers:         5,
		PoolSize:        10,
		ShutdownTimeout: time.Second * 10,
	}

	if v := strings.TrimSpace(os.Getenv("HTTP_PORT")); v != "" {
		cfg.HTTPPort = fmt.Sprintf(":%s", v)
	}
	if v := strings.TrimSpace(os.Getenv("WORKERS")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.Workers = n
		}
	}
	if v := strings.TrimSpace(os.Getenv("POOL_SIZE")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.PoolSize = n
		}
	}
	if v := strings.TrimSpace(os.Getenv("SHUTDOWN_TIMEOUT")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.ShutdownTimeout = time.Duration(n) * time.Second
		}
	}

	return cfg

}
