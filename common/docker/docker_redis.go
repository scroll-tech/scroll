package docker

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/api/types"

	"scroll-tech/common/cmd"
	"scroll-tech/common/utils"
)

// ImgRedis the redis image.
type ImgRedis struct {
	image string
	name  string
	id    string

	port    int
	running bool
	cmd     *cmd.Cmd
}

// NewImgRedis return redis img instance.
func NewImgRedis(t *testing.T, image string, port int) ImgInstance {
	img := &ImgRedis{
		image: image,
		name:  fmt.Sprintf("%s-%d", image, time.Now().Nanosecond()),
		port:  port,
	}
	img.cmd = cmd.NewCmd(t, img.name, img.prepare()...)
	return img
}

// Start run image and check if it is running healthily.
func (r *ImgRedis) Start() error {
	id := GetContainerID(r.name)
	if id != "" {
		return fmt.Errorf("container already exist, name: %s", r.name)
	}
	// Add check status function.
	keyword := "Ready to accept connections"
	okCh := make(chan struct{}, 1)
	r.cmd.RegistFunc(keyword, func(buf string) {
		if strings.Contains(buf, keyword) {
			select {
			case okCh <- struct{}{}:
			default:
				return
			}
		}
	})
	defer r.cmd.UnRegistFunc(keyword)

	// Start redis.
	r.cmd.RunCmd(true)

	// Wait result of keyword.
	select {
	case <-okCh:
		utils.TryTimes(10, func() bool {
			r.id = GetContainerID(r.name)
			return r.id != ""
		})
	case <-time.After(time.Second * 10):
	}

	// Set redis status.
	r.running = r.id != ""
	if !r.running {
		_ = r.Stop()
		return fmt.Errorf("failed to start image: %s", r.image)
	}
	return nil
}

// Stop the docker container.
func (r *ImgRedis) Stop() error {
	if !r.running {
		return nil
	}
	r.running = false
	ctx := context.Background()
	id := GetContainerID(r.name)
	if id != "" {
		timeout := time.Second * 3
		if err := cli.ContainerStop(ctx, id, &timeout); err != nil {
			return err
		}
		r.id = id
	}
	// remove the stopped container.
	return cli.ContainerRemove(ctx, r.id, types.ContainerRemoveOptions{})
}

// Endpoint return the connection endpoint.
func (r *ImgRedis) Endpoint() string {
	if !r.running {
		return ""
	}
	port := 6379
	if r.port != 0 {
		port = r.port
	}
	return fmt.Sprintf("redis://default:redistest@localhost:%d/0", port)
}

// docker run --name redis-xxx -p randomport:6379 --requirepass "redistest" redis
func (r *ImgRedis) prepare() []string {
	cmds := []string{"docker", "run", "--name", r.name}
	var ports []string
	if r.port != 0 {
		ports = append(ports, []string{"-p", strconv.Itoa(r.port) + ":6379"}...)
	}
	return append(append(cmds, ports...), r.image)
}
