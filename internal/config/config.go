package config

import (
	"go-simpler.org/env"
)

type LogLevel string

var cfg struct {
	LogLevel LogLevel `env:"LOG_LEVEL" default:"INFO"`
}

func LoadConfig() {
	if err := env.Load(&cfg, &env.Options{}); err != nil {
		panic(err)
	}
}
