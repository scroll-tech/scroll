package client

import (
	"context"

	"github.com/scroll-tech/go-ethereum"

	"scroll-tech/common/message"
)

// RequestToken generates token for roller
func (c *Client) RequestToken(ctx context.Context, authMsg *message.AuthMessage) (string, error) {
	var token string
	err := c.CallContext(ctx, &token, "roller_requestToken", authMsg)
	return token, err
}

// RegisterAndSubscribe subscribe roller and register, verified by sign data.
func (c *Client) RegisterAndSubscribe(ctx context.Context, traceChan chan *message.TaskMsg, authMsg *message.AuthMessage) (ethereum.Subscription, error) {
	return c.Subscribe(ctx, "roller", traceChan, "register", authMsg)
}

// SubmitProof get proof from roller.
func (c *Client) SubmitProof(ctx context.Context, proof *message.AuthZkProof) (bool, error) {
	var ok bool
	return ok, c.CallContext(ctx, &ok, "roller_submitProof", proof)
}
