package docker

import (
	"context"
	"fmt"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/pkg/stdcopy"
	"io"
	"strings"
	"time"
)

// Container state.
type dockerState string

var (
	runningState dockerState = "running"
	exitedState  dockerState = "exited"
)

func getSpecifiedContainer(image string, keyword string) (string, bool) {
	containers, _ := cli.ContainerList(context.Background(), types.ContainerListOptions{
		All: true,
	})
	var ID string
	for _, container := range containers {
		// Just use the exited container.
		if container.Image != image ||
			!strings.Contains(container.Names[0], keyword) {
			continue
		}

		// If container state is running choose and return.
		if container.State == string(runningState) {
			return container.ID, true
		}
		// If state is exited just set id.
		if container.State == string(exitedState) {
			ID = container.ID
		}
	}

	return ID, ID != ""
}

func startContainer(id string, stdout io.Writer) (io.Closer, uint16, error) {
	// Get container again, check state and handle public port.
	ct, err := getContainerByID(id)
	if err != nil {
		return nil, 0, err
	}

	// start the exited container.
	if ct.State == string(exitedState) {
		// Start the exist container.
		err = cli.ContainerStart(context.Background(), id, types.ContainerStartOptions{})
		if err != nil {
			return nil, 0, err
		}
	}

	// Get container again, check state and handle public port.
	ct, err = getContainerByID(id)
	if err != nil {
		return nil, 0, err
	}
	if ct.State != string(runningState) {
		return nil, 0, fmt.Errorf("container state is not running, id: %s", id)
	}

	readCloser, err := cli.ContainerLogs(context.Background(), id, types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     true,
	})

	// Handle stdout.
	errCh := make(chan error, 1)
	go func() {
		_, err2 := stdcopy.StdCopy(stdout, stdout, readCloser)
		errCh <- err2
	}()
	select {
	case <-time.After(time.Millisecond * 200):
	case err = <-errCh:
	}
	return readCloser, ct.Ports[0].PublicPort, err
}

func getContainerByID(id string) (*types.Container, error) {
	filter := filters.NewArgs()
	filter.Add("id", id)
	lst, err := cli.ContainerList(context.Background(), types.ContainerListOptions{
		All:     true,
		Filters: filter,
	})

	if len(lst) == 1 {
		return &lst[0], err
	}
	return nil, fmt.Errorf("the container is not exist, id: %s", id)
}

func InitConfig(name string) (string, error) {
	res, err := cli.ConfigCreate(context.Background(), swarm.ConfigSpec{
		Annotations: swarm.Annotations{
			Name:   name,
			Labels: map[string]string{},
		},
		Data: []byte("postgres config for " + name),
	})
	if err != nil {
		return "", err
	}
	return res.ID, nil
}

func SetConfigLabel(id string, keyword string, setOrRm bool) (int, error) {
	filter := filters.NewArgs()
	filter.Add("id", id)
	cfgs, err := cli.ConfigList(context.Background(), types.ConfigListOptions{
		Filters: filter,
	})
	if err != nil || len(cfgs) == 0 {
		return 0, err
	}
	cfg := cfgs[0]
	if setOrRm {
		cfg.Spec.Labels[keyword] = ""
	} else {
		delete(cfg.Spec.Labels, keyword)
	}

	// Update config and return count of labels.
	return len(cfg.Spec.Labels), cli.ConfigUpdate(context.Background(), id, cfg.Version, cfg.Spec)
}
