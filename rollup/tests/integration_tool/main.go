package main

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/scroll-tech/da-codec/encoding"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/urfave/cli/v2"

	"scroll-tech/common/database"
	"scroll-tech/common/utils"
	"scroll-tech/common/version"
)

var app *cli.App
var cfg *config
var codecCfg encoding.CodecVersion = encoding.CodecV8

var outputNumFlag = cli.StringFlag{
	Name:  "counts",
	Usage: "Counts for output (chunks,batches,bundles)",
	Value: "4,2,1",
}

var outputPathFlag = cli.StringFlag{
	Name:  "output",
	Usage: "output file path",
	Value: "testset.json",
}

var seedFlag = cli.Int64Flag{
	Name:  "seed",
	Usage: "random seed, 0 to use random selected seed",
	Value: 0,
}

var codecFlag = cli.IntFlag{
	Name:  "codec",
	Usage: "codec version, valid from 6, default(auto) is 0",
	Value: 0,
}

func parseThreeIntegers(value string) (int, int, int, error) {
	// Split the input string by comma
	parts := strings.Split(value, ",")

	// Check that we have exactly 3 parts
	if len(parts) != 3 {
		return 0, 0, 0, fmt.Errorf("input must contain exactly 3 comma-separated integers, got %s", value)
	}

	// Parse the three integers
	values := make([]int, 3)
	for i, part := range parts {
		// Trim any whitespace
		part = strings.TrimSpace(part)

		// Parse the integer
		val, err := strconv.Atoi(part)
		if err != nil {
			return 0, 0, 0, fmt.Errorf("failed to parse '%s' as integer: %w", part, err)
		}

		// Check that it's positive
		if val <= 0 {
			return 0, 0, 0, fmt.Errorf("all integers must be greater than 0, got %d", val)
		}

		values[i] = val
	}

	// Check that first >= second >= third
	if values[0] < values[1] || values[1] < values[2] {
		return 0, 0, 0, fmt.Errorf("integers must be in descending order: %d >= %d >= %d",
			values[0], values[1], values[2])
	}

	return values[0], values[1], values[2], nil
}

// load a comptabile type of config for rollup
type config struct {
	DBConfig *database.Config `json:"db_config"`
}

func init() {
	// Set up coordinator app info.
	app = cli.NewApp()
	app.Action = action
	app.Name = "integration-test-tool"
	app.Usage = "The Scroll L2 Integration Test Tool"
	app.Version = version.Version
	app.Flags = append(app.Flags, &codecFlag, &seedFlag, &outputNumFlag, &outputPathFlag)
	app.Flags = append(app.Flags, utils.CommonFlags...)
	app.Before = func(ctx *cli.Context) error {
		if err := utils.LogSetup(ctx); err != nil {
			return err
		}

		cfgFile := ctx.String(utils.ConfigFileFlag.Name)
		var err error
		cfg, err = newConfig(cfgFile)
		if err != nil {
			log.Crit("failed to load config file", "config file", cfgFile, "error", err)
		}
		return nil
	}
}

func newConfig(file string) (*config, error) {
	buf, err := os.ReadFile(filepath.Clean(file))
	if err != nil {
		return nil, err
	}

	cfg := &config{}
	err = json.Unmarshal(buf, cfg)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}

func action(ctx *cli.Context) error {

	if ctx.Args().Len() < 2 {
		return fmt.Errorf("specify begin and end block number")
	}

	codecFl := ctx.Int(codecFlag.Name)
	if codecFl != 0 {
		switch codecFl {
		case 6:
			codecCfg = encoding.CodecV6
		case 7:
			codecCfg = encoding.CodecV7
		case 8:
			codecCfg = encoding.CodecV8
		default:
			return fmt.Errorf("invalid codec version %d", codecFl)
		}
		log.Info("set codec", "version", codecCfg)
	}

	beginBlk, err := strconv.ParseUint(ctx.Args().First(), 10, 64)
	if err != nil {
		return fmt.Errorf("invalid begin block number: %w", err)
	}
	endBlk, err := strconv.ParseUint(ctx.Args().Get(1), 10, 64)
	if err != nil {
		return fmt.Errorf("invalid begin block number: %w", err)
	}

	chkNum, batchNum, bundleNum, err := parseThreeIntegers(ctx.String(outputNumFlag.Name))
	if err != nil {
		return err
	}

	seed := ctx.Int64(seedFlag.Name)
	if seed == 0 {
		seed = rand.Int63()
	}

	outputPath := ctx.String(outputPathFlag.Name)
	log.Info("output", "Seed", seed, "file", outputPath)
	ret, err := importData(ctx.Context, beginBlk, endBlk, chkNum, batchNum, bundleNum, seed)
	if err != nil {
		return err
	}
	// Marshal the ret variable to JSON with indentation for readability
	jsonData, err := json.MarshalIndent(ret, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal result data to JSON: %w", err)
	}

	// Write the JSON data to the specified file
	err = os.WriteFile(outputPath, jsonData, 0644)
	if err != nil {
		return fmt.Errorf("failed to write result to file %s: %w", outputPath, err)
	}

	return nil
}

func main() {
	if err := app.Run(os.Args); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
