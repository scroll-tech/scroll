package l1

import (
	"testing"

	"github.com/stretchr/testify/assert"

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

	m.Run()

	base.Free()
}

func setupEnv(t *testing.T) {
	// Load config.
	var err error
	cfg, err = config.NewConfig("../config.json")
	assert.NoError(t, err)
	base.RunImages(t)

	cfg.L2Config.RelayerConfig.SenderConfig.Endpoint = base.L1gethImg.Endpoint()
	cfg.L1Config.RelayerConfig.SenderConfig.Endpoint = base.L2gethImg.Endpoint()
	cfg.DBConfig = base.DBConfig
}

func TestL1(t *testing.T) {
	setupEnv(t)

	t.Run("testCreateNewL1Relayer", testCreateNewL1Relayer)
	t.Run("testStartWatcher", testStartWatcher)
}
