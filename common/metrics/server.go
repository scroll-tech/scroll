package metrics

import (
	"errors"
	"fmt"
	"net/http"
	"time"

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

	address := fmt.Sprintf(":%s", c.String(utils.MetricsPort.Name))

	log.Info("Starting metrics server", "address", address)

	server := &http.Server{
		Addr:              address,
		Handler:           promhttp.HandlerFor(reg, promhttp.HandlerOpts{Registry: reg}),
		ReadHeaderTimeout: time.Minute,
	}

	go func() {
		if runServerErr := server.ListenAndServe(); runServerErr != nil && !errors.Is(runServerErr, http.ErrServerClosed) {
			log.Crit("run metrics http server failure", "error", runServerErr)
		}
	}()
}
