package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/api/types"
)

// ImgGeth the geth image manager include l1geth and l2geth.
type ImgGeth struct {
	image string
	name  string
	id    string

	volume    string
	ipcPath   string
	httpPort  int
	wsPort    int
	contracts Contract

	running bool
	*Cmd
}
type Contract struct {
	L2 L2Contracts
	L1 L1Contracts
}

type Proxy struct {
	implementation string `json:"implementation"`
	proxy          string `json:"proxy"`
}

type L1Contracts struct {
}

type L2Contracts struct {
	ProxyAdmin                 string `json:"ProxyAdmin"`
	WETH                       string `json:"WETH"`
	Whitelist                  string `json:"Whitelist"`
	ScrollStandardERC20        string `json:"ScrollStandardERC20"`
	ScrollStandardERC20Factory string `json:"ScrollStandardERC20Factory"`
	L2ScrollMessenger          string `json:"L2ScrollMessenger"`
	L2GatewayRouter            Proxy  `json:"L2GatewayRouter"`
	L2StandardERC20Gateway     Proxy  `json:"L2StandardERC20Gateway"`
	L2WETHGateway              Proxy  `json:"L2WETHGateway"`
}

// NewImgGeth return geth img instance.
func NewImgGeth(t *testing.T, image, volume, ipc string, hPort, wPort int) ImgInstance {
	return &ImgGeth{
		image:    image,
		name:     fmt.Sprintf("%s-%d", image, time.Now().Nanosecond()),
		volume:   volume,
		ipcPath:  ipc,
		httpPort: hPort,
		wsPort:   wPort,
		Cmd:      NewCmd(t),
	}
}

// Start run image and check if it is running healthily.
func (i *ImgGeth) Start() error {
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
		i.id = GetContainerID(i.name)
		return i.id != ""
	case <-time.NewTimer(time.Second * 10).C:
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

// GetDeployments returns pre_deployed contract address
func (i *ImgGeth) GetDeployments() map[string]string {
	if !i.running {
		return nil
	}
	cmds := []string{"docker", "copy", "--name", i.name}
	cmds = append(cmds, []string{"/deployments/l2geth.json", "l2geth.json"}...)
	//cmds = append(cmds, []string{"/deployments/l1geth.json", "l1geth.json"}...)

	i.Cmd.RunCmd(cmds, false)
	plan, _ := ioutil.ReadFile("l2geth.json")

	err := json.Unmarshal(plan, &data)

	return nil

}
