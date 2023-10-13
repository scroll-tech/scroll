package main

import (
	"net/http"
	_ "net/http/pprof"
	"scroll-tech/prover/cmd/app"
)

func main() {
	go func() {
		http.ListenAndServe("0.0.0.0:6060", nil)
	}()

	app.Run()
}
