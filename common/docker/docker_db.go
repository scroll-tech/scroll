package docker

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/api/types"
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
	*Cmd
}

// NewImgDB return postgres db img instance.
func NewImgDB(t *testing.T, image, password, dbName string, port int) ImgInstance {
	return &ImgDB{
		image:    image,
		name:     fmt.Sprintf("%s-%s_%d", image, dbName, port),
		password: password,
		dbName:   dbName,
		port:     port,
		Cmd:      NewCmd(t),
	}
}

// Start postgres db container.
func (i *ImgDB) Start() error {
	id := GetContainerID(i.name)
	if id != "" {
		return fmt.Errorf("container already exist, name: %s", i.name)
	}
	i.Cmd.RunCmd(i.prepare(), true)
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
	// check if container is running, stop the running container.
	id := GetContainerID(i.name)
	if id != "" {
		timeout := time.Second * 3
		if err := cli.ContainerStop(ctx, id, &timeout); err != nil {
			return err
		}
		i.id = id
	}
	// remove the stopped container.
	return cli.ContainerRemove(ctx, i.id, types.ContainerRemoveOptions{})
}

// Endpoint return the dsn.
func (i *ImgDB) Endpoint() string {
	if !i.running {
		return ""
	}
	return fmt.Sprintf("postgres://postgres:%s@localhost:%d/%s?sslmode=disable", i.password, i.port, i.dbName)
}

func (i *ImgDB) prepare() []string {
	cmd := []string{"docker", "run", "--name", i.name, "-p", fmt.Sprintf("%d:5432", i.port)}
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
	i.RegistFunc(keyword, func(buf string) {
		if strings.Contains(buf, keyword) {
			select {
			case okCh <- struct{}{}:
			default:
				return
			}
		}
	})
	defer i.UnRegistFunc(keyword)

	select {
	case <-okCh:
		time.Sleep(time.Millisecond * 1500)
		i.id = GetContainerID(i.name)
		return i.id != ""
	case <-time.NewTimer(time.Second * 10).C:
		return false
	}
}
