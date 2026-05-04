// Package config provides application configuration management via environment variables.
package config

import (
	"errors"
	"log/slog"
	"strings"

	"github.com/joho/godotenv"
	"go-simpler.org/env"
)

type LogLevel string

type Config struct {
	LogLevel   LogLevel `env:"LOG_LEVEL" default:"INFO"`
	DBDSN      string   `env:"DB_DSN" default:"file:cardman.db?_fk=1"`
	SSHPort    int      `env:"SSH_PORT" default:"2222"`
	SSHHostKey string   `env:"SSH_HOST_KEY" default:"~/.ssh/cardman_host_key"`
	APIKey     string   `env:"API_KEY" default:""`
}

var Cfg Config

func LoadConfig() error {
	cfg, err := Load()
	if err != nil {
		return err
	}
	Cfg = cfg
	return nil
}

func Load() (Config, error) {
	_ = godotenv.Load()
	var cfg Config
	if err := env.Load(&cfg, &env.Options{}); err != nil {
		return Config{}, err
	}
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) Validate() error {
	if c.SSHPort < 1 || c.SSHPort > 65535 {
		return errors.New("SSH_PORT must be between 1 and 65535")
	}
	return nil
}

func (c Config) GetAPIKey() string {
	return c.APIKey
}

func GetAPIKey() string {
	return Cfg.GetAPIKey()
}

func (c Config) GetLogLevel() slog.Level {
	switch strings.ToUpper(string(c.LogLevel)) {
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

func GetLogLevel() slog.Level {
	return Cfg.GetLogLevel()
}
