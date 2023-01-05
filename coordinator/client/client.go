package client

import (
	"context"

	"github.com/scroll-tech/go-ethereum"
	"github.com/scroll-tech/go-ethereum/rpc"

	"scroll-tech/common/message"
)

// Client defines typed wrappers for the Ethereum RPC API.
type Client struct {
	client *rpc.Client
}

// Dial connects a client to the given URL.
func Dial(rawurl string) (*Client, error) {
	return DialContext(context.Background(), rawurl)
}

// DialContext connects a client to the given URL with a given context.
func DialContext(ctx context.Context, rawurl string) (*Client, error) {
	c, err := rpc.DialContext(ctx, rawurl)
	if err != nil {
		return nil, err
	}
	return NewClient(c), nil
}

// NewClient creates a client that uses the given RPC client.
func NewClient(c *rpc.Client) *Client {
	return &Client{client: c}
}

// RequestToken generates token for roller
func (c *Client) RequestToken(ctx context.Context, authMsg *message.AuthMsg) (string, error) {
	var token string
	err := c.client.CallContext(ctx, &token, "roller_requestToken", authMsg)
	return token, err
}

// RegisterAndSubscribe subscribe roller and register, verified by sign data.
func (c *Client) RegisterAndSubscribe(ctx context.Context, taskCh chan *message.TaskMsg, authMsg *message.AuthMsg) (ethereum.Subscription, error) {
	return c.client.Subscribe(ctx, "roller", taskCh, "register", authMsg)
}

// SubmitProof get proof from roller.
func (c *Client) SubmitProof(ctx context.Context, proof *message.ProofMsg) (bool, error) {
	var ok bool
	return ok, c.client.CallContext(ctx, &ok, "roller_submitProof", proof)
}
