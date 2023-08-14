package metrics

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	// enable the pprof
	_ "net/http/pprof"

	"github.com/gin-contrib/pprof"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/urfave/cli/v2"

	"scroll-tech/common/utils"
)

// Server starts the metrics server on the given address, will be closed when the given
// context is canceled.
func Server(c *cli.Context, reg *prometheus.Registry) {
	if !c.Bool(utils.MetricsEnabled.Name) {
		return
	}

	r := gin.New()
	r.Use(gin.Recovery())
	pprof.Register(r)
	r.GET("/metrics", func(context *gin.Context) {
		promhttp.Handler().ServeHTTP(context.Writer, context.Request)
	})

	address := fmt.Sprintf(":%s", c.String(utils.MetricsPort.Name))
	server := &http.Server{
		Addr:              address,
		Handler:           r,
		ReadHeaderTimeout: time.Minute,
	}
	log.Info("Starting metrics server", "address", address)

	go func() {
		if runServerErr := server.ListenAndServe(); runServerErr != nil && !errors.Is(runServerErr, http.ErrServerClosed) {
			log.Crit("run metrics http server failure", "error", runServerErr)
		}
	}()
}
