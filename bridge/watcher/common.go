package watcher

import "github.com/scroll-tech/go-ethereum/common"

const contractEventsBlocksFetchLimit = int64(10)

type relayedMessage struct {
	msgHash      common.Hash
	txHash       common.Hash
	isSuccessful bool
}
