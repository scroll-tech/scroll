package client

import (
	"context"

	"github.com/scroll-tech/go-ethereum"

	"scroll-tech/common/message"
)

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
