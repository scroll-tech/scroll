package dockercompose

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/scroll-tech/go-ethereum/log"
	tc "github.com/testcontainers/testcontainers-go/modules/compose"
	"github.com/testcontainers/testcontainers-go/wait"
)

// PoSL1TestEnv represents the config needed to test in PoS Layer 1.
type PoSL1TestEnv struct {
	composeFilePath string
	compose         tc.ComposeStack
}

// NewPoSL1TestEnv creates and initializes a new instance of PoSL1TestEnv with a random HTTP port.
func NewPoSL1TestEnv() (*PoSL1TestEnv, error) {
	rootDir, err := findProjectRootDir()
	if err != nil {
		return nil, fmt.Errorf("failed to find project root directory: %v", err)
	}

	return &PoSL1TestEnv{
		composeFilePath: filepath.Join(rootDir, "common", "docker-compose", "l1", "docker-compose.yml"),
	}, nil
}

// Start starts the PoS L1 test environment by running the associated Docker Compose configuration.
func (e *PoSL1TestEnv) Start() error {
	var err error
	defer func() {
		if err != nil {
			if errStop := e.Stop(); errStop != nil {
				log.Error("failed to stop PoS L1 test environment", "err", errStop)
			}
		}
	}()

	e.compose, err = tc.NewDockerCompose([]string{e.composeFilePath}...)
	if err != nil {
		return fmt.Errorf("failed to create docker compose: %w", err)
	}

	if err = e.compose.WaitForService("geth", wait.NewHTTPStrategy("/").WithPort("8545/tcp").WithStartupTimeout(30*time.Second)).WithOsEnv().Up(context.Background()); err != nil {
		return fmt.Errorf("failed to start PoS L1 test environment: %w", err)
	}

	return nil
}

// Stop stops the PoS L1 test environment by stopping and removing the associated Docker Compose services.
func (e *PoSL1TestEnv) Stop() error {
	if e.compose != nil {
		if err := e.compose.Down(context.Background(), tc.RemoveOrphans(true), tc.RemoveImagesLocal); err != nil {
			return fmt.Errorf("failed to stop PoS L1 test environment: %w", err)
		}
	}
	return nil
}

// Endpoint returns the HTTP endpoint for the PoS L1 test environment.
func (e *PoSL1TestEnv) Endpoint() string {
	return "http://127.0.0.1:8545"
}

// L1Client returns an ethclient by dialing the running PoS L1 test environment
func (e *PoSL1TestEnv) L1Client() (*ethclient.Client, error) {
	if e == nil {
		return nil, fmt.Errorf("PoS L1 test environment is not initialized")
	}

	client, err := ethclient.Dial(e.Endpoint())
	if err != nil {
		return nil, fmt.Errorf("failed to dial PoS L1 test environment: %w", err)
	}
	return client, nil
}

func findProjectRootDir() (string, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to get working directory: %w", err)
	}

	for {
		_, err := os.Stat(filepath.Join(currentDir, "go.work"))
		if err == nil {
			return currentDir, nil
		}

		parentDir := filepath.Dir(currentDir)
		if parentDir == currentDir {
			return "", fmt.Errorf("go.work file not found in any parent directory")
		}

		currentDir = parentDir
	}
}
