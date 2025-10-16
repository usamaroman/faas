#!/bin/bash
set -e

# Step 0: K8s setup
chmod +x install_knative_1_17_kourier.sh
./install_knative_1_17_kourier.sh

# Step 1: Install goose using Makefile
echo "Step 1: Installing goose..."
make goose

# Step 2: Build all services
echo "Step 2: Building all Go services..."
make build_all

# Step 3: Start infrastructure containers
echo "Step 3: Starting infrastructure (Postgres, Zookeeper, ClickHouse, Kafka)..."
docker compose up -d postgres zookeeper clickhouse kafka kafka-ui

# Wait for Postgres and ClickHouse to be ready 
echo "Waiting for infrastructure services to be ready..."
sleep 15

# Step 4: Run database migrations
echo "Step 4: Running Postgres migrations..."
GOOSE_DBSTRING="postgres://postgres:5432@localhost:5432/control-plane" ./bin/goose postgres up -dir price_service/migrations

echo "Running ClickHouse migrations..."
GOOSE_DBSTRING="tcp://user:1234@localhost:9000/metrics" ./bin/goose clickhouse up -dir migrations/clickhouse

# Step 5: Run application services 
echo "Step 5: Starting application services..."
docker compose up -d meter price_service notifier invoicer control_plane

echo "Setup completed successfully."
