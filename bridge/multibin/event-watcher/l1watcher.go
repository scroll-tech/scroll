package eventwatcher

import (
	"scroll-tech/bridge/config"
	"scroll-tech/bridge/l1"
	"scroll-tech/database"
)

type L1EventWatcher struct {
	cfg     *config.L1Config
	watcher *l1.Watcher
	orm     database.OrmFactory
}

func (w *L1EventWatcher) Start() {
	w.watcher.Start()
}
