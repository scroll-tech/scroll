package api

import (
	"scroll-tech/common/types/message"
	"scroll-tech/coordinator/internal/controller/cron"
)

type CoordinatorController struct {
	co *cron.Collector
}

func NewCoordinatorController(co *cron.Collector) *CoordinatorController {
	return &CoordinatorController{co: co}
}

func (c *CoordinatorController) StartSendTask(typ message.ProveType) error {
	c.co.Start(typ)
	return nil
}

func (c *CoordinatorController) PauseSendTask(typ message.ProveType) error {
	c.co.Pause(typ)
	return nil
}
