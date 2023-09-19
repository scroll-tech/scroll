package cron

import (
	"fmt"
	"time"

	"github.com/scroll-tech/go-ethereum/log"

	"scroll-tech/common/utils"
)

func (c *Collector) cleanupChallenge() {
	defer func() {
		if err := recover(); err != nil {
			nerr := fmt.Errorf("clean challenge panic error:%v", err)
			log.Warn(nerr.Error())
		}
	}()

	ticker := time.NewTicker(time.Minute * 10)
	for {
		select {
		case <-ticker.C:
			expiredTime := utils.NowUTC().Add(-time.Hour)
			if err := c.challenge.DeleteExpireChallenge(c.ctx, expiredTime); err != nil {
				log.Error("delete expired challenge failure", "error", err)
			}
		case <-c.ctx.Done():
			if c.ctx.Err() != nil {
				log.Error("manager context canceled with error", "error", c.ctx.Err())
			}
			return
		case <-c.stopTimeoutChan:
			log.Info("the coordinator run loop exit")
			return
		}
	}

}
