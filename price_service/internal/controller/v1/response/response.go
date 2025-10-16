package response

import "github.com/usamaroman/faas_demo/price_service/internal/entity"

type GetAllTariffs struct {
	Tariffs []entity.Tariff `json:"tariffs"`
}
