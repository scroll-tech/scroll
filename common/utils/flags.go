package utils

import (
	"github.com/urfave/cli/v2"
)

var (
	// CommonFlags is used for app common flags in different modules
	CommonFlags = []cli.Flag{
		&ConfigFileFlag,
		&VerbosityFlag,
		&LogFileFlag,
		&LogJSONFormat,
		&LogDebugFlag,
		&MetricsEnabled,
		&MetricsAddr,
		&MetricsPort,
		&ServicePortFlag,
		&Genesis,
	}
	// RollupRelayerFlags contains flags only used in rollup-relayer
	RollupRelayerFlags = []cli.Flag{
		&MinCodecVersionFlag,
	}
	// ProposerToolFlags contains flags only used in proposer tool
	ProposerToolFlags = []cli.Flag{
		&StartL2BlockFlag,
	}
	// ConfigFileFlag load json type config file.
	ConfigFileFlag = cli.StringFlag{
		Name:  "config",
		Usage: "JSON configuration file",
		Value: "./conf/config.json",
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
		Usage: "Tells the module where to write log entries",
	}
	// LogJSONFormat decides the log format is json or not
	LogJSONFormat = cli.BoolFlag{
		Name:  "log.json",
		Usage: "Tells the module whether log format is json or not",
		Value: true,
	}
	// LogDebugFlag make log messages with call-site location
	LogDebugFlag = cli.BoolFlag{
		Name:  "log.debug",
		Usage: "Prepends log messages with call-site location (file and line number)",
	}
	// MetricsEnabled enable metrics collection and reporting
	MetricsEnabled = cli.BoolFlag{
		Name:     "metrics",
		Usage:    "Enable metrics collection and reporting",
		Category: "METRICS",
		Value:    false,
	}
	// MetricsAddr is listening address of Metrics reporting server
	MetricsAddr = cli.StringFlag{
		Name:     "metrics.addr",
		Usage:    "Metrics reporting server listening address",
		Category: "METRICS",
		Value:    "127.0.0.1",
	}
	// MetricsPort is listening port of Metrics reporting server
	MetricsPort = cli.IntFlag{
		Name:     "metrics.port",
		Usage:    "Metrics reporting server listening port",
		Category: "METRICS",
		Value:    6060,
	}
	// ServicePortFlag is the port the service will listen on
	ServicePortFlag = cli.IntFlag{
		Name:  "service.port",
		Usage: "Port that the service will listen on",
		Value: 8080,
	}
	// Genesis is the genesis file
	Genesis = cli.StringFlag{
		Name:  "genesis",
		Usage: "Genesis file of the network",
		Value: "./conf/genesis.json",
	}
	// MinCodecVersionFlag defines the minimum codec version required for the chunk/batch/bundle proposers
	MinCodecVersionFlag = cli.UintFlag{
		Name:     "min-codec-version",
		Usage:    "Minimum required codec version for the chunk/batch/bundle proposers",
		Required: true,
	}
	// StartL2BlockFlag indicates the start L2 block number for proposer tool
	StartL2BlockFlag = cli.Uint64Flag{
		Name:  "start-l2-block",
		Usage: "Start L2 block number for proposer tool",
		Value: 0,
	}
)
