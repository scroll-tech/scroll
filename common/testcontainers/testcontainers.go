package testcontainers

import (
	"context"
	"crypto/rand"
	"fmt"
	"log"
	"math/big"
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
	postgresContainer *postgres.PostgresContainer
	l1GethContainer   *testcontainers.DockerContainer
	l2GethContainer   *testcontainers.DockerContainer
	poSL1Container    compose.ComposeStack

	// common time stamp in nanoseconds.
	Timestamp     int
	poSL1GethPort int64
}

// NewTestcontainerApps returns new instance of TestcontainerApps struct
func NewTestcontainerApps() *TestcontainerApps {
	timestamp := time.Now().Nanosecond()
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

// StartL1GethContainer starts a L1Geth container
func (t *TestcontainerApps) StartL1GethContainer() error {
	if t.l1GethContainer != nil && t.l1GethContainer.IsRunning() {
		return nil
	}
	req := testcontainers.ContainerRequest{
		Image:        "scroll_l1geth",
		ExposedPorts: []string{"8546/tcp", "8545/tcp"},
		WaitingFor: wait.ForAll(
			wait.ForListeningPort("8546").WithStartupTimeout(100*time.Second),
			wait.ForListeningPort("8545").WithStartupTimeout(100*time.Second),
		),
		Cmd: []string{"--log.debug", "ANY"},
	}
	genericContainerReq := testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	}
	container, err := testcontainers.GenericContainer(context.Background(), genericContainerReq)
	if err != nil {
		log.Printf("failed to start scroll_l1geth container: %s", err)
		return err
	}
	t.l1GethContainer, _ = container.(*testcontainers.DockerContainer)
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
		hostPath          string
		found             bool
		rnd               *big.Int
		dockerComposeFile string
		env               = map[string]string{}
	)

	if rootDir, err = findProjectRootDir(); err != nil {
		return fmt.Errorf("failed to find project root directory: %v", err)
	}

	if rnd, err = rand.Int(rand.Reader, big.NewInt(65536-1024)); err != nil {
		return fmt.Errorf("failed to generate a random: %v", err)
	}
	t.poSL1GethPort = int64(int(rnd.Int64()) + 1024)

	if err = os.Setenv("GETH_HTTP_PORT", fmt.Sprintf("%d", t.poSL1GethPort)); err != nil {
		return fmt.Errorf("failed to set GETH_HTTP_PORT: %v", err)
	}
	dockerComposeFile = filepath.Join(rootDir, "common", "testcontainers", "docker-compose.yml")

	if t.poSL1Container, err = compose.NewDockerCompose([]string{dockerComposeFile}...); err != nil {
		return err
	}
	env["GETH_HTTP_PORT"] = fmt.Sprintf("%d", t.poSL1GethPort)
	if hostPath, found = os.LookupEnv("HOST_PATH"); found {
		env["HOST_PATH"] = hostPath
	}
	err = t.poSL1Container.WaitForService("geth", wait.NewHTTPStrategy("/").
		WithPort("8545/tcp").
		WithStartupTimeout(15*time.Second)).
		WithEnv(env).
		Up(context.Background())
	if err != nil {
		t.poSL1Container = nil
		t.poSL1GethPort = 0
		return fmt.Errorf("failed to start PoS L1 container: %w", err)
	}
	return nil
}

// GetPoSL1EndPoint returns the endpoint of the running PoS L1 endpoint
func (t *TestcontainerApps) GetPoSL1EndPoint() (string, error) {
	if t.poSL1Container == nil || t.poSL1GethPort == int64(0) {
		return "", fmt.Errorf("PoS L1 container is not running")
	}
	return fmt.Sprintf("http://127.0.0.1:%d", t.poSL1GethPort), nil
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
		return "", fmt.Errorf("postgres is not running")
	}
	return t.postgresContainer.ConnectionString(context.Background(), "sslmode=disable")
}

// GetL1GethEndPoint returns the endpoint of the running L1Geth container
func (t *TestcontainerApps) GetL1GethEndPoint() (string, error) {
	if t.l1GethContainer == nil || !t.l1GethContainer.IsRunning() {
		return "", fmt.Errorf("l1 geth is not running")
	}
	endpoint, err := t.l1GethContainer.PortEndpoint(context.Background(), "8546/tcp", "ws")
	if err != nil {
		return "", err
	}
	return endpoint, nil
}

// GetL2GethEndPoint returns the endpoint of the running L2Geth container
func (t *TestcontainerApps) GetL2GethEndPoint() (string, error) {
	if t.l2GethContainer == nil || !t.l2GethContainer.IsRunning() {
		return "", fmt.Errorf("l2 geth is not running")
	}
	endpoint, err := t.l2GethContainer.PortEndpoint(context.Background(), "8546/tcp", "ws")
	if err != nil {
		return "", err
	}
	return endpoint, nil
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

// GetL1GethClient returns a ethclient by dialing running L1Geth
func (t *TestcontainerApps) GetL1GethClient() (*ethclient.Client, error) {
	endpoint, err := t.GetL1GethEndPoint()
	if err != nil {
		return nil, err
	}
	client, err := ethclient.Dial(endpoint)
	if err != nil {
		return nil, err
	}
	return client, nil
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
	if t.l1GethContainer != nil && t.l1GethContainer.IsRunning() {
		if err := t.l1GethContainer.Terminate(ctx); err != nil {
			log.Printf("failed to stop scroll_l1geth container: %s", err)
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
			t.poSL1GethPort = 0
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
			return "", fmt.Errorf("go.work file not found in any parent directory")
		}

		currentDir = parentDir
	}
}
