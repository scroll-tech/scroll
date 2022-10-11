package utils

import "github.com/urfave/cli/v2"

var (
	CommonFlags = []cli.Flag{
		&ConfigFileFlag,
		&VerbosityFlag,
		&LogFileFlag,
		&LogJsonFormat,
		&LogDebugFlag,
		&VerifierFlag,
	}
	// ConfigFileFlag load json type config file.
	ConfigFileFlag = cli.StringFlag{
		Name:  "config",
		Usage: "JSON configuration file",
		Value: "./config.json",
	}
	// VerbosityFlag log level.
	VerbosityFlag = cli.IntFlag{
		Name:  "verbosity",
		Usage: "Logging verbosity: 0=silent, 1=error, 2=warn, 3=info, 4=debug, 5=detail",
		Value: 3,
	}
	// LogFileFlag decides where the logger output is sent. If this flag is left
	// empty, it will log to stdout.
	LogFileFlag = cli.StringFlag{
		Name:  "log.file",
		Usage: "Tells the sequencer where to write log entries",
	}
	LogJsonFormat = cli.BoolFlag{
		Name:  "log.json",
		Usage: "Tells the sequencer whether log format is json or not",
		Value: true,
	}
	LogDebugFlag = cli.BoolFlag{
		Name:  "log.debug",
		Usage: "Prepends log messages with call-site location (file and line number)",
	}
	VerifierFlag = cli.StringFlag{
		Name:  "verifier-socket-file",
		Usage: "The path of ipc-verifier socket file",
		Value: "/tmp/verifier.sock",
	}

	// HTTPEnabledFlag enable rpc server.
	APIFlags = []cli.Flag{
		&WSPortFlag,
		&HTTPEnabledFlag,
		&HTTPListenAddrFlag,
		&HTTPPortFlag,
	}
	// websocket port
	WSPortFlag = cli.IntFlag{
		Name:  "ws.port",
		Usage: "WS-RPC server listening port",
		Value: 9000,
	}
	HTTPEnabledFlag = cli.BoolFlag{
		Name:  "http",
		Usage: "Enable the HTTP-RPC server",
		Value: false,
	}
	// HTTPListenAddrFlag set the http address.
	HTTPListenAddrFlag = cli.StringFlag{
		Name:  "http.addr",
		Usage: "HTTP-RPC server listening interface",
		Value: "localhost",
	}
	// HTTPPortFlag set http.port.
	HTTPPortFlag = cli.IntFlag{
		Name:  "http.port",
		Usage: "HTTP-RPC server listening port",
		Value: 8290,
	}

	// l1
	L1Flags = []cli.Flag{
		&L1ChainIDFlag,
		&L1UrlFlag,
	}
	L1ChainIDFlag = cli.IntFlag{
		Name:  "l1.chainID",
		Usage: "l1 chain id",
		Value: 4,
	}
	L1UrlFlag = cli.StringFlag{
		Name:  "l1.endpoint",
		Usage: "The endpoint connect to l1chain node",
		Value: "https://goerli.infura.io/v3/9aa3d95b3bc440fa88ea12eaa4456161",
	}

	// l2
	L2Flags = []cli.Flag{
		&L2ChainIDFlag,
		&L2UrlFlag,
	}
	L2ChainIDFlag = cli.IntFlag{
		Name:  "l2.chainID",
		Usage: "l2 chain id",
		Value: 53077,
	}
	L2UrlFlag = cli.StringFlag{
		Name:  "l2.endpoint",
		Usage: "The endpoint connect to l2chain node",
		Value: "/var/lib/jenkins/workspace/SequencerPipeline/MyPrivateNetwork/geth.ipc",
	}

	// db
	DBflags = []cli.Flag{
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
