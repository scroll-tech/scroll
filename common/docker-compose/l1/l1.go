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
)

// PoSL1TestEnv represents the config needed to test in PoS Layer 1.
type PoSL1TestEnv struct {
	httpPort int
}

// NewPoSL1TestEnv creates and initializes a new instance of PoSL1TestEnv with a random HTTP port.
func NewPoSL1TestEnv() *PoSL1TestEnv {
	id, _ := rand.Int(rand.Reader, big.NewInt(10000))
	return &PoSL1TestEnv{
		httpPort: int(id.Int64() + 50000),
	}
}

// Start starts the PoS L1 test environment by running the associated Docker Compose configuration.
func (e *PoSL1TestEnv) Start() error {
	rootDir, err := findProjectRootDir()
	if err != nil {
		return fmt.Errorf("failed to find project root directory: %v", err)
	}

	dockerComposeFile := filepath.Join(rootDir, "common", "docker-compose", "l1", "docker-compose.yml")
	err = runPoSL1DockerCompose(e.httpPort, dockerComposeFile)
	if err != nil {
		return fmt.Errorf("failed to start PoS L1 test environment: %w", err)
	}

	err = waitForServices(dockerComposeFile)
	if err != nil {
		return fmt.Errorf("failed to wait for PoS L1 test environment services: %w", err)
	}

	return nil
}

// Stop stops the PoS L1 test environment by stopping and removing the associated Docker Compose services.
func (e *PoSL1TestEnv) Stop() error {
	rootDir, err := findProjectRootDir()
	if err != nil {
		return fmt.Errorf("failed to find project root directory: %v", err)
	}

	dockerComposeFile := filepath.Join(rootDir, "common", "docker-compose", "l1", "docker-compose.yml")

	err = stopAndCleanUpPoSL1(dockerComposeFile)
	if err != nil {
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

func runPoSL1DockerCompose(httpPort int, dockerComposeFile string) error {
	_, err := os.Stat(dockerComposeFile)
	if os.IsNotExist(err) {
		return fmt.Errorf("docker-compose.yml not found at %s", dockerComposeFile)
	}

	err = os.Setenv("HTTP_PORT", strconv.Itoa(httpPort))
	if err != nil {
		return fmt.Errorf("failed to set HTTP_PORT environment variable: %w", err)
	}

	cmd := exec.Command("docker-compose", "-f", dockerComposeFile, "up", "-d")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to run docker-compose up command: %w", err)
	}

	return nil
}

func stopAndCleanUpPoSL1(dockerComposeFile string) error {
	cmd := exec.Command("docker-compose", "-f", dockerComposeFile, "down")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to run docker-compose down command: %w", err)
	}
	return nil
}

func waitForServices(dockerComposeFile string) error {
	cmd := exec.Command("docker-compose", "-f", dockerComposeFile, "ps")

	var output []byte
	var err error

	timeout := time.After(30 * time.Second)
	tick := time.NewTicker(time.Second)

	for {
		select {
		case <-timeout:
			return fmt.Errorf("timed out waiting for PoS L1 test environment services to start")
		case <-tick.C:
			if output == nil {
				output, err = cmd.Output()
				if err != nil {
					if exitErr, ok := err.(*exec.ExitError); ok {
						fmt.Printf("docker-compose ps command failed with exit code %d\n", exitErr.ExitCode())
					} else {
						fmt.Printf("failed to execute docker-compose ps command: %v\n", err)
					}
					return fmt.Errorf("failed to execute docker-compose ps command: %w", err)
				}
				fmt.Printf("docker-compose ps output:\n%s\n", string(output))
			}

			lines := strings.Split(string(output), "\n")
			var beaconChainRunning bool
			var validatorRunning bool
			var gethRunning bool
			for _, line := range lines {
				fmt.Println("line", line)
				if strings.Contains(line, "beacon") {
					beaconChainRunning = true
					fmt.Println("Beacon chain service is running")
				}
				if strings.Contains(line, "validator") {
					validatorRunning = true
					fmt.Println("Validator service is running")
				}
				if strings.Contains(line, "geth") {
					gethRunning = true
					fmt.Println("Validator service is running")
				}
			}

			if beaconChainRunning && validatorRunning && gethRunning {
				fmt.Println("Required services are running")
				return nil
			}
			fmt.Println("Required services are not running, waiting...", "beaconChainRunning", beaconChainRunning, "validatorRunning", validatorRunning, "gethRunning", gethRunning)
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
