package client

import (
	"context"
	"scroll-tech/coordinator/types"

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

// ------ debug API ----------

// ListRollers returns all live rollers
func (c *Client) ListRollers(ctx context.Context) ([]*types.RollerInfo, error) {
	var results []*types.RollerInfo
	return results, c.client.CallContext(ctx, &results, "debug_listRollers")
}

// GetSessionInfo returns the session information given the session id.
func (c *Client) GetSessionInfo(ctx context.Context, sessionID string) (*types.SessionInfo, error) {
	var info types.SessionInfo
	return &info, c.client.CallContext(ctx, &info, "debug_getSessionInfo", sessionID)
}
