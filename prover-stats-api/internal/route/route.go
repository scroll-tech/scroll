package route

import (
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	"scroll-tech/prover-stats-api/internal/config"
	"scroll-tech/prover-stats-api/internal/controller"
	"scroll-tech/prover-stats-api/internal/middleware"
)

// Route routes the APIs
func Route(router *gin.Engine, conf *config.Config) {
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	r := router.Group("api/prover_task")

	v1(r, conf)
}

func v1(router *gin.RouterGroup, conf *config.Config) {
	r := router.Group("/v1")

	authMiddleware := middleware.AuthMiddleware(conf)
	r.GET("/request_token", authMiddleware.LoginHandler)

	// need jwt token api
	r.Use(authMiddleware.MiddlewareFunc())
	{
		r.GET("/tasks", controller.ProverTask.ProverTasks)
		r.GET("/total_rewards", controller.ProverTask.GetTotalRewards)
		r.GET("/task", controller.ProverTask.GetTask)
	}
}
