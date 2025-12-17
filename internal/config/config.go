package config

import (
	"time"
)

type Config struct {
	HTTPPort        string
	Workers         int
	PoolSize        int
	ShutdownTimeout time.Duration
}

func New() Config {
	return Config{
		HTTPPort:        ":8080",
		Workers:         5,
		PoolSize:        100,
		ShutdownTimeout: time.Second * 10,
	}
}
