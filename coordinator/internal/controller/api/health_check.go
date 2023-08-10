package api

import (
	"github.com/gin-gonic/gin"

	ctypes "scroll-tech/common/types"

	"scroll-tech/coordinator/internal/types"
)

// HealthCheckController is health check API
type HealthCheckController struct {
}

// NewHealthCheckController returns an HealthCheckController instance
func NewHealthCheckController() *HealthCheckController {
	return &HealthCheckController{}
}

// HealthCheck the api controller for coordinator health check
func (a *HealthCheckController) HealthCheck(c *gin.Context) {
	types.RenderJSON(c, ctypes.Success, nil, nil)
}
