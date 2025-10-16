package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"github.com/go-gomail/gomail"
	"github.com/kelseyhightower/envconfig"

	"github.com/usamaroman/faas_demo/pkg/kafka"
	"github.com/usamaroman/faas_demo/pkg/logger"
)

type Config struct {
	SMTP struct {
		Host     string `envconfig:"SMTP_HOST" required:"true"`
		Port     int    `envconfig:"SMTP_PORT" required:"true"`
		Username string `envconfig:"SMTP_USERNAME" required:"true"`
		Password string `envconfig:"SMTP_PASSWORD" required:"true"`
	}
	Kafka struct {
		Topic           string   `envconfig:"KAFKA_TOPIC" required:"true"`
		ConsumerGroupID string   `envconfig:"KAFKA_CONSUMER_GROUP_ID" required:"true"`
		Addrs           []string `envconfig:"KAFKA_ADDRS" required:"true"`
	}
}

type NotificationMessage struct {
	TenantID  string  `json:"tenant_id"`
	Email     string  `json:"email"`
	MemoryMB  float64 `json:"memory_mb"`
	TotalCost float64 `json:"total_cost"`
	PodName   string  `json:"pod_name"`
	Timestamp int64   `json:"timestamp"`
}

func main() {
	logger.NewLogger()
	slog.Info("Notifier")

	var cfg Config
	if err := envconfig.Process("", &cfg); err != nil {
		slog.Error("failed to load config", slog.String("error", err.Error()))
		os.Exit(1)
	}

	consumerCfg := kafka.ConsumerConfig{
		Topic:   cfg.Kafka.Topic,
		GroupID: cfg.Kafka.ConsumerGroupID,
		Addrs:   cfg.Kafka.Addrs,
	}

	consumer := kafka.NewConsumer(consumerCfg)
	ctx := context.Background()

	for {
		msg, err := consumer.ReadMessage(ctx)
		if err != nil {
			slog.Error("failed to read message", slog.String("error", err.Error()))
			continue
		}

		var notification NotificationMessage
		if err := json.Unmarshal(msg.Value, &notification); err != nil {
			slog.Error("failed to unmarshal notification message", slog.String("error", err.Error()))
			continue
		}

		slog.Info("processing notification",
			slog.String("tenant_id", notification.TenantID),
			slog.String("email", notification.Email),
			slog.Float64("memory_mb", notification.MemoryMB),
			slog.Float64("total_cost", notification.TotalCost),
			slog.String("pod_name", notification.PodName))

		m := gomail.NewMessage()
		m.SetHeader("From", cfg.SMTP.Username)
		m.SetHeader("To", notification.Email)
		m.SetHeader("Subject", "FaaS Billing Notification - Time to Pay!")
		m.SetBody("text/html", fmt.Sprintf(`
			<html>
			<body>
				<h2>FaaS Billing Notification</h2>
				<p>Dear %s,</p>
				<p>Your function execution has completed. Here are the billing details:</p>
				<ul>
					<li><strong>Pod Name:</strong> %s</li>
					<li><strong>Memory Used:</strong> %.2f MB</li>
					<li><strong>Total Cost:</strong> $%.2f</li>
					<li><strong>Timestamp:</strong> %s</li>
				</ul>
				<p>Please ensure payment is processed for your FaaS usage.</p>
				<p>Best regards,<br>FaaS Team</p>
			</body>
			</html>
		`, notification.TenantID, notification.PodName, notification.MemoryMB, notification.TotalCost,
			fmt.Sprintf("%d", notification.Timestamp)))

		d := gomail.NewDialer(cfg.SMTP.Host, cfg.SMTP.Port, cfg.SMTP.Username, cfg.SMTP.Password)
		if err := d.DialAndSend(m); err != nil {
			slog.Error("failed to send email",
				slog.String("error", err.Error()),
				slog.String("email", notification.Email))
		} else {
			slog.Info("email sent successfully",
				slog.String("email", notification.Email),
				slog.String("tenant_id", notification.TenantID))
		}
	}
}
