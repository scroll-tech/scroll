package dockercompose

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/scroll-tech/go-ethereum/log"
)

// PoSL1TestEnv represents the config needed to test in PoS Layer 1.
type PoSL1TestEnv struct {
	httpPort          int
	cleanScriptPath   string
	dockerComposePath string
}

// NewPoSL1TestEnv creates and initializes a new instance of PoSL1TestEnv with a random HTTP port.
func NewPoSL1TestEnv() (*PoSL1TestEnv, error) {
	id, err := rand.Int(rand.Reader, big.NewInt(10000))
	if err != nil {
		return nil, fmt.Errorf("failed to generate random number: %v", err)
	}
	httpPort := int(id.Int64() + 50000)

	if err := os.Setenv("HTTP_PORT", strconv.Itoa(httpPort)); err != nil {
		return nil, fmt.Errorf("failed to set HTTP_PORT environment variable: %w", err)
	}

	rootDir, err := findProjectRootDir()
	if err != nil {
		return nil, fmt.Errorf("failed to find project root directory: %v", err)
	}
	cleanScriptPath := filepath.Join(rootDir, "common", "docker-compose", "l1", "clean.sh")
	dockerComposePath := filepath.Join(rootDir, "common", "docker-compose", "l1", "docker-compose.yml")

	return &PoSL1TestEnv{
		httpPort:          httpPort,
		cleanScriptPath:   cleanScriptPath,
		dockerComposePath: dockerComposePath,
	}, nil
}

// Start starts the PoS L1 test environment by running the associated Docker Compose configuration.
func (e *PoSL1TestEnv) Start() error {
	if err := e.cleanUp(); err != nil {
		return fmt.Errorf("failed to clean up: %w", err)
	}

	if err := e.runPoSL1DockerCompose(); err != nil {
		return fmt.Errorf("failed to start PoS L1 test environment: %w", err)
	}

	if err := e.waitForServicesToStart(); err != nil {
		return fmt.Errorf("failed to wait for PoS L1 test environment services: %w", err)
	}

	return nil
}

// Stop stops the PoS L1 test environment by stopping and removing the associated Docker Compose services.
func (e *PoSL1TestEnv) Stop() error {
	if err := e.stopAndCleanUpPoSL1(); err != nil {
		return fmt.Errorf("failed to stop and clean up PoS L1 test environment: %w", err)
	}
	return nil
}

// Endpoint returns the HTTP endpoint for the PoS L1 test environment.
func (e *PoSL1TestEnv) Endpoint() string {
	return fmt.Sprintf("http://127.0.0.1:%d", e.httpPort)
}

// L1Client returns an ethclient by dialing the running PoS L1 test environment
func (e *PoSL1TestEnv) L1Client() (*ethclient.Client, error) {
	if e == nil {
		return nil, fmt.Errorf("PoS L1 test environment is not initialized")
	}

	endpoint := e.Endpoint()
	client, err := ethclient.Dial(endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to dial PoS L1 test environment: %w", err)
	}

	return client, nil
}

func (e *PoSL1TestEnv) runPoSL1DockerCompose() error {
	cmd := exec.Command("docker-compose", "-f", e.dockerComposePath, "up", "-d")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run docker-compose up command: %w", err)
	}
	return nil
}

func (e *PoSL1TestEnv) stopAndCleanUpPoSL1() error {
	cmd := exec.Command("docker-compose", "-f", e.dockerComposePath, "down")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to run docker-compose down command: %w", err)
	}
	return nil
}

func (e *PoSL1TestEnv) cleanUp() error {
	if _, err := os.Stat(e.cleanScriptPath); os.IsNotExist(err) {
		return fmt.Errorf("clean.sh script does not exist in expected path: %v", err)
	}

	cmd := exec.Command("/bin/sh", e.cleanScriptPath)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to execute clean.sh script: %v", err)
	}
	return nil
}

func (e *PoSL1TestEnv) waitForServicesToStart() error {
	timeout := time.After(30 * time.Second)
	tick := time.NewTicker(time.Second)

	cmd := exec.Command("docker-compose", "-f", e.dockerComposePath, "ps")

	for {
		select {
		case <-timeout:
			return fmt.Errorf("timed out waiting for PoS L1 test environment services to start")
		case <-tick.C:
			if output, err := cmd.Output(); err != nil {
				return fmt.Errorf("failed to execute docker-compose ps command: %w", err)
			} else {
				lines := strings.Split(string(output), "\n")
				var beaconChainRunning bool
				var validatorRunning bool
				var gethRunning bool
				for _, line := range lines {
					fmt.Println("line", line)
					if strings.Contains(line, "beacon") {
						beaconChainRunning = true
					}
					if strings.Contains(line, "validator") {
						validatorRunning = true
					}
					if strings.Contains(line, "geth") {
						gethRunning = true
					}
				}
				if beaconChainRunning && validatorRunning && gethRunning {
					return nil
				}
				log.Info("Required services are not running, waiting...", "beaconChainRunning", beaconChainRunning, "validatorRunning", validatorRunning, "gethRunning", gethRunning)
			}
		}
	}
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
