package repository

import "time"

type Tenant struct {
	ID           int64
	Name         string
	ContactEmail *string
	CreatedAt    time.Time
}

type Function struct {
	ID          int64
	TenantID    int64
	Name        string
	Description *string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type FunctionVersion struct {
	ID          int64
	FunctionID  int64
	Version     string
	Tag         *string
	DockerImage string
	CommitHash  *string
	BuildDate   time.Time
	Changelog   *string
	Active      bool
	CreatedAt   time.Time
}

type Deployment struct {
	ID                int64
	FunctionVersionID int64
	InstanceID        *string
	Status            string
	Replicas          int32
	CreatedAt         time.Time
	UpdatedAt         time.Time
}
