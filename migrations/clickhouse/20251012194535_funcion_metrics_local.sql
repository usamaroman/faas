-- +goose Up
-- +goose StatementBegin
CREATE TABLE IF NOT EXISTS metrics.function_metrics_local (
    pod          String, 
    cpu_percent  Float32,
    mem_mb       Float32,
    timestamp    DateTime,
    tenant       String,
    type         String,
    start_time   DateTime,
    end_time     DateTime
)
ENGINE = MergeTree
PARTITION BY toYYYYMMDD(timestamp)
ORDER BY (pod, timestamp);
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS metrics.function_metrics_local;
-- +goose StatementEnd
