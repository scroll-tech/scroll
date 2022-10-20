package client

import (
	"context"

	"github.com/scroll-tech/go-ethereum"
	"github.com/scroll-tech/go-ethereum/rpc"

	"scroll-tech/common/message"
)

// Client wrap the ethclient websocket for roller.
type Client struct {
	*rpc.Client
}

// Dial connects a client to the given URL.
func Dial(rawurl string) (*Client, error) {
	return DialContext(context.Background(), rawurl)
}

// DialContext connects a client to the given URL and context.
func DialContext(ctx context.Context, rawurl string) (*Client, error) {
	c, err := rpc.DialContext(ctx, rawurl)
	if err != nil {
		return nil, err
	}
	return NewClient(c), nil
}

// NewClient creates a client that uses the given RPC client.
func NewClient(c *rpc.Client) *Client {
	return &Client{Client: c}
}

// RegisterAndSubscribe register roller into scroll and subscribe blocktraces from scroll.
func (c *Client) RegisterAndSubscribe(ctx context.Context, traceChan chan *message.BlockTraces, authMsg *message.AuthMessage) (ethereum.Subscription, error) {
	return c.Subscribe(ctx, "roller", traceChan, "register", authMsg)
}

// SubmitProof submit proof into scroll.
func (c *Client) SubmitProof(ctx context.Context, proof *message.AuthZkProof) (bool, error) {
	var ok bool
	return ok, c.CallContext(ctx, &ok, "roller_submitProof", proof)
}
