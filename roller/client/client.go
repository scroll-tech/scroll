package client

import (
	"context"

	"github.com/scroll-tech/go-ethereum"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/rpc"

	"scroll-tech/go-roller/message"
)

type rollerClient interface {
	SubscribeRegister(ctx context.Context, traceChan chan *types.BlockResult, authMsg *message.AuthMessage) (ethereum.Subscription, error)
	SubmitProof(ctx context.Context, proof *message.AuthZkProof) (bool, error)
}

type Client struct {
	*rpc.Client
}

type RollerClient struct {
	rollerClient
}

func (c *Client) SubscribeRegister(ctx context.Context, traceChan chan *types.BlockResult, authMsg *message.AuthMessage) (ethereum.Subscription, error) {
	return c.Subscribe(ctx, "roller", traceChan, "register", authMsg)
}

func (c *Client) SubmitProof(ctx context.Context, proof *message.AuthZkProof) (bool, error) {
	var ok bool
	return ok, c.CallContext(ctx, &ok, "roller_submitProof", proof)
}

// Dial connects a client to the given URL.
func Dial(rawurl string) (*RollerClient, error) {
	return DialContext(context.Background(), rawurl)
}

func DialContext(ctx context.Context, rawurl string) (*RollerClient, error) {
	c, err := rpc.DialContext(ctx, rawurl)
	if err != nil {
		return nil, err
	}
	return NewClient(c), nil
}

// NewClient creates a client that uses the given RPC client.
func NewClient(c *rpc.Client) *RollerClient {
	return &RollerClient{rollerClient: rollerClient(&Client{c})}
}
