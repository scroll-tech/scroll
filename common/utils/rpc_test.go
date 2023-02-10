package utils

import (
	"compress/flate"
	"context"
	"testing"

	"github.com/scroll-tech/go-ethereum/rpc"
	"github.com/stretchr/testify/assert"
)

type testService struct{}

type echoArgs struct {
	S string
}

type echoResult struct {
	Name string
	ID   int
	Args *echoArgs
}

func (s *testService) NoArgsRets() {}

func (s *testService) Echo(str string, i int, args *echoArgs) echoResult {
	return echoResult{str, i, args}
}

func TestStartHTTPEndpoint(t *testing.T) {
	endpoint := "localhost:18080"
	handler, _, err := StartHTTPEndpoint(endpoint, []rpc.API{
		{
			Public:    true,
			Namespace: "test",
			Service:   new(testService),
		},
	})
	assert.NoError(t, err)
	defer handler.Shutdown(context.Background())

	client, err := rpc.Dial("http://" + endpoint)
	assert.NoError(t, err)

	assert.NoError(t, client.Call(nil, "test_noArgsRets"))

	result := echoResult{}
	assert.NoError(t, client.Call(&result, "test_echo", "test", 0, &echoArgs{S: "test"}))
	assert.Equal(t, 0, result.ID)
	assert.Equal(t, "test", result.Name)

	defer client.Close()
}

func TestStartWSEndpoint(t *testing.T) {
	endpoint := "localhost:18081"
	handler, _, err := StartWSEndpoint(endpoint, []rpc.API{
		{
			Public:    true,
			Namespace: "test",
			Service:   new(testService),
		},
	}, flate.NoCompression)
	assert.NoError(t, err)
	defer handler.Shutdown(context.Background())

	client, err := rpc.Dial("ws://" + endpoint)
	assert.NoError(t, err)

	assert.NoError(t, client.Call(nil, "test_noArgsRets"))

	result := echoResult{}
	assert.NoError(t, client.Call(&result, "test_echo", "test", 0, &echoArgs{S: "test"}))
	assert.Equal(t, 0, result.ID)
	assert.Equal(t, "test", result.Name)

	defer client.Close()
}
