package testcontainers

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"scroll-tech/database"
	"time"

	"github.com/scroll-tech/go-ethereum/ethclient"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
)

type TestContainerApps struct {
	postgresContainer *postgres.PostgresContainer
	l1GethContainer   *testcontainers.DockerContainer
	l2GethContainer   *testcontainers.DockerContainer

	dbClient     *sql.DB
	DBConfig     *database.DBConfig
	DBConfigFile string

	// common time stamp.
	Timestamp int
}

func NewTestContainerApps() *TestContainerApps {
	timestamp := time.Now().Nanosecond()
	return &TestContainerApps{
		DBConfigFile: fmt.Sprintf("/tmp/%d_db-config.json", timestamp),
		Timestamp:    timestamp,
	}
}

func (t *TestContainerApps) StartPostgresContainer() (*postgres.PostgresContainer, error) {
	if t.postgresContainer != nil && t.postgresContainer.IsRunning() {
		return t.postgresContainer, nil
	}
	postgresContainer, err := postgres.RunContainer(context.Background(),
		testcontainers.WithImage("postgres"),
		postgres.WithDatabase("test_db"),
		postgres.WithPassword("123456"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").WithOccurrence(2).WithStartupTimeout(5*time.Second)),
	)
	if err != nil {
		log.Printf("failed to start container: %s", err)
		return nil, err
	}
	t.postgresContainer = postgresContainer
	return t.postgresContainer, nil
}

func (t *TestContainerApps) StartL1GethContainer() (*testcontainers.DockerContainer, error) {
	if t.l1GethContainer != nil && t.l1GethContainer.IsRunning() {
		return t.l1GethContainer, nil
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
		return nil, err
	}
	t.l1GethContainer, _ = container.(*testcontainers.DockerContainer)
	return t.l1GethContainer, nil
}

func (t *TestContainerApps) StartL2GethContainer() (*testcontainers.DockerContainer, error) {
	if t.l2GethContainer != nil && t.l2GethContainer.IsRunning() {
		return t.l2GethContainer, nil
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
		return nil, err
	}
	t.l2GethContainer, _ = container.(*testcontainers.DockerContainer)
	return t.l2GethContainer, nil
}

func (t *TestContainerApps) GetDBEndPoint() (string, error) {
	if t.postgresContainer == nil || !t.postgresContainer.IsRunning() {
		return "", fmt.Errorf("postgres is not running")
	}
	return t.postgresContainer.ConnectionString(context.Background(), "sslmode=disable")
}

func (t *TestContainerApps) GetL1GethEndPoint() (string, error) {
	if t.l1GethContainer == nil || !t.l1GethContainer.IsRunning() {
		return "", fmt.Errorf("l1 geth is not running")
	}
	endpoint, err := t.l1GethContainer.PortEndpoint(context.Background(), "8546/tcp", "ws")
	if err != nil {
		return "", err
	}
	return endpoint, nil
}

func (t *TestContainerApps) GetL2GethEndPoint() (string, error) {
	if t.l2GethContainer == nil || !t.l2GethContainer.IsRunning() {
		return "", fmt.Errorf("l2 geth is not running")
	}
	endpoint, err := t.l2GethContainer.PortEndpoint(context.Background(), "8546/tcp", "ws")
	if err != nil {
		return "", err
	}
	return endpoint, nil
}

func (t *TestContainerApps) GetL1GethClient() (*ethclient.Client, error) {
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

func (t *TestContainerApps) GetL2GethClient() (*ethclient.Client, error) {
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

func (t *TestContainerApps) Free(ctx context.Context) {
	if t.postgresContainer != nil && t.postgresContainer.IsRunning() {
		t.postgresContainer.Terminate(ctx)
	}
	if t.l1GethContainer != nil && t.l1GethContainer.IsRunning() {
		t.l1GethContainer.Terminate(ctx)
	}
	if t.l2GethContainer != nil && t.l2GethContainer.IsRunning() {
		t.l2GethContainer.Terminate(ctx)
	}
}
