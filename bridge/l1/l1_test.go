package l1_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"scroll-tech/common/docker"
	"scroll-tech/common/viper"
)

var (
	// config
	vp *viper.Viper

	// docker consider handler.
	l1gethImg docker.ImgInstance
	l2gethImg docker.ImgInstance
	dbImg     docker.ImgInstance
)

func setupEnv(t *testing.T) {
	// Load config.
	var err error
	vp, err = viper.NewViper("../config.json", true)
	assert.NoError(t, err)

	// Create l1geth container.
	l1gethImg = docker.NewTestL1Docker(t)
	vp.Set("l2_config.relayer_config.sender_config.endpoint", l1gethImg.Endpoint())
	vp.Set("l1_config.endpoint", l1gethImg.Endpoint())

	// Create l2geth container.
	l2gethImg = docker.NewTestL2Docker(t)
	vp.Set("l1_config.relayer_config.sender_config.endpoint", l2gethImg.Endpoint())
	vp.Set("l2_config.endpoint", l2gethImg.Endpoint())

	// Create db container.
	driverName := vp.Sub("db_config").GetString("driver_name")
	dbImg = docker.NewTestDBDocker(t, driverName)
	vp.Set("db_config.dsn", dbImg.Endpoint())
}

func free(t *testing.T) {
	if dbImg != nil {
		assert.NoError(t, dbImg.Stop())
	}
	if l1gethImg != nil {
		assert.NoError(t, l1gethImg.Stop())
	}
	if l2gethImg != nil {
		assert.NoError(t, l2gethImg.Stop())
	}
}

func TestL1(t *testing.T) {
	setupEnv(t)

	t.Run("testCreateNewL1Relayer", testCreateNewL1Relayer)
	t.Run("testStartWatcher", testStartWatcher)

	t.Cleanup(func() {
		free(t)
	})
}
