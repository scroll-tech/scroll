package controller

import (
	"github.com/gin-gonic/gin"

	"bridge-history-api/internal/types"
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
	types.RenderSuccess(c, nil)
}
