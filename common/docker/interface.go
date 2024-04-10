package docker

import (
	"context"
	"math/big"

	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
)

var (
	cli *client.Client
)

func init() {
	var err error
	cli, err = client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		panic(err)
	}
	cli.NegotiateAPIVersion(context.Background())
}

// ImgInstance is an img instance.
type ImgInstance interface {
	Start() error
	Stop() error
	Endpoint() string
	IsRunning() bool
}

// GethImgInstance based on ImgInstance and add chainID interface.
type GethImgInstance interface {
	ImgInstance
	ChainID() *big.Int
}

// GetContainerID returns the ID of Container.
func GetContainerID(name string) string {
	filter := filters.NewArgs()
	filter.Add("name", name)
	lst, _ := cli.ContainerList(context.Background(), container.ListOptions{
		Filters: filter,
	})
	if len(lst) > 0 {
		return lst[0].ID
	}
	return ""
}
