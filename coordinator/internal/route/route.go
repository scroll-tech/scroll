package route

import (
	"github.com/gin-gonic/gin"

	"scroll-tech/coordinator/internal/config"
	"scroll-tech/coordinator/internal/controller/api"
	"scroll-tech/coordinator/internal/middleware"
)

// Route register route for coordinator
func Route(router *gin.Engine, cfg *config.Config) {
	r := router.Group("coordinator")
	v1(r, cfg)
}

func v1(router *gin.RouterGroup, conf *config.Config) {
	r := router.Group("/v1")

	authMiddleware := middleware.AuthMiddleware(conf)
	r.POST("/login", authMiddleware.LoginHandler)

	// need jwt token api
	r.Use(authMiddleware.MiddlewareFunc())
	{
		r.POST("/prover_tasks", api.ProverTask.ProverTasks)
		r.POST("/submit_proof", api.SubmitProof.SubmitProof)
	}
}
