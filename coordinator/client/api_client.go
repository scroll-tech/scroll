package client

import (
	"context"

	"github.com/scroll-tech/go-ethereum"

	"scroll-tech/common/message"
)

func (c *Client) RequestTicket(ctx context.Context, authMsg *message.AuthMessage) (message.Ticket, error) {
	var ticket message.Ticket
	err := c.CallContext(ctx, &ticket, "roller_requestTicket", authMsg)
	return ticket, err
}

// RegisterAndSubscribe subscribe roller and register, verified by sign data.
func (c *Client) RegisterAndSubscribe(ctx context.Context, traceChan chan *message.BlockTraces, authMsg *message.AuthMessage) (ethereum.Subscription, error) {
	return c.Subscribe(ctx, "roller", traceChan, "register", authMsg)
}

// SubmitProof get proof from roller.
func (c *Client) SubmitProof(ctx context.Context, proof *message.AuthZkProof) (bool, error) {
	var ok bool
	return ok, c.CallContext(ctx, &ok, "roller_submitProof", proof)
}
