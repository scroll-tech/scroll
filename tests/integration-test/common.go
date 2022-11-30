package integration

import (
	"context"
	"crypto/ecdsa"
	"golang.org/x/sync/errgroup"
	"math/big"
	"strconv"
	"testing"

	"github.com/scroll-tech/go-ethereum/common"
	"github.com/stretchr/testify/assert"

	_ "scroll-tech/bridge/cmd/app"
	"scroll-tech/bridge/config"
	"scroll-tech/bridge/sender"
	"scroll-tech/common/docker"
	_ "scroll-tech/coordinator/cmd/app"
	"scroll-tech/database"
	_ "scroll-tech/database/cmd/app"
	_ "scroll-tech/roller/cmd/app"
)

var (
	l1gethImg docker.ImgInstance
	l2gethImg docker.ImgInstance
	dbImg     docker.ImgInstance
	l2db      database.OrmFactory
)

func setupEnv(t *testing.T) {
	l1gethImg = docker.NewTestL1Docker(t)
	l2gethImg = docker.NewTestL2Docker(t)
	dbImg = docker.NewTestDBDocker(t, "postgres")

	// Create db handler and reset db.
	var err error
	l2db, err = database.NewOrmFactory(&database.DBConfig{
		DriverName: "postgres",
		DSN:        dbImg.Endpoint(),
	})
	assert.NoError(t, err)
}

func free(t *testing.T) {
	assert.NoError(t, l1gethImg.Stop())
	assert.NoError(t, l2gethImg.Stop())
	assert.NoError(t, l2db.Close())
	assert.NoError(t, dbImg.Stop())
}

func runBridgeApp(t *testing.T, args ...string) *docker.Cmd {
	cmd := docker.NewCmd(t)
	args = append(args, "--log.debug", "--config", "../../bridge/config.json")
	if l1gethImg != nil {
		args = append(args, "--l1.endpoint", l1gethImg.Endpoint())
	}
	if l2gethImg != nil {
		args = append(args, "--l2.endpoint", l2gethImg.Endpoint())
	}
	if dbImg != nil {
		args = append(args, "--db.dsn", dbImg.Endpoint())
	}
	// start process
	cmd.Run("bridge-test", args...)
	return cmd
}

func runCoordinatorApp(t *testing.T, args ...string) *docker.Cmd {
	cmd := docker.NewCmd(t)
	args = append(args, "--log.debug", "--config", "../../coordinator/config.json")
	if dbImg != nil {
		args = append(args, "--db.dsn", dbImg.Endpoint())
	}
	// start process
	cmd.Run("coordinator-test", args...)
	return cmd
}

func runDBCliApp(t *testing.T, args ...string) *docker.Cmd {
	cmd := docker.NewCmd(t)
	args = append(args, "--config", "../../database/config.json")
	if dbImg != nil {
		args = append(args, "--db.dsn", dbImg.Endpoint())
	}
	cmd.Run("db_cli-test", args...)
	return cmd
}

func runRollerApp(t *testing.T, args ...string) *docker.Cmd {
	cmd := docker.NewCmd(t)
	args = append(args, "--log.debug", "--config", "../../roller/config.toml")
	cmd.Run("roller-test", args...)
	return cmd
}

func runSender(t *testing.T, cfg *config.SenderConfig, privs []*ecdsa.PrivateKey, to common.Address, data []byte) *sender.Sender {
	newSender, err := sender.NewSender(context.Background(), cfg, privs)
	assert.NoError(t, err)
	eg := errgroup.Group{}
	for i := 0; i < newSender.NumberOfAccounts(); i++ {
		idx := i
		eg.Go(func() error {
			_, err = newSender.SendTransaction(strconv.Itoa(idx), &to, big.NewInt(1), nil)
			if err == nil {
				<-newSender.ConfirmChan()
			}
			return err
		})
	}
	assert.NoError(t, eg.Wait())
	return newSender
}
