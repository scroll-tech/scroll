package docker

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"

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
}

// NewImgDB return postgres db img instance.
func NewImgDB(image, password, dbName string, port int) ImgInstance {
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
	id := GetContainerID(i.name)
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
	if !i.running {
		return nil
	}
	i.running = false

	ctx := context.Background()
	// stop the running container.
	if i.id == "" {
		i.id = GetContainerID(i.name)
	}

	timeoutSec := 3
	timeout := container.StopOptions{
		Timeout: &timeoutSec,
	}
	if err := cli.ContainerStop(ctx, i.id, timeout); err != nil {
		return err
	}
	// remove the stopped container.
	return cli.ContainerRemove(ctx, i.id, types.ContainerRemoveOptions{})
}

// Endpoint return the dsn.
func (i *ImgDB) Endpoint() string {
	return fmt.Sprintf("postgres://postgres:%s@localhost:%d/%s?sslmode=disable", i.password, i.port, i.dbName)
}

// IsRunning returns docker container's running status.
func (i *ImgDB) IsRunning() bool {
	return i.running
}

// OpenLog open log.
func (i *ImgDB) OpenLog(open bool) {
	i.cmd.OpenLog(open)
}

func (i *ImgDB) prepare() []string {
	cmd := []string{"docker", "run", "--rm", "--name", i.name, "-p", fmt.Sprintf("%d:5432", i.port)}
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
