package app

import (
	"encoding/json"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/urfave/cli/v2"
	"io"
	"os"
	"scroll-tech/roller/config"
	"scroll-tech/roller/prover"
)

var (
	// ProveFlags is only used for prove sub command.
	ProveFlags = []cli.Flag{&ParamsDirFlag, &SeedFlag, &TraceFlag}
	// ParamsDirFlag is params dir for params files.
	ParamsDirFlag = cli.StringFlag{
		Name:    "params",
		Aliases: []string{"p"},
		Usage:   "params dir",
		Value:   "./params",
	}
	// SeedFlag is seed file name.
	SeedFlag = cli.StringFlag{
		Name:    "seed",
		Aliases: []string{"s"},
		Usage:   "seed file path",
		Value:   "seed",
	}
	// TraceFlag is trace file name.
	TraceFlag = cli.StringFlag{
		Name:    "traces",
		Aliases: []string{"t"},
		Usage:   "trace file",
		Value:   "trace.json",
	}
)

func prove(ctx *cli.Context) error {
	paramsDir := ctx.String(ParamsDirFlag.Name)
	seedPath := ctx.String(SeedFlag.Name)
	tracePath := ctx.String(TraceFlag.Name)
	pvr, err := prover.NewProver(&config.ProverConfig{
		ParamsPath: paramsDir,
		SeedPath:   seedPath,
		ProveType:  0,
		DumpDir:    "proofs",
	})
	if err != nil {
		return err
	}
	f, err := os.Open(tracePath) //nolint:gosec
	if err != nil {
		return err
	}
	byt, err := io.ReadAll(f)
	if err != nil {
		return err
	}
	trace := new(types.BlockTrace)
	err = json.Unmarshal(byt, trace)
	if err != nil {
		return err
	}
	_, err = pvr.Prove("test", []*types.BlockTrace{trace})
	return err
}
