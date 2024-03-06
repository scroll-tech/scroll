package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"scroll-tech/common/database"

	"github.com/scroll-tech/go-ethereum/core"
)

// Config load configuration items.
type Config struct {
	L1Config *L1Config        `json:"l1_config"`
	L2Config *L2Config        `json:"l2_config"`
	DBConfig *database.Config `json:"db_config"`
}

func (c *Config) validate() error {
	if maxChunkPerBatch := c.L2Config.BatchProposerConfig.MaxChunkNumPerBatch; maxChunkPerBatch <= 0 {
		return fmt.Errorf("Invalid max_chunk_num_per_batch configuration: %v", maxChunkPerBatch)
	}
	return nil
}

// NewConfig returns a new instance of Config.
func NewConfig(file string) (*Config, error) {
	buf, err := os.ReadFile(filepath.Clean(file))
	if err != nil {
		return nil, err
	}

	cfg := &Config{}
	err = json.Unmarshal(buf, cfg)
	if err != nil {
		return nil, err
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

func ReadGenesis(genesisPath string) (*core.Genesis, error) {
	file, err := os.Open(filepath.Clean(genesisPath))
	if err != nil {
		return nil, err
	}

	genesis := new(core.Genesis)
	if err := json.NewDecoder(file).Decode(genesis); err != nil {
		return nil, errors.Join(err, file.Close())
	}
	return genesis, file.Close()
}
