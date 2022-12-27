package native

import (
	"context"
	"errors"
	"math/big"
	"math/rand"
	"sync"
	"time"

	"github.com/scroll-tech/go-ethereum/accounts/abi/bind"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/scroll-tech/go-ethereum/log"

	"tool/accounts"
	"tool/utils"
)

type Native struct {
	batchCh chan struct{}

	ctx context.Context

	accounts *accounts.Accounts
	client   *ethclient.Client
}

func NewNative(ctx context.Context, accounts *accounts.Accounts, client *ethclient.Client) *Native {
	return &Native{
		ctx:      ctx,
		batchCh:  make(chan struct{}, 1),
		accounts: accounts,
		client:   client,
	}
}

func (n *Native) Transfer(to common.Address, balance *big.Int) (*types.Transaction, error) {
	root := n.accounts.Root
	tx, err := n.createSignedTx(root, to, balance)
	if err != nil {
		return nil, err
	}
	err = n.client.SendTransaction(n.ctx, tx)
	if err != nil {
		return nil, err
	}
	utils.WaitPendingTx(n.ctx, n.client, tx.Hash())
	return tx, nil
}

func (n *Native) BalanceOf(account common.Address) (*big.Int, error) {
	return n.client.BalanceAt(n.ctx, account, big.NewInt(-1))
}

func (n *Native) Pressure(count, batch, tps int) error {
	select {
	case n.batchCh <- struct{}{}:
		batch = utils.MinInt(utils.MinInt(utils.MinInt(batch, count), tps), len(accounts.AddrPrivs))
		sleep := time.Millisecond * 1000 * time.Duration(batch) / time.Duration(tps)
		go n.sendTxs(count, batch, sleep)
	default:
		return errors.New("current batch task is not finished")
	}
	return nil
}

func (c *Native) sendTxs(count, batch int, sleep time.Duration) {
	defer func() { <-c.batchCh }()
	start := time.Now()

	var wait sync.WaitGroup
	for j := 0; j < batch; j++ {
		wait.Add(1)
		go func() {
			defer wait.Done()
			loopCount := count / batch
			for i := 0; i < loopCount; i++ {
				auth := c.accounts.GetAccount()
				signedTx, err := c.createSignedTx(
					auth,
					common.BigToAddress(big.NewInt(int64(rand.Intn(1000000)))),
					big.NewInt(100),
				)
				if err = c.client.SendTransaction(c.ctx, signedTx); err != nil {
					log.Error("Failed to send tx", "err", err)
				}
				c.accounts.SetAccount(auth)
				if i < loopCount-1 {
					time.Sleep(sleep)
				}
			}
		}()
	}
	wait.Wait()

	log.Info("Send native txs finished", "txs sum", count, "time used(ms)", time.Now().Sub(start).Milliseconds())

}

func (n *Native) createSignedTx(auth *bind.TransactOpts, to common.Address, value *big.Int) (signedTx *types.Transaction, err error) {
	nonce, err := n.client.PendingNonceAt(n.ctx, auth.From)
	if err != nil {
		return nil, err
	}
	gasprice, _ := n.client.SuggestGasPrice(n.ctx)

	return auth.Signer(auth.From, types.NewTx(&types.LegacyTx{
		Nonce:    nonce,
		To:       &to,
		Value:    value,
		Gas:      5000000,
		GasPrice: gasprice,
	}))
}
