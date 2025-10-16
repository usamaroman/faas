package tariff

import (
	"context"
	"errors"
	"log/slog"

	"github.com/usamaroman/faas_demo/pkg/postgresql"
	"github.com/usamaroman/faas_demo/price_service/internal/entity"
	"github.com/usamaroman/faas_demo/price_service/internal/repo/repoerrors"

	"github.com/Masterminds/squirrel"
	"github.com/jackc/pgx/v5"
)

type Repo struct {
	*postgresql.Postgres
}

func NewRepo(pg *postgresql.Postgres) *Repo {
	return &Repo{
		Postgres: pg,
	}
}

func (r *Repo) Create(ctx context.Context, body *entity.Tariff) (*entity.Tariff, error) {
	q, args, err := r.Builder.Insert("tariffs").
		Columns("name", "exec_price", "mem_price", "cpu_price", "cold_start_price_per_second").
		Values(body.Name, body.ExecPrice, body.MemPrice, body.CpuPrice, body.ColdStartPricePerSecond).
		Suffix("RETURNING id, exec_price, mem_price, cpu_price, cold_start_price_per_second, created_at, updated_at").
		ToSql()
	if err != nil {
		slog.Error("failed to make query", err.Error())
		return nil, err
	}

	slog.Debug("create tariff query", slog.String("query", q))

	if err := r.Pool.QueryRow(ctx, q, args...).Scan(
		&body.ID,
		&body.ExecPrice,
		&body.MemPrice,
		&body.CpuPrice,
		&body.ColdStartPricePerSecond,
		&body.CreatedAt,
		&body.UpdatedAt,
	); err != nil {
		slog.Error("failed to scan returning values after creating tariff", err.Error())
		return nil, err
	}

	return body, nil
}

func (r *Repo) GetByID(ctx context.Context, id int) (*entity.Tariff, error) {
	q, args, err := r.Builder.
		Select("*").
		From("tariffs").
		Where(squirrel.Eq{"id": id}).
		ToSql()

	if err != nil {
		slog.Error("failed to build query", err.Error())
		return nil, err
	}

	slog.Debug("get tariff by id query", slog.String("query", q))

	var tariff entity.Tariff
	if err := r.Pool.QueryRow(ctx, q, args...).Scan(
		&tariff.ID,
		&tariff.Name,
		&tariff.ExecPrice,
		&tariff.MemPrice,
		&tariff.CpuPrice,
		&tariff.ColdStartPricePerSecond,
		&tariff.CreatedAt,
		&tariff.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			slog.Error("no tariff found", slog.Any("id", id), err.Error())
			return nil, repoerrors.ErrNotFound
		}

		slog.Error("failed to scan tariff", err.Error())
		return nil, err
	}

	return &tariff, nil
}

func (r *Repo) GetAll(ctx context.Context, filters *entity.TariffFilters) ([]entity.Tariff, error) {
	qb := r.Builder.
		Select(
			"id",
			"name",
			"exec_price",
			"mem_price",
			"cpu_price",
			"cold_start_price_per_second",
			"created_at",
			"updated_at",
		).
		From("tariffs")

	q, args, err := qb.Limit(filters.Limit).
		Offset(filters.Offset).
		ToSql()

	if err != nil {
		slog.Error("failed to build query", err.Error())
		return nil, err
	}

	slog.Debug("get all tariffs query", slog.String("query", q))

	rows, err := r.Pool.Query(ctx, q, args...)
	if err != nil {
		slog.Error("failed to get tariffs from database", err.Error())
		return nil, err
	}
	defer rows.Close()

	var tariffs []entity.Tariff
	for rows.Next() {
		var t entity.Tariff
		if err := rows.Scan(
			&t.ID,
			&t.Name,
			&t.ExecPrice,
			&t.MemPrice,
			&t.CpuPrice,
			&t.ColdStartPricePerSecond,
			&t.CreatedAt,
			&t.UpdatedAt,
		); err != nil {
			slog.Error("failed to scan tariff row", err.Error())
			return nil, err
		}
		tariffs = append(tariffs, t)
	}

	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return tariffs, nil
}

func (r *Repo) UpdateByID(ctx context.Context, id int, updates *entity.Tariff) (*entity.Tariff, error) {
	builder := r.Builder.Update("tariffs")

	if updates.Name != "" {
		builder = builder.Set("name", updates.Name)
	}

	if updates.ExecPrice != 0 {
		builder = builder.Set("exec_price", updates.ExecPrice)
	}

	if updates.MemPrice != 0 {
		builder = builder.Set("mem_price", updates.MemPrice)
	}

	if updates.CpuPrice != 0 {
		builder = builder.Set("cpu_price", updates.CpuPrice)
	}

	if updates.ColdStartPricePerSecond != 0 {
		builder = builder.Set("cold_start_price_per_second", updates.ColdStartPricePerSecond)
	}

	q, args, err := builder.Where(squirrel.Eq{"id": id}).
		Suffix("RETURNING id, exec_price, mem_price, cpu_price, cold_start_price_per_second, created_at, updated_at").
		ToSql()
	if err != nil {
		slog.Error("failed to build SQL query", slog.Any("id", id), err.Error())
		return nil, err
	}

	slog.Debug("update tariff query", slog.String("query", q))

	if err := r.Pool.QueryRow(ctx, q, args...).Scan(
		&updates.ID,
		&updates.ExecPrice,
		&updates.MemPrice,
		&updates.CpuPrice,
		&updates.ColdStartPricePerSecond,
		&updates.CreatedAt,
		&updates.UpdatedAt,
	); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			slog.Error("no tariff for update", err.Error())
			return nil, repoerrors.ErrNotFound
		}

		slog.Error("failed to scan returning values after updating tariff", err.Error())
		return nil, err
	}

	slog.Debug("update rows successfuly")

	return updates, nil
}

func (r *Repo) DeleteByID(ctx context.Context, id int) error {
	q, args, err := r.Builder.Delete("tariffs").Where(squirrel.Eq{"id": id}).ToSql()
	if err != nil {
		slog.Error("failed to make query", err.Error())
		return err
	}

	slog.Debug("delete tariff by id query", slog.String("query", q))

	result, err := r.Pool.Exec(ctx, q, args...)
	if err != nil {
		slog.Error("failed to delete tariff by id", slog.Any("id", id), err.Error())
		return err
	}

	if result.RowsAffected() == 0 {
		slog.Error("no tariff found with the given ID to delete", slog.Any("id", id))
		return repoerrors.ErrNotFound
	}

	return nil
}
