package config

import (
	"os"
	"strconv"
	"strings"
)

type HTTPConfig struct {
	Addr string
}

type DockerConfig struct {
	Host     string
	Username string
	Password string
}

type KafkaConfig struct {
	Topic   string
	Brokers []string
}

type LimitsConfig struct {
	MaxUploadSize int64
}

type Config struct {
	HTTP   HTTPConfig
	Kafka  KafkaConfig
	K8S    K8SConfig
	Limits LimitsConfig
	Meter  MeterConfig
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getEnvInt64(key string, def int64) int64 {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil {
			return n
		}
	}
	return def
}

func Load() Config {
	// Defaults are simple and overridable by env
	addr := getEnv("HTTP_ADDR", ":8080")

	kafkaBrokers := getEnv("KAFKA_BROKERS", "")
	kafkaTopic := getEnv("KAFKA_TOPIC", "")

	return Config{
		HTTP: HTTPConfig{Addr: addr},
		Kafka: KafkaConfig{
			Topic:   kafkaTopic,
			Brokers: splitAndTrim(kafkaBrokers),
		},
		K8S: K8SConfig{
			Namespace: getEnv("K8S_NAMESPACE", "default"),
		},
		Limits: LimitsConfig{
			MaxUploadSize: getEnvInt64("MAX_UPLOAD_SIZE_BYTES", 50*1024*1024),
		},
		Meter: MeterConfig{
			URL: getEnv("METER_URL", "host.docker.internal:5461"),
		},
	}
}

type K8SConfig struct {
	Namespace  string
	Kubeconfig string
}

type MeterConfig struct {
	URL string
}

func splitAndTrim(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		t := strings.TrimSpace(p)
		if t != "" {
			out = append(out, t)
		}
	}
	return out
}
