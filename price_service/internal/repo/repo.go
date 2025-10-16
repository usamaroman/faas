package repo

import (
	"context"

	"github.com/usamaroman/faas_demo/price_service/internal/entity"
	"github.com/usamaroman/faas_demo/price_service/internal/repo/tariff"
	"github.com/usamaroman/faas_demo/pkg/postgresql"
)

type Tariff interface {
	Create(ctx context.Context, body *entity.Tariff) (*entity.Tariff, error)
	GetByID(ctx context.Context, id int) (*entity.Tariff, error)
	GetAll(ctx context.Context, filters *entity.TariffFilters) ([]entity.Tariff, error)
	UpdateByID(ctx context.Context, id int, updates *entity.Tariff) (*entity.Tariff, error)
	DeleteByID(ctx context.Context, id int) error
}

type Repositories struct {
	Tariff
}

func NewRepositories(pg *postgresql.Postgres) *Repositories {
	return &Repositories{
		Tariff: tariff.NewRepo(pg),
	}
}
