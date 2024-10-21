package testcontainers

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/compose"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	"gorm.io/gorm"

	"scroll-tech/common/database"
)

// TestcontainerApps testcontainers struct
type TestcontainerApps struct {
	postgresContainer   *postgres.PostgresContainer
	l2GethContainer     *testcontainers.DockerContainer
	poSL1Container      compose.ComposeStack
	web3SignerContainer *testcontainers.DockerContainer

	// common time stamp in nanoseconds.
	Timestamp int
}

// NewTestcontainerApps returns new instance of TestcontainerApps struct
func NewTestcontainerApps() *TestcontainerApps {
	timestamp := time.Now().Nanosecond()
	// In order to solve the problem of "creating reaper failed: failed to create container"
	// refer to https://github.com/testcontainers/testcontainers-go/issues/2172
	if err := os.Setenv("TESTCONTAINERS_RYUK_DISABLED", "true"); err != nil {
		panic("set env failed: " + err.Error())
	}
	return &TestcontainerApps{
		Timestamp: timestamp,
	}
}

// StartPostgresContainer starts a postgres container
func (t *TestcontainerApps) StartPostgresContainer() error {
	if t.postgresContainer != nil && t.postgresContainer.IsRunning() {
		return nil
	}
	postgresContainer, err := postgres.RunContainer(context.Background(),
		testcontainers.WithImage("postgres"),
		postgres.WithDatabase("test_db"),
		postgres.WithPassword("123456"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").WithOccurrence(2).WithStartupTimeout(5*time.Second)),
	)
	if err != nil {
		log.Printf("failed to start postgres container: %s", err)
		return err
	}
	t.postgresContainer = postgresContainer
	return nil
}

// StartL2GethContainer starts a L2Geth container
func (t *TestcontainerApps) StartL2GethContainer() error {
	if t.l2GethContainer != nil && t.l2GethContainer.IsRunning() {
		return nil
	}
	req := testcontainers.ContainerRequest{
		Image:        "scroll_l2geth",
		ExposedPorts: []string{"8546/tcp", "8545/tcp"},
		WaitingFor: wait.ForAll(
			wait.ForListeningPort("8546").WithStartupTimeout(100*time.Second),
			wait.ForListeningPort("8545").WithStartupTimeout(100*time.Second),
		),
	}
	genericContainerReq := testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	}
	container, err := testcontainers.GenericContainer(context.Background(), genericContainerReq)
	if err != nil {
		log.Printf("failed to start scroll_l2geth container: %s", err)
		return err
	}
	t.l2GethContainer, _ = container.(*testcontainers.DockerContainer)
	return nil
}

// StartPoSL1Container starts the PoS L1 container by running the associated Docker Compose configuration
func (t *TestcontainerApps) StartPoSL1Container() error {
	var (
		err               error
		rootDir           string
		dockerComposeFile string
	)

	if rootDir, err = findProjectRootDir(); err != nil {
		return fmt.Errorf("failed to find project root directory: %v", err)
	}

	dockerComposeFile = filepath.Join(rootDir, "common", "testcontainers", "docker-compose.yml")

	if t.poSL1Container, err = compose.NewDockerCompose([]string{dockerComposeFile}...); err != nil {
		return err
	}
	if err = t.poSL1Container.WaitForService("geth", wait.ForListeningPort("8545").WithStartupTimeout(15*time.Second)).Up(context.Background()); err != nil {
		t.poSL1Container = nil
		return fmt.Errorf("failed to start PoS L1 container: %w", err)
	}
	return nil
}

func (t *TestcontainerApps) StartWeb3SignerContainer(chainId int) error {
	if t.web3SignerContainer != nil && t.web3SignerContainer.IsRunning() {
		return nil
	}
	var (
		err     error
		rootDir string
	)
	if rootDir, err = findProjectRootDir(); err != nil {
		return fmt.Errorf("failed to find project root directory: %v", err)
	}

	// web3signerconf/keyconf.yaml may contain multiple keys configured and web3signer then choses one corresponding to from field of tx
	web3SignerConfDir := filepath.Join(rootDir, "common", "testcontainers", "web3signerconf")

	req := testcontainers.ContainerRequest{
		Image:        "consensys/web3signer:develop",
		ExposedPorts: []string{"9000/tcp"},
		Cmd:          []string{"--key-config-path", "/web3signerconf/", "eth1", "--chain-id", fmt.Sprintf("%d", chainId)},
		Files: []testcontainers.ContainerFile{
			{
				HostFilePath:      web3SignerConfDir,
				ContainerFilePath: "/",
				FileMode:          0o777,
			},
		},
		WaitingFor: wait.ForLog("ready to handle signing requests"),
	}
	genericContainerReq := testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	}
	container, err := testcontainers.GenericContainer(context.Background(), genericContainerReq)
	if err != nil {
		log.Printf("failed to start web3signer container: %s", err)
		return err
	}
	t.web3SignerContainer, _ = container.(*testcontainers.DockerContainer)
	return nil
}

