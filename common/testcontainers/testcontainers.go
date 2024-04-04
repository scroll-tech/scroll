package testcontainers

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

// TestcontainerApps testcontainers struct
type TestcontainerApps struct {
	postgresContainer *postgres.PostgresContainer
	l1GethContainer   *testcontainers.DockerContainer
	l2GethContainer   *testcontainers.DockerContainer

	// common time stamp in nanoseconds.
	Timestamp int
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
		WaitingFor:   wait.ForHTTP("/").WithPort("8545").WithStartupTimeout(100 * time.Second),
		Cmd:          []string{"--log.debug", "ANY"},
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
		WaitingFor:   wait.ForHTTP("/").WithPort("8545").WithStartupTimeout(100 * time.Second),
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
}
