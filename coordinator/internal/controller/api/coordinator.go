package api

import (
	"errors"

	"scroll-tech/common/types/message"

	"scroll-tech/coordinator/internal/controller/cron"
)

const (
	StatusPause = iota
	StatusStart
)

type CoordinatorController struct {
	co *cron.Collector
}

func NewCoordinatorController(co *cron.Collector) *CoordinatorController {
	return &CoordinatorController{co: co}
}

func (c *CoordinatorController) SetSendTaskStatus(typ message.ProveType, status int) error {
	switch status {
	case StatusPause:
		c.co.Pause(typ)
	case StatusStart:
		c.co.Start(typ)
	default:
		return errors.New("invalid status code")
	}
	return nil
}
