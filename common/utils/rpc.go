package utils

import (
	"net"
	"net/http"

	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/rpc"
)

// StartHTTPEndpoint starts the HTTP RPC endpoint.
func StartHTTPEndpoint(endpoint string, apis []rpc.API) (*http.Server, net.Addr, error) {
	srv := rpc.NewServer()
	for _, api := range apis {
		if err := srv.RegisterName(api.Namespace, api.Service); err != nil {
			log.Crit("register namespace failed", "namespace", api.Namespace, "error", err)
		}
	}
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
		Handler:      srv,
		ReadTimeout:  rpc.DefaultHTTPTimeouts.ReadTimeout,
		WriteTimeout: rpc.DefaultHTTPTimeouts.WriteTimeout,
		IdleTimeout:  rpc.DefaultHTTPTimeouts.IdleTimeout,
	}
	go func() {
		_ = httpSrv.Serve(listener)
	}()
	return httpSrv, listener.Addr(), err
}

// StartWSEndpoint starts the WS RPC endpoint.
func StartWSEndpoint(endpoint string, apis []rpc.API, compressionLevel int) (*http.Server, net.Addr, error) {
	handler, addr, err := StartHTTPEndpoint(endpoint, apis)
	if err == nil {
		srv := (handler.Handler).(*rpc.Server)
		err = srv.SetCompressionLevel(compressionLevel)
		if err != nil {
			log.Error("failed to set ws compression level", "compression level", compressionLevel, "err", err)
		}
		handler.Handler = srv.WebsocketHandler(nil)
	}
	return handler, addr, err
}
