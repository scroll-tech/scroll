package docker

import (
	"context"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
)

// ImgInstance is an img instance.
type ImgInstance interface {
	Start() error
	Stop() error
	Endpoint() string
}

// GetContainerID returns the ID of Container.
func GetContainerID(name string) string {
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
