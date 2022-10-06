package main

import "github.com/urfave/cli/v2"

var (
	commonFlags = []cli.Flag{
		&configFileFlag,
		&verbosityFlag,
		&logFileFlag,
		&logJsonFormat,
		&logDebugFlag,
		&verifierFlag,
	}
	// configFileFlag load json type config file.
	configFileFlag = cli.StringFlag{
		Name:  "config",
		Usage: "JSON configuration file",
		Value: "./config.json",
	}
	// verbosityFlag log level.
	verbosityFlag = cli.IntFlag{
		Name:  "verbosity",
		Usage: "Logging verbosity: 0=silent, 1=error, 2=warn, 3=info, 4=debug, 5=detail",
		Value: 3,
	}
	// logFileFlag decides where the logger output is sent. If this flag is left
	// empty, it will log to stdout.
	logFileFlag = cli.StringFlag{
		Name:  "log.file",
		Usage: "Tells the sequencer where to write log entries",
	}
	logJsonFormat = cli.BoolFlag{
		Name:  "log.json",
		Usage: "Tells the sequencer whether log format is json or not",
		Value: true,
	}
	logDebugFlag = cli.BoolFlag{
		Name:  "log.debug",
		Usage: "Prepends log messages with call-site location (file and line number)",
	}
	verifierFlag = cli.StringFlag{
		Name:  "verifier-socket-file",
		Usage: "The path of ipc-verifier socket file",
		Value: "/tmp/verifier.sock",
	}

	// httpEnabledFlag enable rpc server.
	apiFlags = []cli.Flag{
		&wsPortFlag,
		&httpEnabledFlag,
		&httpListenAddrFlag,
		&httpPortFlag,
	}
	// websocket port
	wsPortFlag = cli.IntFlag{
		Name:  "ws.port",
		Usage: "WS-RPC server listening port",
		Value: 9000,
	}
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

	// l1
	l1Flags = []cli.Flag{
		&l1ChainIDFlag,
		&l1UrlFlag,
	}
	l1ChainIDFlag = cli.IntFlag{
		Name:  "l1.chainID",
		Usage: "l1 chain id",
		Value: 4,
	}
	l1UrlFlag = cli.StringFlag{
		Name:  "l1.endpoint",
		Usage: "The endpoint connect to l1chain node",
		Value: "https://goerli.infura.io/v3/9aa3d95b3bc440fa88ea12eaa4456161",
	}

	// l2
	l2Flags = []cli.Flag{
		&l2ChainIDFlag,
		&l2UrlFlag,
	}
	l2ChainIDFlag = cli.IntFlag{
		Name:  "l2.chainID",
		Usage: "l2 chain id",
		Value: 53077,
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
		Name:  "driver",
		Usage: "db driver name",
		Value: "postgres",
	}
	dsnFlag = cli.StringFlag{
		Name:  "dsn",
		Usage: "data source name",
		Value: "postgres://postgres:@localhost/postgres?sslmode=disable",
	}
)
