-- +goose Up
-- +goose StatementBegin
CREATE MATERIALIZED VIEW IF NOT EXISTS metrics.function_metrics_mv
TO metrics.function_metrics_local
AS
SELECT
    pod,
    cpu_percent,
    mem_mb,
    toDateTime(timestamp) AS timestamp,
    tenant
FROM metrics.function_metrics_kafka;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP MATERIALIZED VIEW IF EXISTS metrics.function_metrics_mv;
-- +goose StatementEnd
