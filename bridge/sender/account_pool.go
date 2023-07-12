package sender

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"

	"github.com/scroll-tech/go-ethereum/accounts/abi/bind"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/scroll-tech/go-ethereum/log"
)

type accountPool struct {
	client *ethclient.Client

	minBalance *big.Int
	accounts   map[common.Address]*bind.TransactOpts
	accsCh     chan *bind.TransactOpts
}

// newAccounts creates an accountPool instance.
func newAccountPool(ctx context.Context, minBalance *big.Int, client *ethclient.Client, privs []*ecdsa.PrivateKey) (*accountPool, error) {
	accs := &accountPool{
		client:     client,
		minBalance: minBalance,
		accounts:   make(map[common.Address]*bind.TransactOpts, len(privs)),
		accsCh:     make(chan *bind.TransactOpts, len(privs)+2),
	}

	// get chainID from client
	chainID, err := client.ChainID(ctx)
	if err != nil {
		return nil, err
	}

	for _, privStr := range privs {
		auth, err := bind.NewKeyedTransactorWithChainID(privStr, chainID)
		if err != nil {
			log.Error("failed to create account", "chainID", chainID.String(), "err", err)
			return nil, err
		}

		// Set pending nonce
		nonce, err := client.PendingNonceAt(ctx, auth.From)
		if err != nil {
			return nil, err
		}
		auth.Nonce = big.NewInt(int64(nonce))
		accs.accounts[auth.From] = auth
		accs.accsCh <- auth
	}

	return accs, accs.checkAndSetBalances(ctx)
}

// getAccount get auth from channel.
func (a *accountPool) getAccount() *bind.TransactOpts {
	select {
	case auth := <-a.accsCh:
		return auth
	default:
		return nil
	}
}

// releaseAccount set used auth into channel.
func (a *accountPool) releaseAccount(auth *bind.TransactOpts) {
	a.accsCh <- auth
}

// reSetNonce reset nonce if send signed tx failed.
func (a *accountPool) resetNonce(ctx context.Context, auth *bind.TransactOpts) {
	nonce, err := a.client.PendingNonceAt(ctx, auth.From)
	if err != nil {
		log.Warn("failed to reset nonce", "address", auth.From.String(), "err", err)
		return
	}
	auth.Nonce = big.NewInt(int64(nonce))
}

// checkAndSetBalance check balance and set min balance.
func (a *accountPool) checkAndSetBalances(ctx context.Context) error {
	var (
		root      *bind.TransactOpts
		maxBls    = big.NewInt(0)
		lostAuths []*bind.TransactOpts
	)

	for addr, auth := range a.accounts {
		bls, err := a.client.BalanceAt(ctx, addr, nil)
		if err != nil || bls.Cmp(a.minBalance) < 0 {
			if err != nil {
				log.Warn("failed to get balance", "address", addr.String(), "err", err)
				return err
			}
			lostAuths = append(lostAuths, auth)
			continue
		} else if bls.Cmp(maxBls) > 0 { // Find the biggest balance account.
			root, maxBls = auth, bls
		}
	}
	if root == nil {
		return fmt.Errorf("no account has enough balance")
	}
	if len(lostAuths) == 0 {
		return nil
	}

	var (
		tx  *types.Transaction
		err error
	)
	for _, auth := range lostAuths {
		tx, err = a.createSignedTx(root, &auth.From, a.minBalance)
		if err != nil {
			return err
		}
		err = a.client.SendTransaction(ctx, tx)
		if err != nil {
			log.Error("Failed to send balance to account", "err", err)
			return err
		}
		log.Debug("send balance to account", "account", auth.From.String(), "balance", a.minBalance.String())
	}
	// wait util mined
	if _, err = bind.WaitMined(ctx, a.client, tx); err != nil {
		return err
	}

	// Reset root's nonce.
	a.resetNonce(ctx, root)

	return nil
}

func (a *accountPool) createSignedTx(from *bind.TransactOpts, to *common.Address, value *big.Int) (*types.Transaction, error) {
	gasPrice, err := a.client.SuggestGasPrice(context.Background())
	if err != nil {
		return nil, err
	}
	gasPrice.Mul(gasPrice, big.NewInt(2))

	// Get pending nonce
	nonce, err := a.client.PendingNonceAt(context.Background(), from.From)
	if err != nil {
		return nil, err
	}

	tx := types.NewTx(&types.LegacyTx{
		Nonce:    nonce,
		To:       to,
		Value:    value,
		Gas:      500000,
		GasPrice: gasPrice,
	})
	signedTx, err := from.Signer(from.From, tx)
	if err != nil {
		return nil, err
	}

	return signedTx, nil
}
