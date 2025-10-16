package service

import (
	"context"

	"github.com/usamaroman/faas_demo/price_service/internal/entity"
	"github.com/usamaroman/faas_demo/price_service/internal/repo"
)

type Tariff interface {
	Create(ctx context.Context, body *TariffInput) (*entity.Tariff, error)
	GetByID(ctx context.Context, id int) (*entity.Tariff, error)
	GetAll(ctx context.Context, filters *entity.TariffFilters) ([]entity.Tariff, error)
	UpdateByID(ctx context.Context, id int, updates *TariffInput) (*entity.Tariff, error)
	DeleteByID(ctx context.Context, id int) error
}

type Dependencies struct {
	Repos *repo.Repositories
}

type Services struct {
	Tariff Tariff
}

func NewServices(deps *Dependencies) *Services {
	services := &Services{
		Tariff: NewTariffService(deps.Repos.Tariff),
	}

	return services
}
