-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS metrics.function_metrics_kafka (
    pod          String, 
    cpu_percent  Float32,
    mem_mb       Float32,
    timestamp    Int64,
    tenant       String,
    type         String,
    start_time   Int64,
    end_time     Int64
) ENGINE = Kafka
SETTINGS 
    kafka_broker_list = 'kafka:29092',
    kafka_topic_list = 'function_metrics',
    kafka_group_name = 'function_metrics_clickhouse',
    kafka_format = 'JSONEachRow';
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS metrics.function_metrics_kafka;
-- +goose StatementEnd
