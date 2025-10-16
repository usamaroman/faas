package main

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/usamaroman/faas_demo/pkg/logger"
	"github.com/usamaroman/faas_demo/pkg/types"
)

var (
	conn net.Conn
	err  error
)

const defaultMeterURLAddr = "localhost:5461"

func main() {
	logger.NewLogger()
	slog.Info("Meter Agent")

	meterURL, ok := os.LookupEnv("METER_URL")
	if !ok {
		meterURL = defaultMeterURLAddr
	}

	tenant := os.Getenv("TENANT")

	podName := os.Getenv("POD_NAME")
	if podName == "" {
		// fall back to hostname
		if hn, herr := os.Hostname(); herr == nil {
			podName = hn
		}
	}
	slog.Info("meter_agent pod identity", slog.String("podName", podName), slog.String("tenant", tenant))

	conn, err = net.Dial("udp", meterURL)
	if err != nil {
		slog.Error("failed to dial udp", slog.String("error", err.Error()))
		os.Exit(1)
	}
	slog.Info("udp connection", slog.Any("conn", conn))

	defer func() {
		if err = conn.Close(); err != nil {
			slog.Error("failed to close udp connection", slog.String("error", err.Error()))
		}
	}()

	ctx := context.Background()
	ctx, cancel := signal.NotifyContext(ctx, syscall.SIGTTIN, syscall.SIGTERM)
	defer cancel()

	metricsURL := os.Getenv("KNATIVE_METRICS_URL")
	if metricsURL == "" {
		metricsURL = "http://localhost:9091/metrics"
	}

	intervalSec := 1
	if v := os.Getenv("SCRAPE_INTERVAL_SEC"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			intervalSec = n
		}
	}

	ticker := time.NewTicker(time.Duration(intervalSec) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			// send stop action with duration
			endTs := time.Now().Unix()
			stopAction := types.Action{Pod: podName, Action: "stop", Timestamp: endTs, Tenant: tenant}
			sendAction(stopAction)
			return
		case <-ticker.C:
			memMB := scrapeKnativeMemoryMB(metricsURL)
			metric := types.Metric{Pod: podName, CPUPercent: 0.0, MemMB: memMB, Timestamp: time.Now().Unix(), Tenant: tenant}
			sendMetric(metric)
		}
	}
}

func scrapeKnativeMemoryMB(url string) float64 {
	resp, err := http.Get(url)
	if err != nil {
		slog.Error("failed to GET knative metrics", slog.String("error", err.Error()))
		return 0
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		slog.Error("failed to read knative metrics", slog.String("error", err.Error()))
		return 0
	}

	for line := range strings.SplitSeq(string(body), "\n") {
		if strings.HasPrefix(line, "revision_go_heap_alloc") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				valStr := parts[len(parts)-1]
				slog.Debug("got revision_go_heap_alloc metric", slog.String("value", valStr))
				if f, err := strconv.ParseFloat(valStr, 64); err == nil {
					return f / (1024.0 * 1024.0)
				}
			}
		}
	}

	return 0
}

func sendMetric(m types.Metric) {
	data, err := json.Marshal(m)
	if err != nil {
		slog.Error("failed to marshal metric", slog.String("error", err.Error()))
		return
	}
	env := types.Envelope{Type: "metadata", Payload: data}
	envBytes, err := json.Marshal(env)
	if err != nil {
		slog.Error("failed to marshal envelope", slog.String("error", err.Error()))
		return
	}
	slog.Debug("sending metadata event", slog.String("event", string(envBytes)))
	if _, err := conn.Write(envBytes); err != nil {
		slog.Error("failed to send metric envelope", slog.String("error", err.Error()))
	}
}

func sendAction(a types.Action) {
	data, err := json.Marshal(a)
	if err != nil {
		slog.Error("failed to marshal action", slog.String("error", err.Error()))
		return
	}
	env := types.Envelope{Type: "action", Payload: data}
	envBytes, err := json.Marshal(env)
	if err != nil {
		slog.Error("failed to marshal envelope", slog.String("error", err.Error()))
		return
	}
	slog.Debug("sending action event", slog.String("event", string(envBytes)))
	if _, err := conn.Write(envBytes); err != nil {
		slog.Error("failed to send action envelope", slog.String("error", err.Error()))
	}
}
