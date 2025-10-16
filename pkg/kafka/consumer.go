package kafka

import (
	"github.com/segmentio/kafka-go"
)

type ConsumerConfig struct {
	Topic   string
	GroupID string
	Addrs   []string
}

func NewConsumer(cfg ConsumerConfig) *kafka.Reader {
	return kafka.NewReader(kafka.ReaderConfig{
		Brokers: cfg.Addrs,
		GroupID: cfg.GroupID,
		Topic:   cfg.Topic,
	})
}
