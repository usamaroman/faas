package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	gokafka "github.com/segmentio/kafka-go"
	"github.com/usamaroman/faas_demo/pkg/clickhouse"
	"github.com/usamaroman/faas_demo/pkg/kafka"
	"github.com/usamaroman/faas_demo/pkg/logger"
	"github.com/usamaroman/faas_demo/pkg/types"
)

const defaultTariffID = 1

type Invoicer struct {
	clickhouseClient *clickhouse.Client
	priceServiceURL  string
	actionsConsumer  *gokafka.Reader
	notifyProducer   *gokafka.Writer
	metricsCache     map[string][]types.Metric
}

type BillingData struct {
	PodName               string    `json:"pod_name"`
	StartTime             time.Time `json:"start_time"`
	EndTime               time.Time `json:"end_time"`
	TotalMemoryConsumedMB float64   `json:"total_memory_consumed_mb_sec"`
	DurationSeconds       int64     `json:"duration_seconds"`
}

type Tariff struct {
	ID        int     `json:"id"`
	Name      string  `json:"name"`
	ExecPrice float64 `json:"exec_price"`
	MemPrice  float64 `json:"mem_price"`
	CpuPrice  float64 `json:"cpu_price"`
}

type BillingResponse struct {
	TenantID     string    `json:"tenant_id"`
	PodName      string    `json:"pod_name"`
	DurationSec  int64     `json:"duration_sec"`
	MemoryMB     float64   `json:"memory_mb"`
	ExecCost     float64   `json:"exec_cost"`
	MemoryCost   float64   `json:"memory_cost"`
	TotalCost    float64   `json:"total_cost"`
	TariffName   string    `json:"tariff_name"`
	CalculatedAt time.Time `json:"calculated_at"`
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
	slog.Info("Starting Invoicer service")

	// ClickHouse config
	clickhouseCfg := clickhouse.Config{
		Host:     getEnv("CLICKHOUSE_HOST", "localhost"),
		Port:     getEnv("CLICKHOUSE_PORT", "9000"),
		Username: getEnv("CLICKHOUSE_USER", "default"),
		Password: getEnv("CLICKHOUSE_PASSWORD", ""),
		Database: getEnv("CLICKHOUSE_DB", "metrics"),
	}

	clickhouseClient, err := clickhouse.New(context.Background(), clickhouseCfg)
	if err != nil {
		slog.Error("failed to connect to clickhouse", slog.Any("cfg", clickhouseCfg), slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer clickhouseClient.Close()

	// Price service URL
	priceServiceURL := getEnv("PRICE_SERVICE_URL", "http://localhost:8080")

	// Kafka consumers
	addresses, ok := os.LookupEnv("KAFKA_ADDRS")
	if !ok {
		slog.Error("provide KAFKA_ADDRS env var")
		os.Exit(1)
	}

	actionsTopic := getEnv("KAFKA_ACTIONS_TOPIC", "function_actions")
	actionsConsumerGroupName := getEnv("KAFKA_ACTIONS_CONSUMER_GROUP_NAME", "invoicer-actions")

	actionsConsumerCfg := kafka.ConsumerConfig{
		Topic:   actionsTopic,
		GroupID: actionsConsumerGroupName,
		Addrs:   strings.Split(addresses, ","),
	}

	actionsConsumer := kafka.NewConsumer(actionsConsumerCfg)

	// Notify producer
	notifyTopic := getEnv("KAFKA_NOTIFY_TOPIC", "notify")
	notifyProducerCfg := kafka.ProducerConfig{
		Topic: notifyTopic,
		Addrs: strings.Split(addresses, ","),
	}
	notifyProducer := kafka.NewProducer(notifyProducerCfg)

	invoicer := &Invoicer{
		clickhouseClient: clickhouseClient,
		priceServiceURL:  priceServiceURL,
		actionsConsumer:  actionsConsumer,
		notifyProducer:   notifyProducer,
		metricsCache:     make(map[string][]types.Metric),
	}

	// Start background consumers
	go invoicer.consumeActions()

	// Setup HTTP server
	r := gin.Default()
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Billing endpoint
	r.GET("/billing/:tenant_id", invoicer.getBilling)

	port := getEnv("PORT", "8080")
	slog.Info("Starting HTTP server", slog.String("port", port))
	if err := r.Run(":" + port); err != nil {
		slog.Error("Failed to start HTTP server", slog.String("error", err.Error()))
		os.Exit(1)
	}
}

func (i *Invoicer) consumeActions() {
	ctx := context.Background()
	slog.Info("actions consumer started")

	for {
		msg, err := i.actionsConsumer.ReadMessage(ctx)
		if err != nil {
			slog.Error("failed to read action message", slog.String("error", err.Error()))
			continue
		}

		var action types.Action
		if err := json.Unmarshal(msg.Value, &action); err != nil {
			slog.Error("failed to unmarshal action", slog.String("error", err.Error()))
			continue
		}

		slog.Info("processing action",
			slog.String("pod", action.Pod),
			slog.String("action", action.Action),
		)

		if action.Action == "stop" {
			// Send email notification for stop action
			go i.sendStopNotification(action)
		}
	}
}

func (i *Invoicer) getBilling(c *gin.Context) {
	tenantID := c.Param("tenant_id")
	if tenantID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "tenant_id is required"})
		return
	}

	// Get billing data from ClickHouse
	billingData, err := i.getBillingDataFromClickHouse(tenantID)
	if err != nil {
		slog.Error("failed to get billing data", slog.String("error", err.Error()))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get billing data"})
		return
	}

	if len(billingData) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "no billing data found for tenant"})
		return
	}

	tariff, err := i.getTariffFromPriceService(defaultTariffID)
	if err != nil {
		slog.Error("failed to get tariff", slog.String("error", err.Error()))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to get tariff"})
		return
	}

	var totalExecCost, totalMemoryCost float64
	var totalDuration int64
	var totalMemoryMB float64

	resp := make([]BillingResponse, 0, len(billingData))

	for _, data := range billingData {
		duration := int64(data.EndTime.Sub(data.StartTime).Seconds())
		totalDuration += duration
		totalMemoryMB += data.TotalMemoryConsumedMB

		// Calculate costs based on tariff
		execCost := float64(duration) * tariff.ExecPrice
		memoryCost := data.TotalMemoryConsumedMB * tariff.MemPrice

		totalExecCost += execCost
		totalMemoryCost += memoryCost

		totalCost := totalExecCost + totalMemoryCost
		resp = append(resp, BillingResponse{
			TenantID:     tenantID,
			PodName:      billingData[0].PodName,
			DurationSec:  totalDuration,
			MemoryMB:     totalMemoryMB,
			ExecCost:     totalExecCost,
			MemoryCost:   totalMemoryCost,
			TotalCost:    totalCost,
			TariffName:   tariff.Name,
			CalculatedAt: time.Now(),
		})
	}

	c.JSON(http.StatusOK, resp)
}

