package request

type CreateTariff struct {
	Name      string  `json:"name" validate:"required"`
	ExecPrice float64 `json:"exec_price" validate:"required,gte=0"`
	MemPrice  float64 `json:"mem_price" validate:"required,gte=0"`
	CpuPrice  float64 `json:"cpu_price" validate:"required,gte=0"`
}

type UpdateTariff struct {
	Name      string  `json:"name"`
	ExecPrice float64 `json:"exec_price" validate:"gte=0"`
	MemPrice  float64 `json:"mem_price" validate:"gte=0"`
	CpuPrice  float64 `json:"cpu_price" validate:"gte=0"`
}
