package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/usamaroman/faas_demo/pkg/logger"
	"github.com/usamaroman/faas_demo/pkg/postgresql"
	"github.com/usamaroman/faas_demo/price_service/internal/config"
	v1 "github.com/usamaroman/faas_demo/price_service/internal/controller/v1"
	"github.com/usamaroman/faas_demo/price_service/internal/repo"
	"github.com/usamaroman/faas_demo/price_service/internal/service"
)

func Run() {
	logger.NewLogger()
	slog.Info("Starting control plane service")
	ctx, cancel := context.WithCancel(context.Background())

	cfg, err := config.New(ctx)
	if err != nil {
		slog.Error("failed to init config", slog.String("error", err.Error()))
		os.Exit(1)
	}

	slog.Info("postgresql starting")
	pgConfig := postgresql.Config{
		Host:     cfg.Postgresql.Host,
		Port:     cfg.Postgresql.Port,
		User:     cfg.Postgresql.User,
		Password: cfg.Postgresql.Password,
		Database: cfg.Postgresql.Database,
	}
	postgres, err := postgresql.New(pgConfig)
	if err != nil {
		slog.Error("failed to init postgresql", slog.String("error", err.Error()))
		os.Exit(1)
	}

	slog.Info("repositories init")
	repositories := repo.NewRepositories(postgres)

	slog.Info("services init")
	services := service.NewServices(&service.Dependencies{
		Repos: repositories,
	})

	r := router()
	v1.NewRouter(r, services)

	slog.Debug("server starting")
	server := &http.Server{
		Addr:    fmt.Sprintf("%s:%s", cfg.HTTP.Host, cfg.HTTP.Port),
		Handler: r,
	}

	serverErrors := make(chan error, 1)
	go func() {
		slog.Info("starting http server", slog.String("address", fmt.Sprintf("%s:%s", cfg.HTTP.Host, cfg.HTTP.Port)))
		serverErrors <- server.ListenAndServe()
	}()

	slog.Info("configuring graceful shutdown")
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	select {
	case s := <-interrupt:
		slog.Info("application got signal", slog.String("signal", s.String()))
	case err = <-serverErrors:
		if err != nil {
			slog.Error("http server error", slog.String("error", err.Error()))
		}
	}

	slog.Info("shutting down server")
	ctx, cancel = context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err = server.Shutdown(ctx); err != nil {
		slog.Error("failed to shutdown http server", slog.String("error", err.Error()))
		if err = server.Close(); err != nil {
			slog.Error("failed to close http server", slog.String("error", err.Error()))
		}
	}

	postgres.Close()
	slog.Info("application stopped")
}

func router() *gin.Engine {
	var r *gin.Engine

	if env := os.Getenv("APP_ENV"); env == "prod" {
		gin.SetMode(gin.ReleaseMode)
		r = gin.New()
		r.Use(gin.Recovery())
	} else {
		r = gin.Default()
	}

	return r
}