func (i *Invoicer) getBillingDataFromClickHouse(tenantID string) ([]BillingData, error) {
	query := `
		SELECT 
			pod,
			min(timestamp) AS start_time,
			max(timestamp) AS end_time,
			sum(mem_mb) AS total_memory_consumed_mb_sec
		FROM function_metrics_local 
		WHERE tenant = ? 
		GROUP BY pod
	`

	rows, err := i.clickhouseClient.Query(context.Background(), query, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []BillingData
	for rows.Next() {
		var data BillingData
		if err := rows.Scan(&data.PodName, &data.StartTime, &data.EndTime, &data.TotalMemoryConsumedMB); err != nil {
			return nil, err
		}
		data.DurationSeconds = int64(data.EndTime.Sub(data.StartTime).Seconds())
		result = append(result, data)
	}

	return result, nil
}

func (i *Invoicer) getTariffFromPriceService(tariffID int) (*Tariff, error) {
	url := fmt.Sprintf("%s/v1/tariff/%d", i.priceServiceURL, tariffID)

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("price service returned status %d", resp.StatusCode)
	}

	var response struct {
		Tariff Tariff `json:"tariff"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	return &response.Tariff, nil
}

func (i *Invoicer) sendStopNotification(action types.Action) {
	// Get billing data for this pod to calculate costs
	billingData, err := i.getBillingDataFromClickHouse(action.Tenant)
	if err != nil {
		slog.Error("failed to get billing data for notification", slog.String("error", err.Error()))
		return
	}

	if len(billingData) == 0 {
		slog.Warn("no billing data found for notification", slog.String("tenant", action.Tenant))
		return
	}

	// Get tariff
	tariff, err := i.getTariffFromPriceService(defaultTariffID) // Default tariff ID
	if err != nil {
		slog.Error("failed to get tariff for notification", slog.String("error", err.Error()))
		return
	}

	// Calculate costs
	var totalMemoryMB, totalCost float64
	for _, data := range billingData {
		if data.PodName == action.Pod {
			duration := int64(data.EndTime.Sub(data.StartTime).Seconds())
			execCost := float64(duration) * tariff.ExecPrice
			memoryCost := data.TotalMemoryConsumedMB * tariff.MemPrice
			totalMemoryMB = data.TotalMemoryConsumedMB
			totalCost = execCost + memoryCost
			break
		}
	}

	// Create notification message
	notification := NotificationMessage{
		TenantID:  action.Tenant,
		Email:     action.Tenant,
		MemoryMB:  totalMemoryMB,
		TotalCost: totalCost,
		PodName:   action.Pod,
		Timestamp: time.Now().Unix(),
	}

	// Send notification to Kafka
	notificationBytes, err := json.Marshal(notification)
	if err != nil {
		slog.Error("failed to marshal notification", slog.String("error", err.Error()))
		return
	}

	ctx := context.Background()
	if err := i.notifyProducer.WriteMessages(ctx, gokafka.Message{Value: notificationBytes}); err != nil {
		slog.Error("failed to send notification", slog.String("error", err.Error()))
		return
	}

	slog.Info("sent stop notification",
		slog.String("pod", action.Pod),
		slog.String("tenant", action.Tenant),
		slog.Float64("memory_mb", totalMemoryMB),
		slog.Float64("total_cost", totalCost))
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
