package route

import (
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"

	"scroll-tech/common/observability"

	"scroll-tech/coordinator/internal/config"
	"scroll-tech/coordinator/internal/controller/api"
	"scroll-tech/coordinator/internal/controller/proxy"
	"scroll-tech/coordinator/internal/middleware"
)

// Route register route for coordinator
func Route(router *gin.Engine, cfg *config.Config, reg prometheus.Registerer) {
	router.Use(gin.Recovery())

	observability.Use(router, "coordinator", reg)

	r := router.Group("coordinator")

	v1(r, cfg)
}

func v1(router *gin.RouterGroup, conf *config.Config) {
	r := router.Group("/v1")

	challengeMiddleware := middleware.ChallengeMiddleware(conf.Auth)
	r.GET("/challenge", challengeMiddleware.LoginHandler)

	loginMiddleware := middleware.LoginMiddleware(conf)
	r.POST("/login", challengeMiddleware.MiddlewareFunc(), loginMiddleware.LoginHandler)

	// need jwt token api
	r.Use(loginMiddleware.MiddlewareFunc())
	{
		r.POST("/proxy_login", loginMiddleware.LoginHandler)
		r.POST("/get_task", api.GetTask.GetTasks)
		r.POST("/submit_proof", api.SubmitProof.SubmitProof)
	}
}

func v1_proxy(router *gin.RouterGroup, conf *config.ProxyConfig) {
	r := router.Group("/v1")

	challengeMiddleware := middleware.ChallengeMiddleware(conf.Auth)
	r.GET("/challenge", challengeMiddleware.LoginHandler)

	loginMiddleware := middleware.ProxyLoginMiddleware(conf)
	r.POST("/login", challengeMiddleware.MiddlewareFunc(), loginMiddleware.LoginHandler)

	// need jwt token api
	r.Use(loginMiddleware.MiddlewareFunc())
	{
		r.POST("/get_task", proxy.GetTask.GetTasks)
		r.POST("/submit_proof", proxy.SubmitProof.SubmitProof)
	}
}
