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

type TestcontainerApps struct {
	postgresContainer *postgres.PostgresContainer
	l1GethContainer   *testcontainers.DockerContainer
	l2GethContainer   *testcontainers.DockerContainer

	dbClient     *sql.DB
	DBConfig     *database.DBConfig
	DBConfigFile string

	// common time stamp.
	Timestamp int
}

func NewTestcontainerApps() *TestcontainerApps {
	timestamp := time.Now().Nanosecond()
	return &TestcontainerApps{
		DBConfigFile: fmt.Sprintf("/tmp/%d_db-config.json", timestamp),
		Timestamp:    timestamp,
	}
}

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
		log.Printf("failed to start container: %s", err)
		return err
	}
	t.postgresContainer = postgresContainer
	return nil
}

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

func (t *TestcontainerApps) GetDBEndPoint() (string, error) {
	if t.postgresContainer == nil || !t.postgresContainer.IsRunning() {
		return "", fmt.Errorf("postgres is not running")
	}
	return t.postgresContainer.ConnectionString(context.Background(), "sslmode=disable")
}

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

func (t *TestcontainerApps) Free(ctx context.Context) {
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
