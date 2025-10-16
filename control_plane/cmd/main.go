package main

import (
	"log/slog"
	"net/http"
	"os"
	"strings"

	docs "github.com/usamaroman/faas_demo/control_plane/docs"
	"github.com/usamaroman/faas_demo/control_plane/internal/config"
	httpapi "github.com/usamaroman/faas_demo/control_plane/internal/http"
	"github.com/usamaroman/faas_demo/pkg/k8s"
	"github.com/usamaroman/faas_demo/pkg/kafka"
	"github.com/usamaroman/faas_demo/pkg/logger"
)

// @title			Control Plane API
// @version		1.0
// @description	API for running functions on Knative
// @BasePath		/
func main() {
	logger.NewLogger()
	run()
}

func run() {
	cfg := config.Load()

	if strings.HasPrefix(cfg.HTTP.Addr, ":") {
		docs.SwaggerInfo.Host = "localhost" + cfg.HTTP.Addr
	} else {
		docs.SwaggerInfo.Host = cfg.HTTP.Addr
	}
	docs.SwaggerInfo.BasePath = "/"
	docs.SwaggerInfo.Schemes = []string{"http"}

	restCfg, err := k8s.NewRESTConfig(k8s.Config{InCluster: false})
	if err != nil {
		slog.Error("failed to get rest config", slog.String("error", err.Error()))
		return
	}

	addresses, ok := os.LookupEnv("KAFKA_ADDRS")
	if !ok {
		slog.Error("provide KAFKA_ADDRS env var")
		os.Exit(1)
	}

	actionsTopic := os.Getenv("FUNCTION_ACTIONS_TOPIC")
	if actionsTopic == "" {
		actionsTopic = "function_actions"
	}

	brokers := strings.Split(addresses, ",")

	actionsProducer := kafka.NewProducer(kafka.ProducerConfig{Topic: actionsTopic, Addrs: brokers})

	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) })

	api := httpapi.New(cfg, actionsProducer, restCfg)
	api.Register(mux)

	slog.Info("control plane listening", slog.String("addr", cfg.HTTP.Addr))
	if err := http.ListenAndServe(cfg.HTTP.Addr, mux); err != nil {
		slog.Error("server error", slog.String("error", err.Error()))
	}
}
