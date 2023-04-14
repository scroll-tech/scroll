package metrics

import (
	"context"
	"net"
	"net/http"
	"strconv"

	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/metrics"
	"github.com/scroll-tech/go-ethereum/metrics/prometheus"
	"github.com/scroll-tech/go-ethereum/rpc"
	"github.com/urfave/cli/v2"

	"scroll-tech/common/utils"
)

var (
	// ScrollRegistry is used for scroll metrics.
	ScrollRegistry = metrics.NewRegistry()
)

// Serve starts the metrics server on the given address, will be closed when the given
// context is canceled.
func Serve(ctx context.Context, c *cli.Context) {
	if !c.Bool(utils.MetricsEnabled.Name) {
		return
	}

	address := net.JoinHostPort(
		c.String(utils.MetricsAddr.Name),
		strconv.Itoa(c.Int(utils.MetricsPort.Name)),
	)

	server := &http.Server{
		Addr:         address,
		Handler:      prometheus.Handler(ScrollRegistry),
		ReadTimeout:  rpc.DefaultHTTPTimeouts.ReadTimeout,
		WriteTimeout: rpc.DefaultHTTPTimeouts.WriteTimeout,
		IdleTimeout:  rpc.DefaultHTTPTimeouts.IdleTimeout,
	}

	go func() {
		<-ctx.Done()
		if err := server.Close(); err != nil {
			log.Error("Failed to close metrics server", "error", err)
		}
	}()

	log.Info("Starting metrics server", "address", address)

	go func() {
		if err := server.ListenAndServe(); err != nil {
			log.Error("start metrics server error", "error", err)
		}
	}()
}
