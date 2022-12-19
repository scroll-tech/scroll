package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/api/types"

	"scroll-tech/common/utils"
)

// ImgGeth the geth image manager include l1geth and l2geth.
type ImgGeth struct {
	image string
	name  string
	id    string

	volume      string
	ipcPath     string
	httpPort    int
	wsPort      int
	addressFile *AddressFile

	running bool
	*Cmd
	t *testing.T
}

// AddressFile stores l1/l2 contract address
type AddressFile struct {
	L2 *L2Contracts
	L1 *L1Contracts
}

// Proxy contains proxy address and implementation address
type Proxy struct {
	Implementation string `json:"implementation"`
	Proxy          string `json:"proxy"`
}

// L1Contracts stores pre-deployed contracts address of scroll_l1geth
type L1Contracts struct {
	ProxyAdmin             string `json:"ProxyAdmin"`
	ZKRollup               Proxy  `json:"ZKRollup"`
	L1ScrollMessenger      Proxy  `json:"L1ScrollMessenger"`
	L1GatewayRouter        Proxy  `json:"L1GatewayRouter"`
	L1StandardERC20Gateway Proxy  `json:"L1StandardERC20Gateway"`
	L1WETHGateway          Proxy  `json:"L1WETHGateway"`
}

// L2Contracts stores pre-deployed contracts address of scroll_l2geth
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
	img := &ImgGeth{
		image:       image,
		name:        fmt.Sprintf("%s-%d", image, time.Now().Nanosecond()),
		volume:      volume,
		ipcPath:     ipc,
		httpPort:    hPort,
		wsPort:      wPort,
		addressFile: nil,
		t:           t,
	}
	img.Cmd = NewCmd(img.t, img.name, img.prepare()...)
	return img
}

// Start run image and check if it is running healthily.
func (i *ImgGeth) Start() error {
	id := GetContainerID(i.name)
	if id != "" {
		return fmt.Errorf("container already exist, name: %s", i.name)
	}
	i.Cmd.RunCmd(true)
	i.running = i.isOk()
	if !i.running {
		_ = i.Stop()
		return fmt.Errorf("failed to start image: %s", i.image)
	}
	i.addressFile = i.getDeployments(i.t)
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
		utils.TryTimes(3, func() bool {
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

// GetAddressFile returnsd addressfile in imgGeth
func (i *ImgGeth) GetAddressFile() *AddressFile {
	return i.addressFile
}

// getDeployments returns pre_deployed contract address
func (i *ImgGeth) getDeployments(t *testing.T) *AddressFile {
	cmds := []string{"docker", "cp", i.name + ":/deployments/l2geth.json", "."}
	i.Cmd = NewCmd(t, i.name, cmds...)

	i.Cmd.RunCmd(false)
	L2addressFile, err := os.ReadFile("l2geth.json")
	if err != nil {
		i.Fatal(err)
		return nil
	}
	var l2data *L2Contracts
	err = json.Unmarshal(L2addressFile, &l2data)
	if err != nil {
		i.Fatal(err)
		l2data = nil
	}
	err = os.Remove("l2geth.json")
	if err != nil {
		i.Fatal(err)
		return nil
	}

	cmds = []string{"docker", "cp", i.name + ":/deployments/l1geth.json", "."}
	i.Cmd = NewCmd(t, i.name, cmds...)
	i.Cmd.RunCmd(false)
	L1addressFile, err := os.ReadFile("l1geth.json")
	if err != nil {
		i.Fatal(err)
		return nil
	}
	var l1data *L1Contracts
	err = json.Unmarshal(L1addressFile, &l1data)
	if err != nil {
		i.Fatal(err)
		l1data = nil
	}
	err = os.Remove("l1geth.json")
	if err != nil {
		i.Fatal(err)
		return nil
	}

	return &AddressFile{
		L1: l1data,
		L2: l2data,
	}

}
