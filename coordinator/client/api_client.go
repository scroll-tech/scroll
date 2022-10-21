package client

import (
	"context"

	"github.com/scroll-tech/go-ethereum"

	"scroll-tech/common/message"
)

// SubscribeRegister subscribe roller and register, verified by sign data.
func (c *Client) SubscribeRegister(ctx context.Context, traceChan chan *message.BlockTraces, authMsg *message.AuthMessage) (ethereum.Subscription, error) {
	return c.Subscribe(ctx, "roller", traceChan, "register", authMsg)
}

// SubmitProof get proof from roller.
func (c *Client) SubmitProof(ctx context.Context, proof *message.AuthZkProof) (bool, error) {
	var ok bool
	return ok, c.CallContext(ctx, &ok, "roller_submitProof", proof)
}
