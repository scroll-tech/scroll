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

	r.GET("/health", api.HealthCheck.HealthCheck)

	challengeMiddleware := middleware.ChallengeMiddleware(conf)
	r.GET("/challenge", challengeMiddleware.LoginHandler)

	loginMiddleware := middleware.LoginMiddleware(conf)
	r.POST("/login", challengeMiddleware.MiddlewareFunc(), loginMiddleware.LoginHandler)

	// need jwt token api
	r.Use(loginMiddleware.MiddlewareFunc())
	{
		r.POST("/get_task", api.GetTask.GetTasks)
		r.POST("/submit_proof", api.SubmitProof.SubmitProof)
	}
}
