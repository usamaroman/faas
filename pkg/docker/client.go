package docker

import (
	"log/slog"

	"github.com/docker/docker/client"
)

func NewClient(host string) (*client.Client, error) {
	cli, err := client.NewClientWithOpts(
		client.WithHost(host),
		client.WithAPIVersionNegotiation())
	if err != nil {
		slog.Error("failed to make docker client", slog.String("error", err.Error()))
		return nil, err
	}

	return cli, nil
}
