package docker

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/api/types"

	"scroll-tech/common/cmd"
	"scroll-tech/common/utils"
)

// ImgGeth the geth image manager include l1geth and l2geth.
type ImgGeth struct {
	image string
	name  string
	id    string

	volume   string
	ipcPath  string
	httpPort int
	wsPort   int

	running bool
	cmd     *cmd.Cmd
}

// NewImgGeth return geth img instance.
func NewImgGeth(image, volume, ipc string, hPort, wPort int) ImgInstance {
	img := &ImgGeth{
		image:    image,
		name:     fmt.Sprintf("%s-%d", image, time.Now().Nanosecond()),
		volume:   volume,
		ipcPath:  ipc,
		httpPort: hPort,
		wsPort:   wPort,
	}
	img.cmd = cmd.NewCmd(img.name, img.prepare()...)
	return img
}

// Start run image and check if it is running healthily.
func (i *ImgGeth) Start() error {
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

// Endpoint return the connection endpoint.
func (i *ImgGeth) Endpoint() string {
	if !i.running {
		return ""
	}
	switch true {
	case i.httpPort != 0:
		return fmt.Sprintf("http://127.0.0.1:%d", i.httpPort)
	case i.wsPort != 0:
		return fmt.Sprintf("ws://127.0.0.1:%d", i.wsPort)
	default:
		return i.ipcPath
	}
}

func (i *ImgGeth) isOk() bool {
	keyword := "WebSocket enabled"
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
		return i.id != ""
	case <-time.After(time.Second * 10):
		return false
	}
}

// Stop the docker container.
func (i *ImgGeth) Stop() error {
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

func (i *ImgGeth) prepare() []string {
	cmds := []string{"docker", "run", "--name", i.name}
	var ports []string
	if i.httpPort != 0 {
		ports = append(ports, []string{"-p", strconv.Itoa(i.httpPort) + ":8545"}...)
	}
	if i.wsPort != 0 {
		ports = append(ports, []string{"-p", strconv.Itoa(i.wsPort) + ":8546"}...)
	}

	var envs []string
	if i.ipcPath != "" {
		envs = append(envs, []string{"-e", fmt.Sprintf("IPC_PATH=%s", i.ipcPath)}...)
	}

	if i.volume != "" {
		cmds = append(cmds, []string{"-v", fmt.Sprintf("%s:%s", i.volume, i.volume)}...)
	}

	cmds = append(cmds, ports...)
	cmds = append(cmds, envs...)

	return append(cmds, i.image)
}
