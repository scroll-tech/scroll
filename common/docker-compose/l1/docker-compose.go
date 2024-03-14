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
	httpPort int
	workDir  string
}

// NewPoSL1TestEnv creates and initializes a new instance of PoSL1TestEnv with a random HTTP port.
func NewPoSL1TestEnv() (*PoSL1TestEnv, error) {
	id, err := rand.Int(rand.Reader, big.NewInt(10000))
	if err != nil {
		return nil, fmt.Errorf("failed to generate random number: %v", err)
	}
	httpPort := int(id.Int64() + 50000)

	if err = os.Setenv("HTTP_PORT", strconv.Itoa(httpPort)); err != nil {
		return nil, fmt.Errorf("failed to set environment variable: %w", err)
	}

	rootDir, err := findProjectRootDir()
	if err != nil {
		return nil, fmt.Errorf("failed to find project root directory: %v", err)
	}

	return &PoSL1TestEnv{
		httpPort: httpPort,
		workDir:  filepath.Join(rootDir, "common", "docker-compose", "l1"),
	}, nil
}

// Start starts the PoS L1 test environment by running the associated Docker Compose configuration.
func (e *PoSL1TestEnv) Start() error {
	var err error
	defer func() {
		if err != nil {
			if err = e.stopAndCleanUpPoSL1(); err != nil {
				log.Error("failed to stop and clean up PoS L1 test environment", "err", err)
			}
		}
	}()

	currentDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}
	defer func() {
		if err = os.Chdir(currentDir); err != nil {
			log.Error("failed to restore directory", "error", err)
		}
	}()

	if err = os.Chdir(e.workDir); err != nil {
		return fmt.Errorf("failed to change directory: %w", err)
	}

	if err = e.cleanUp(); err != nil {
		return fmt.Errorf("failed to clean up: %w", err)
	}

	if err = e.runPoSL1DockerCompose(); err != nil {
		return fmt.Errorf("failed to start PoS L1 test environment: %w", err)
	}

	if err = e.waitForServicesToStart(); err != nil {
		return fmt.Errorf("failed to wait for PoS L1 test environment services: %w", err)
	}

	return nil
}

// Stop stops the PoS L1 test environment by stopping and removing the associated Docker Compose services.
func (e *PoSL1TestEnv) Stop() error {
	currentDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}
	defer func() {
		if err = os.Chdir(currentDir); err != nil {
			log.Error("failed to restore directory", "error", err)
		}
	}()

	if err := os.Chdir(e.workDir); err != nil {
		return fmt.Errorf("failed to change directory: %w", err)
	}

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

	client, err := ethclient.Dial(e.Endpoint())
	if err != nil {
		return nil, fmt.Errorf("failed to dial PoS L1 test environment: %w", err)
	}
	return client, nil
}

func (e *PoSL1TestEnv) runPoSL1DockerCompose() error {
	log.Info("start pos L1 devnet env", "dir", e.workDir)
	cmd := exec.Command("docker-compose", "-p", "posl1", "up", "-d")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run docker-compose up command: %w", err)
	}
	return nil
}

func (e *PoSL1TestEnv) stopAndCleanUpPoSL1() error {
	log.Info("shut down pos L1 devnet env", "dir", e.workDir)
	cmd := exec.Command("docker-compose", "-p", "posl1", "down")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run docker-compose down command: %w", err)
	}
	if err := e.cleanUp(); err != nil {
		return fmt.Errorf("failed to clean up: %w", err)
	}
	return nil
}

func (e *PoSL1TestEnv) cleanUp() error {
	log.Info("clean up pos L1 devnet env", "dir", e.workDir)
	cmd := exec.Command("/bin/bash", "clean.sh")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to execute clean.sh script: %w", err)
	}
	return nil
}

func (e *PoSL1TestEnv) waitForServicesToStart() error {
	timeout := time.After(30 * time.Second)
	tick := time.NewTicker(time.Second)

	cmd := exec.Command("docker-compose", "-p", "posl1", "ps")

	for {
		select {
		case <-timeout:
			return fmt.Errorf("timed out waiting for PoS L1 test environment services to start")
		case <-tick.C:
			output, err := cmd.Output()
			if err != nil {
				return fmt.Errorf("failed to execute docker-compose ps command: %w", err)
			}
			lines := strings.Split(string(output), "\n")
			var beaconChainRunning bool
			var validatorRunning bool
			var gethRunning bool
			for _, line := range lines {
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
				client, err := ethclient.Dial(e.Endpoint())
				if err == nil {
					log.Info("Probe works successfully", "endpoint", e.Endpoint())
					client.Close()
					return nil
				}
				log.Info("Geth is not yet available, waiting...", "error", err)
				return nil
			}
			log.Info("Required services are not running, waiting...", "beaconChainRunning", beaconChainRunning, "validatorRunning", validatorRunning, "gethRunning", gethRunning)
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
