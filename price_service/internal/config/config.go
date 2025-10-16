package config

import (
	"context"
	"log/slog"

	"github.com/sethvargo/go-envconfig"
)

type Config struct {
	HTTP       HTTP
	Postgresql Postgresql
}

type HTTP struct {
	Host string `env:"HTTP_HOST, default=0.0.0.0"`
	Port string `env:"HTTP_PORT, default=8085"`
}

type Postgresql struct {
	Host     string `env:"PG_HOST, default=127.0.0.1"`
	Port     string `env:"PG_PORT, default=5432"`
	User     string `env:"PG_USER, default=postgres"`
	Password string `env:"PG_PASSWORD, default=5432"`
	Database string `env:"PG_DATABASE, default=control-plane"`
}

func New(ctx context.Context) (*Config, error) {
	var configHttp HTTP
	var configPostgresql Postgresql

	if err := envconfig.ProcessWith(ctx, &envconfig.Config{
		Target: &configHttp,

		DefaultDelimiter: ";",
		DefaultSeparator: "@",
	}); err != nil {
		slog.Error("failed to process env http vars", err.Error())
		return nil, err
	}

	if err := envconfig.ProcessWith(ctx, &envconfig.Config{
		Target: &configPostgresql,

		DefaultDelimiter: ";",
		DefaultSeparator: "@",
	}); err != nil {
		slog.Error("failed to process env postgresql vars", err.Error())
		return nil, err
	}

	return &Config{
		HTTP:       configHttp,
		Postgresql: configPostgresql,
	}, nil
}
