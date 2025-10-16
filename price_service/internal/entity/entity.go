package entity

import "time"

type Tariff struct {
	ID        int       `db:"id"`
	Name      string    `db:"name"`
	ExecPrice float64   `db:"exec_price"`
	MemPrice  float64   `db:"mem_price"`
	CpuPrice  float64   `db:"cpu_price"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}

type TariffFilters struct {
	Limit  uint64
	Offset uint64
}
