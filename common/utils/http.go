package utils

import (
	"net/http"
	"time"
)

// StartHTTPServer a public http server to be used.
func StartHTTPServer(address string, handler http.Handler) (*http.Server, error) {
	srv := &http.Server{
		Handler:      handler,
		Addr:         address,
		ReadTimeout:  time.Second * 3,
		WriteTimeout: time.Second * 3,
		IdleTimeout:  time.Second * 12,
	}
	errCh := make(chan error, 1)
	go func() {
		errCh <- srv.ListenAndServe()
	}()
	select {
	case err := <-errCh:
		return nil, err
	case <-time.After(time.Second):
	}
	return srv, nil
}
