package docker

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/pkg/stdcopy"

	"scroll-tech/common/cmd"
	"scroll-tech/common/utils"
)

// ImgDB the postgres image manager.
type ImgDB struct {
	image string
	name  string
	id    string

	dbName   string
	port     int
	password string

	running bool
	cmd     *cmd.Cmd

	closer io.Closer
}

// NewImgDB return postgres db img instance.
func NewImgDB(password, dbName string, port int) ImgInstance {
	image := "postgres"
	img := &ImgDB{
		image:    image,
		name:     fmt.Sprintf("%s-%s_%d", image, dbName, port),
		password: password,
		dbName:   dbName,
		port:     port,
	}
	img.cmd = cmd.NewCmd(img.name, img.prepare()...)

	return img
}

// Start postgres db container.
func (i *ImgDB) Start() error {
	// If exist exited container, handle it's id and try to reuse it.
	id, exist := getSpecifiedContainer(i.image, "postgres-test_db_")
	if exist {
		closer, port, err := startContainer(i.id, i.cmd)
		if err != nil { // If start a exist container failed, log error message then create and start a new one.
			fmt.Printf("failed to start a exist container, id: %s, err: %v\n", i.id, err)
			i.id = ""
		} else {
			i.running = true
			i.port = int(port)
			i.closer = closer
			i.id = id
			return nil
		}
	}

	// Create and start a new container.
	id = GetContainerID(i.name)
	if id != "" {
		return fmt.Errorf("container already exist, name: %s", i.name)
	}
	i.running = i.isOk()
	if !i.running {
		_ = i.Stop()
		return fmt.Errorf("failed to start image: %s", i.image)
	}
	return nil
}

// Stop the container.
func (i *ImgDB) Stop() error {
	if i.closer != nil {
		_ = i.closer.Close()
	}

	if !i.running {
		return nil
	}
	i.running = false

	ctx := context.Background()
	// stop the running container.
	if i.id == "" {
		i.id = GetContainerID(i.name)
	}
	timeout := time.Second * 3
	return cli.ContainerStop(ctx, i.id, &timeout)
}

// Endpoint return the dsn.
func (i *ImgDB) Endpoint() string {
	return fmt.Sprintf("postgres://postgres:%s@localhost:%d/%s?sslmode=disable", i.password, i.port, i.dbName)
}

// IsRunning returns docker container's running status.
func (i *ImgDB) IsRunning() bool {
	return i.running
}

func (i *ImgDB) prepare() []string {
	cmd := []string{"docker", "run" /*"--rm",*/, "--name", i.name, "-p", fmt.Sprintf("%d:5432", i.port)}
	envs := []string{
		"-e", "POSTGRES_PASSWORD=" + i.password,
		"-e", fmt.Sprintf("POSTGRES_DB=%s", i.dbName),
	}

	cmd = append(cmd, envs...)
	return append(cmd, i.image)
}

func (i *ImgDB) isOk() bool {
	keyword := "database system is ready to accept connections"
	okCh := make(chan struct{}, 1)
	i.cmd.RegistFunc(keyword, func(buf string) {
		if strings.Contains(buf, keyword) {
			select {
			case okCh <- struct{}{}:
			default:
				return
			}
		}
	})
	defer i.cmd.UnRegistFunc(keyword)
	// Start cmd in parallel.
	i.cmd.RunCmd(true)

	select {
	case <-okCh:
		utils.TryTimes(20, func() bool {
			i.id = GetContainerID(i.name)
			return i.id != ""
		})
	case err := <-i.cmd.ErrChan:
		if err != nil {
			fmt.Printf("failed to start %s, err: %v\n", i.name, err)
		}
	case <-time.After(time.Second * 20):
		return false
	}
	return i.id != ""
}

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
			!strings.Contains(container.Names[0], keyword) ||
			container.State == string(runningState) {
			continue
		}
		ID = container.ID
		// If the container is not running, just choose it.
		if container.State == string(exitedState) {
			break
		}
	}
	return ID, ID != ""
}

func startContainer(id string, stdout io.Writer) (io.Closer, uint16, error) {
	// Get container is used for checking state.
	ct, err := getContainerByID(id)
	if err != nil {
		return nil, 0, err
	}

	if ct.State != string(runningState) {
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

	//
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
