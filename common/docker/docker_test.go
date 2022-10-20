package docker

import (
	"context"
	"testing"

	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/stretchr/testify/assert"
)

func TestL1Geth(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	img := NewImgGeth(t, "scroll_l1geth", "", "", 8535, 0)
	assert.NoError(t, img.Start())
	defer img.Stop()

	client, err := ethclient.Dial(img.Endpoint())
	assert.NoError(t, err)

	chainID, err := client.ChainID(ctx)
	assert.NoError(t, err)
	t.Logf("chainId: %s", chainID.String())
}

func TestL2Geth(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	img := NewImgGeth(t, "scroll_l2geth", "", "", 8535, 0)
	assert.NoError(t, img.Start())
	defer img.Stop()

	client, err := ethclient.Dial(img.Endpoint())
	assert.NoError(t, err)

	chainID, err := client.ChainID(ctx)
	assert.NoError(t, err)
	t.Logf("chainId: %s", chainID.String())
}
