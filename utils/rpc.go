package utils

import (
	"net"
	"net/http"

	"github.com/scroll-tech/go-ethereum/rpc"
)

// StartHTTPEndpoint starts the HTTP RPC endpoint.
func StartHTTPEndpoint(endpoint string, timeouts rpc.HTTPTimeouts, handler http.Handler) (*http.Server, net.Addr, error) {
	// start the HTTP listener
	var (
		listener net.Listener
		err      error
	)
	if listener, err = net.Listen("tcp", endpoint); err != nil {
		return nil, nil, err
	}
	// Bundle and start the HTTP server
	httpSrv := &http.Server{
		Handler:      handler,
		ReadTimeout:  timeouts.ReadTimeout,
		WriteTimeout: timeouts.WriteTimeout,
		IdleTimeout:  timeouts.IdleTimeout,
	}
	go func() {
		_ = httpSrv.Serve(listener)
	}()
	return httpSrv, listener.Addr(), err
}
