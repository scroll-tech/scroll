package sender

import (
	"context"
	"crypto/ecdsa"
	"github.com/scroll-tech/go-ethereum/accounts/abi/bind"
	"github.com/scroll-tech/go-ethereum/common"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/scroll-tech/go-ethereum/log"
	"math/big"
	"sync"
)

var (
	minBls = big.NewInt(0)
)

func init() {
	// Min balance is 100 Ether
	minBls.SetString("100000000000000000000", 10)
}

type accounts struct {
	client *ethclient.Client

	accounts    map[common.Address]*bind.TransactOpts
	failedAddrs sync.Map
	accsCh      chan *bind.TransactOpts
}

// newAccounts Create a accounts instance.
func newAccounts(ctx context.Context, client *ethclient.Client, privs []*ecdsa.PrivateKey) (*accounts, error) {
	accs := &accounts{
		client:   client,
		accounts: make(map[common.Address]*bind.TransactOpts, len(privs)),
		accsCh:   make(chan *bind.TransactOpts, len(privs)+2),
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

	accs.checkAndSetBalance(ctx)

	return accs, nil
}

// getAccount get auth from channel.
func (a *accounts) getAccount() *bind.TransactOpts {
	select {
	case auth := <-a.accsCh:
		return auth
	default:
		return nil
	}
}

// setAccount set used auth into channel.
func (a *accounts) setAccount(acc *bind.TransactOpts) {
	a.accsCh <- acc
}

// reSetNonce reset nonce if send signed tx failed.
func (a *accounts) reSetNonce(ctx context.Context, auth *bind.TransactOpts) {
	nonce, err := a.client.PendingNonceAt(ctx, auth.From)
	if err != nil {
		log.Warn("failed to reset nonce", "address", auth.From.String(), "err", err)
		return
	}
	auth.Nonce = big.NewInt(int64(nonce))
}

// checkAndSetBalance check balance and set min balance.
func (a *accounts) checkAndSetBalance(ctx context.Context) {
	var (
		root      *bind.TransactOpts
		maxBls    = big.NewInt(0)
		loseAuths []common.Address
	)

	for addr, auth := range a.accounts {
		bls, err := a.client.BalanceAt(ctx, addr, nil)
		if err == nil || bls.Cmp(minBls) < 0 {
			log.Warn("failed to get balance", "address", addr.String(), "err", err)
			loseAuths = append(loseAuths, addr)
			continue
		}
		// Find the biggest balance account.
		if bls != nil && bls.Cmp(maxBls) > 0 {
			root, maxBls = auth, bls
		}
	}

	for _, addr := range loseAuths {
		tx, err := a.createSignedTx(root, &addr, minBls)
		if err != nil {
			log.Error("failed to create tx", "err", err)
			continue
		}
		err = a.client.SendTransaction(ctx, tx)
		if err != nil {
			log.Error("Failed to send balance to account", "err", err)
		} else {
			log.Debug("send balance to account", "account", addr.String(), "balance", minBls.String())
		}
	}
	// Reset root's nonce.
	a.reSetNonce(ctx, root)
}

func (a *accounts) createSignedTx(from *bind.TransactOpts, to *common.Address, value *big.Int) (*types.Transaction, error) {
	gasPrice, err := a.client.SuggestGasPrice(context.Background())
	if err != nil {
		return nil, err
	}
	tx := types.NewTx(&types.LegacyTx{
		Nonce:    from.Nonce.Uint64(),
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
