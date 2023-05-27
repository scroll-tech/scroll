package cron

import (
	"context"
	"fmt"
	"time"

	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/scroll-tech/go-ethereum/log"
	"gorm.io/gorm"

	"scroll-tech/coordinator/internal/config"
	"scroll-tech/coordinator/internal/logic/collector"
)

// Collector collect the block batch or agg task to send to prover
type Collector struct {
	cfg *config.Config

	ctx        context.Context
	stopChan   chan struct{}
	collectors []collector.Collector
}

// NewCollector create a collector to cron collect the data to send to prover
func NewCollector(ctx context.Context, db *gorm.DB, l2gethClient *ethclient.Client, cfg *config.Config) *Collector {
	c := &Collector{
		cfg:      cfg,
		ctx:      ctx,
		stopChan: make(chan struct{}),
	}

	c.collectors = append(c.collectors, collector.NewBlockBatchCollector(ctx, cfg, db, l2gethClient))
	//c.collectors = append(c.collectors, collector.NewAggTaskCollector(ctx))

	go c.run()

	return c
}

func (c *Collector) Stop() {
	c.stopChan <- struct{}{}
}

// run loop and cron collect
func (c *Collector) run() {
	defer func() {
		if err := recover(); err != nil {
			nerr := fmt.Errorf("collector panic err:%w", err)
			log.Warn(nerr.Error())
		}
	}()

	ticker := time.NewTicker(time.Duration(c.cfg.RollerManagerConfig.CollectionTime) * time.Minute)

	for {
		select {
		case <-ticker.C:
			for _, tmpCollector := range c.collectors {
				if err := tmpCollector.Collect(c.ctx); err != nil {
					log.Warn("%s collect data to prover failure:%v", tmpCollector.Name(), err)
				}
			}
		case <-c.ctx.Done():
			if c.ctx.Err() != nil {
				log.Error("manager context canceled with error", "error", c.ctx.Err())
			}
			return
		case <-c.stopChan:
			log.Info("the coordinator run loop exit")
			return
		}
	}
}
