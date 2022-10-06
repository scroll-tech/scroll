package docker

import (
	"context"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
)

type ImgInstance interface {
	Start() error
	Stop() error
	Endpoint() string
}

func getContainerID(name string) string {
	filter := filters.NewArgs()
	filter.Add("name", name)
	lst, _ := cli.ContainerList(context.Background(), types.ContainerListOptions{
		Filters: filter,
	})
	if len(lst) > 0 {
		return lst[0].Names[0]
	}
	return ""
}
