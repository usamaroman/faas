package kafka

import (
	"log/slog"
	"os"
	"strings"

	"github.com/segmentio/kafka-go"
)

type ProducerConfig struct {
	Topic string
	Addrs []string
}

func NewProducer(cfg ProducerConfig) *kafka.Writer {
	conn, err := kafka.Dial("tcp", strings.Join(cfg.Addrs, ","))
	if err != nil {
		slog.Error("failed to dial brokers", slog.String("addrs", strings.Join(cfg.Addrs, ",")), slog.String("error", err.Error()))
		os.Exit(1)
	}

	if err := conn.CreateTopics(kafka.TopicConfig{
		Topic:             cfg.Topic,
		ReplicationFactor: -1,
		NumPartitions:     -1,
	}); err != nil {
		slog.Error("failed to create topic", slog.String("topic", cfg.Topic), slog.String("error", err.Error()))
		os.Exit(1)
	}

	return kafka.NewWriter(kafka.WriterConfig{
		Brokers:  cfg.Addrs,
		Topic:    cfg.Topic,
		Balancer: &kafka.Hash{}, // hash for partitions
	})
}
