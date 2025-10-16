package service

import (
	"context"
	"errors"
	"log/slog"

	"github.com/usamaroman/faas_demo/price_service/internal/entity"
	"github.com/usamaroman/faas_demo/price_service/internal/repo"
	"github.com/usamaroman/faas_demo/price_service/internal/repo/repoerrors"
)

type TariffService struct {
	tariffRepo repo.Tariff
}

type TariffInput struct {
	Name      string  `json:"name"`
	ExecPrice float64 `json:"exec_price"`
	MemPrice  float64 `json:"mem_price"`
	CpuPrice  float64 `json:"cpu_price"`
}

func NewTariffService(tariffRepo repo.Tariff) *TariffService {
	slog.Debug("component", "tariff service")

	return &TariffService{
		tariffRepo: tariffRepo,
	}
}

func (s *TariffService) Create(ctx context.Context, body *TariffInput) (*entity.Tariff, error) {
	tariff, err := s.tariffRepo.Create(ctx, &entity.Tariff{
		Name:      body.Name,
		ExecPrice: body.ExecPrice,
		MemPrice:  body.MemPrice,
		CpuPrice:  body.CpuPrice,
	})
	if err != nil {
		slog.Error("failed to create tariff", err.Error())
		return nil, err
	}

	return tariff, nil
}

func (s *TariffService) GetByID(ctx context.Context, id int) (*entity.Tariff, error) {
	tariff, err := s.tariffRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, repoerrors.ErrNotFound) {
			return nil, ErrTariffNotFound
		}

		return nil, err
	}

	return tariff, nil
}

func (s *TariffService) GetAll(ctx context.Context, filters *entity.TariffFilters) ([]entity.Tariff, error) {
	return s.tariffRepo.GetAll(ctx, filters)
}

func (s *TariffService) UpdateByID(ctx context.Context, id int, updates *TariffInput) (*entity.Tariff, error) {
	updatedTariff, err := s.tariffRepo.UpdateByID(ctx, id, &entity.Tariff{
		Name:      updates.Name,
		ExecPrice: updates.ExecPrice,
		MemPrice:  updates.MemPrice,
		CpuPrice:  updates.CpuPrice,
	})
	if err != nil {
		if errors.Is(err, repoerrors.ErrNotFound) {
			return nil, ErrTariffNotFound
		}

		return nil, err
	}

	return updatedTariff, nil
}

func (s *TariffService) DeleteByID(ctx context.Context, id int) error {
	err := s.tariffRepo.DeleteByID(ctx, id)
	if err != nil {
		if errors.Is(err, repoerrors.ErrNotFound) {
			return ErrTariffNotFound
		}

		return err
	}

	return nil
}
