package bridge

import "github.com/scroll-tech/go-ethereum/ethclient"

// API is the api bridge between l1 and l2.
type API interface {
	GetClient() *ethclient.Client
}
