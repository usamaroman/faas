-- +goose Up
-- +goose StatementBegin
CREATE TABLE tariffs (
     id SERIAL PRIMARY KEY,
     name VARCHAR(50),
     exec_price numeric(10, 2),
     mem_price numeric(10, 2),
     cpu_price numeric(10, 2),
     created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
     updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_tariffs_updated_at BEFORE UPDATE ON tariffs
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

INSERT INTO tariffs (name, exec_price, mem_price, cpu_price) VALUES ('basic', 0, 0, 0)
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TRIGGER IF EXISTS update_tariffs_updated_at ON tariffs;
DROP FUNCTION IF EXISTS update_updated_at_column();
DROP TABLE tariffs;
-- +goose StatementEnd
