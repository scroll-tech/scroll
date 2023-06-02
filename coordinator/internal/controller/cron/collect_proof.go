package cron

import (
	"context"
	"fmt"
	"scroll-tech/common/types/message"
	"time"

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
	collectors map[message.ProveType]collector.Collector
}

// NewCollector create a collector to cron collect the data to send to prover
func NewCollector(ctx context.Context, db *gorm.DB, cfg *config.Config) *Collector {
	c := &Collector{
		cfg:        cfg,
		ctx:        ctx,
		stopChan:   make(chan struct{}),
		collectors: make(map[message.ProveType]collector.Collector),
	}

	c.collectors[message.BasicProve] = collector.NewBlockBatchCollector(cfg, db)
	c.collectors[message.AggregatorProve] = collector.NewAggTaskCollector(cfg, db)

	go c.run()

	return c
}

// Stop all the collector
func (c *Collector) Stop() {
	c.stopChan <- struct{}{}
}

func (c *Collector) Start(collectorType message.ProveType) {
	co, ok := c.collectors[collectorType]
	if !ok {
		log.Warn("no collector type", "collector", collectorType)
		return
	}
	co.Start()
}

func (c *Collector) Pause(collectorType message.ProveType) {
	co, ok := c.collectors[collectorType]
	if !ok {
		log.Warn("no collector type", "collector", collectorType)
		return
	}
	co.Pause()
}

// run loop and cron collect
func (c *Collector) run() {
	defer func() {
		if err := recover(); err != nil {
			nerr := fmt.Errorf("collector panic error:%v", err)
			log.Warn(nerr.Error())
		}
	}()

	ticker := time.NewTicker(time.Duration(c.cfg.RollerManagerConfig.CollectionTime*10) * time.Second)

	for {
		select {
		case <-ticker.C:
			log.Info("star collecting..")
			for _, tmpCollector := range c.collectors {
				if err := tmpCollector.Collect(c.ctx); err != nil {
					log.Warn("%s collect data to prover failure:%v", tmpCollector.Type(), err)
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
