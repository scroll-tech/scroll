package app

import (
	"fmt"
	"os"

	"github.com/docker/docker/pkg/reexec"
	"github.com/urfave/cli/v2"

	"scroll-tech/common/utils"
	"scroll-tech/common/version"
)

var (
	apiFlags = []cli.Flag{
		&httpEnabledFlag,
		&httpListenAddrFlag,
		&httpPortFlag,
	}
	// httpEnabledFlag enable rpc server.
	httpEnabledFlag = cli.BoolFlag{
		Name:  "http",
		Usage: "Enable the HTTP-RPC server",
		Value: false,
	}
	// httpListenAddrFlag set the http address.
	httpListenAddrFlag = cli.StringFlag{
		Name:  "http.addr",
		Usage: "HTTP-RPC server listening interface",
		Value: "localhost",
	}
	// httpPortFlag set http.port.
	httpPortFlag = cli.IntFlag{
		Name:  "http.port",
		Usage: "HTTP-RPC server listening port",
		Value: 8290,
	}
	l1Flags = []cli.Flag{
		&l1UrlFlag,
	}
	l1UrlFlag = cli.StringFlag{
		Name:  "l1.endpoint",
		Usage: "The endpoint connect to l1chain node",
		Value: "https://goerli.infura.io/v3/9aa3d95b3bc440fa88ea12eaa4456161",
	}
	l2Flags = []cli.Flag{
		&l2UrlFlag,
	}
	l2UrlFlag = cli.StringFlag{
		Name:  "l2.endpoint",
		Usage: "The endpoint connect to l2chain node",
		Value: "/var/lib/jenkins/workspace/SequencerPipeline/MyPrivateNetwork/geth.ipc",
	}
	// db
	dbflags = []cli.Flag{
		&driverFlag,
		&dsnFlag,
	}
	driverFlag = cli.StringFlag{
		Name:  "db.driver",
		Usage: "db driver name",
		Value: "postgres",
	}
	dsnFlag = cli.StringFlag{
		Name:  "db.dsn",
		Usage: "data source name",
		Value: "postgres://postgres:@localhost/postgres?sslmode=disable",
	}
)

var (
	// Set up Bridge app info.
	app = cli.NewApp()
)

func init() {
	app.Action = action
	app.Name = "bridge"
	app.Usage = "The Scroll Bridge"
	app.Version = version.Version
	app.Flags = append(app.Flags, commonFlags...)
	app.Flags = append(app.Flags, apiFlags...)
	app.Flags = append(app.Flags, l1Flags...)
	app.Flags = append(app.Flags, l2Flags...)
	app.Flags = append(app.Flags, dbflags...)

	app.Before = func(ctx *cli.Context) error {
		return utils.Setup(&utils.LogConfig{
			LogFile:       ctx.String(logFileFlag.Name),
			LogJSONFormat: ctx.Bool(logJSONFormat.Name),
			LogDebug:      ctx.Bool(logDebugFlag.Name),
			Verbosity:     ctx.Int(verbosityFlag.Name),
		})
	}

	// Run the app for integration-test
	reexec.Register("bridge-test", func() {
		if err := app.Run(os.Args); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		os.Exit(0)
	})
	// check if we have been reexec'd
	reexec.Init()
}
