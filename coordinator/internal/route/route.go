package route

import (
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"

	"scroll-tech/common/ginmetrics"

	"scroll-tech/coordinator/internal/config"
	"scroll-tech/coordinator/internal/controller/api"
	"scroll-tech/coordinator/internal/middleware"
)

// Route register route for coordinator
func Route(router *gin.Engine, cfg *config.Config, reg prometheus.Registerer) {
	router.Use(gin.Recovery())

	apiMetric(router, reg)

	r := router.Group("coordinator")

	v1(r, cfg)
}

func apiMetric(r *gin.Engine, reg prometheus.Registerer) {
	m := ginmetrics.GetMonitor(reg)
	m.SetMetricPath("/metrics")
	m.SetMetricPrefix("coordinator_")
	m.SetSlowTime(1)
	m.SetDuration([]float64{0.025, .05, .1, .5, 1, 5, 10})
	m.Use(r)
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
