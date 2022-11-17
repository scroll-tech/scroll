package main

import "github.com/urfave/cli/v2"

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
)