// GetPoSL1EndPoint returns the endpoint of the running PoS L1 endpoint
func (t *TestcontainerApps) GetPoSL1EndPoint() (string, error) {
	if t.poSL1Container == nil {
		return "", errors.New("PoS L1 container is not running")
	}
	contrainer, err := t.poSL1Container.ServiceContainer(context.Background(), "geth")
	if err != nil {
		return "", err
	}
	return contrainer.PortEndpoint(context.Background(), "8545/tcp", "http")
}

// GetPoSL1Client returns a ethclient by dialing running PoS L1 client
func (t *TestcontainerApps) GetPoSL1Client() (*ethclient.Client, error) {
	endpoint, err := t.GetPoSL1EndPoint()
	if err != nil {
		return nil, err
	}
	return ethclient.Dial(endpoint)
}

// GetDBEndPoint returns the endpoint of the running postgres container
func (t *TestcontainerApps) GetDBEndPoint() (string, error) {
	if t.postgresContainer == nil || !t.postgresContainer.IsRunning() {
		return "", errors.New("postgres is not running")
	}
	return t.postgresContainer.ConnectionString(context.Background(), "sslmode=disable")
}

// GetL2GethEndPoint returns the endpoint of the running L2Geth container
func (t *TestcontainerApps) GetL2GethEndPoint() (string, error) {
	if t.l2GethContainer == nil || !t.l2GethContainer.IsRunning() {
		return "", errors.New("l2 geth is not running")
	}
	endpoint, err := t.l2GethContainer.PortEndpoint(context.Background(), "8546/tcp", "ws")
	if err != nil {
		return "", err
	}
	return endpoint, nil
}

// GetWeb3SignerEndpoint returns the endpoint of the running L2Geth container
func (t *TestcontainerApps) GetWeb3SignerEndpoint() (string, error) {
	if t.web3SignerContainer == nil || !t.web3SignerContainer.IsRunning() {
		return "", errors.New("web3signer is not running")
	}
	return t.web3SignerContainer.PortEndpoint(context.Background(), "9000/tcp", "http")
}

// GetGormDBClient returns a gorm.DB by connecting to the running postgres container
func (t *TestcontainerApps) GetGormDBClient() (*gorm.DB, error) {
	endpoint, err := t.GetDBEndPoint()
	if err != nil {
		return nil, err
	}
	dbCfg := &database.Config{
		DSN:        endpoint,
		DriverName: "postgres",
		MaxOpenNum: 200,
		MaxIdleNum: 20,
	}
	return database.InitDB(dbCfg)
}

// GetL2GethClient returns a ethclient by dialing running L2Geth
func (t *TestcontainerApps) GetL2GethClient() (*ethclient.Client, error) {
	endpoint, err := t.GetL2GethEndPoint()
	if err != nil {
		return nil, err
	}
	client, err := ethclient.Dial(endpoint)
	if err != nil {
		return nil, err
	}
	return client, nil
}

// Free stops all running containers
func (t *TestcontainerApps) Free() {
	ctx := context.Background()
	if t.postgresContainer != nil && t.postgresContainer.IsRunning() {
		if err := t.postgresContainer.Terminate(ctx); err != nil {
			log.Printf("failed to stop postgres container: %s", err)
		}
	}
	if t.l2GethContainer != nil && t.l2GethContainer.IsRunning() {
		if err := t.l2GethContainer.Terminate(ctx); err != nil {
			log.Printf("failed to stop scroll_l2geth container: %s", err)
		}
	}
	if t.poSL1Container != nil {
		if err := t.poSL1Container.Down(context.Background(), compose.RemoveOrphans(true), compose.RemoveVolumes(true), compose.RemoveImagesLocal); err != nil {
			log.Printf("failed to stop PoS L1 container: %s", err)
		} else {
			t.poSL1Container = nil
		}
	}
	if t.web3SignerContainer != nil && t.web3SignerContainer.IsRunning() {
		if err := t.web3SignerContainer.Terminate(ctx); err != nil {
			log.Printf("failed to stop web3signer container: %s", err)
		}
	}
}

// findProjectRootDir find project root directory
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
			return "", errors.New("go.work file not found in any parent directory")
		}

		currentDir = parentDir
	}
}
