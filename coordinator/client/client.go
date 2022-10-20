package client

import (
	"context"

	"github.com/scroll-tech/go-ethereum/rpc"

	"scroll-tech/coordinator"
)

type client interface {
	coordinator.RollerClient
}

// Client defines typed wrappers for the Ethereum RPC API.
type Client struct {
	client
}

// Dial connects a client to the given URL.
func Dial(rawurl string) (*Client, error) {
	return DialContext(context.Background(), rawurl)
}

// nolint
func DialContext(ctx context.Context, rawurl string) (*Client, error) {
	c, err := rpc.DialContext(ctx, rawurl)
	if err != nil {
		return nil, err
	}
	return NewClient(c), nil
}

// NewClient creates a client that uses the given RPC client.
func NewClient(c *rpc.Client) *Client {
	return &Client{client: client(&coordinator.Client{Client: c})}
}
