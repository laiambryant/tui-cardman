package config

import (
	"fmt"
	"log/slog"

	"github.com/joho/godotenv"
	"go-simpler.org/env"
)

type LogLevel string

var Cfg struct {
	LogLevel   LogLevel `env:"LOG_LEVEL" default:"INFO"`
	DBDSN      string   `env:"DB_DSN" default:"file:cardman.db?_fk=1"`
	SSHPort    int      `env:"SSH_PORT" default:"2222"`
	SSHHostKey string   `env:"SSH_HOST_KEY" default:"~/.ssh/cardman_host_key"`
}

func LoadConfig() {
	_ = godotenv.Load()
	if err := env.Load(&Cfg, &env.Options{}); err != nil {
		panic(err)
	}
	fmt.Printf("Current configuration: %+v\n", Cfg)
}

func GetLogLevel() slog.Level {
	switch Cfg.LogLevel {
	case "DEBUG":
		return slog.LevelDebug
	case "INFO":
		return slog.LevelInfo
	case "WARN":
		return slog.LevelWarn
	case "ERROR":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
