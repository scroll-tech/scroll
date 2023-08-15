package app

import "github.com/urfave/cli/v2"

var (
	apiFlags = []cli.Flag{
		&httpPortFlag,
	}
	// httpPortFlag set http.port.
	httpPortFlag = cli.IntFlag{
		Name:  "http.port",
		Usage: "HTTP server listening port",
		Value: 8990,
	}
)
