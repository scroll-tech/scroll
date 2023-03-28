package l1

import (
	"testing"

	"scroll-tech/common/docker"

	"scroll-tech/bridge/config"
)

var (
	// config
	cfg *config.Config

	// docker consider handler.
	base *docker.App
)

func TestMain(m *testing.M) {
	base = docker.NewDockerApp()
	var err error
	cfg, err = config.NewConfig("../config.json")
	if err != nil {
		panic(err)
	}
	cfg.L2Config.RelayerConfig.SenderConfig.Endpoint = base.L1gethImg.Endpoint()
	cfg.L1Config.RelayerConfig.SenderConfig.Endpoint = base.L2gethImg.Endpoint()
	cfg.DBConfig = base.DBConfig

	m.Run()

	base.Free()
}
