package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"os"
	"strings"

	"github.com/usamaroman/faas_demo/pkg/kafka"
	"github.com/usamaroman/faas_demo/pkg/logger"
	"github.com/usamaroman/faas_demo/pkg/types"

	kafkago "github.com/segmentio/kafka-go"
)

func main() {
	logger.NewLogger()
	slog.Info("Meter")

	addr, err := net.ResolveUDPAddr("udp", fmt.Sprintf(":%s", os.Getenv("UDP_PORT")))
	if err != nil {
		slog.Error("failed to resolve udp address", slog.String("error", err.Error()))
		os.Exit(1)
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		slog.Error("failed to listen udp", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer func() {
		if err = conn.Close(); err != nil {
			slog.Error("failed to close udp connection", slog.String("error", err.Error()))
		}
	}()

	addresses, ok := os.LookupEnv("KAFKA_ADDRS")
	if !ok {
		slog.Error("provide KAFKA_ADDRS env var")
		os.Exit(1)
	}

	metricsTopic := os.Getenv("FUNCTION_METRICS_TOPIC")
	if metricsTopic == "" {
		metricsTopic = "function_metrics"
	}
	actionsTopic := os.Getenv("FUNCTION_ACTIONS_TOPIC")
	if actionsTopic == "" {
		actionsTopic = "function_actions"
	}

	brokers := strings.Split(addresses, ",")

	metricsProducer := kafka.NewProducer(kafka.ProducerConfig{Topic: metricsTopic, Addrs: brokers})
	actionsProducer := kafka.NewProducer(kafka.ProducerConfig{Topic: actionsTopic, Addrs: brokers})
	ctx := context.Background()

	buf := make([]byte, 1024)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			slog.Error("failed to read data", slog.String("error", err.Error()))
			continue
		}

		var env types.Envelope
		if err := json.Unmarshal(buf[:n], &env); err != nil {
			slog.Error("failed to unmarshal envelope", slog.String("error", err.Error()))
			continue
		}

		switch env.Type {
		case "metadata":
			slog.Debug("routing to function_metrics", slog.Int("bytes", len(env.Payload)))
			if err := metricsProducer.WriteMessages(ctx, kafkago.Message{Value: env.Payload}); err != nil {
				slog.Error("failed to write metadata to kafka", slog.String("error", err.Error()))
				continue
			}
		case "action":
			slog.Debug("routing to function_actions", slog.Int("bytes", len(env.Payload)))
			if err := actionsProducer.WriteMessages(ctx, kafkago.Message{Value: env.Payload}); err != nil {
				slog.Error("failed to write action to kafka", slog.String("error", err.Error()))
				continue
			}
		default:
			slog.Error("unknown envelope type", slog.String("type", env.Type))
			continue
		}
	}
}
