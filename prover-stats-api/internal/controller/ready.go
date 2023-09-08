package controller

import (
	"github.com/gin-gonic/gin"

	"scroll-tech/common/types"
)

// ReadyController ready API
type ReadyController struct {
}

// NewReadyController returns an ReadyController instance
func NewReadyController() *ReadyController {
	return &ReadyController{}
}

// Ready the api controller for coordinator ready
func (r *ReadyController) Ready(c *gin.Context) {
	types.RenderJSON(c, types.Success, nil, nil)
}
