package types

// TxStatusType represents the status of a transaction.
type TxStatusType int

// Constants for TxStatusType.
const (
	// TxStatusTypeSent is one of the initial statuses for cross-chain messages.
	// It is used as the default value to prevent overwriting the transaction status in scenarios where the message status might change
	// from a later status (e.g., relayed) back to "sent".
	// Example flow (L1 -> L2 message, and L1 fetcher is slower than L2 fetcher):
	// 1. The relayed message is first tracked and processed, setting tx_status to TxStatusTypeRelayed.
	// 2. The sent message is later processed (same cross-chain message), the tx_status should not over-write TxStatusTypeRelayed.
	TxStatusTypeSent           TxStatusType = iota
	TxStatusTypeSentTxReverted              // Not track message hash, thus will not be processed again anymore.
	TxStatusTypeRelayed                     // Terminal status.
	// TxStatusTypeFailedRelayed Retry: this often occurs due to an out of gas (OOG) issue if the transaction was initiated via the frontend.
	TxStatusTypeFailedRelayed
	// TxStatusTypeRelayTxReverted Retry: this often occurs due to an out of gas (OOG) issue if the transaction was initiated via the frontend.
	TxStatusTypeRelayTxReverted
	TxStatusTypeSkipped
	TxStatusTypeDropped // Terminal status.

	// TxStatusBridgeBatchDeposit use deposit token to bridge batch deposit contract
	TxStatusBridgeBatchDeposit
	// TxStatusBridgeBatchDistribute bridge batch deposit contract distribute tokens to user success
	TxStatusBridgeBatchDistribute
	// TxStatusBridgeBatchDistributeFailed bridge batch deposit contract distribute tokens to user failed
	TxStatusBridgeBatchDistributeFailed
)

// TokenType represents the type of token.
type TokenType int

// Constants for TokenType.
const (
	TokenTypeUnknown TokenType = iota
	TokenTypeETH
	TokenTypeERC20
	TokenTypeERC721
	TokenTypeERC1155
)

// MessageType represents the type of message.
type MessageType int

// Constants for MessageType.
const (
	MessageTypeUnknown MessageType = iota
	MessageTypeL1SentMessage
	MessageTypeL2SentMessage
	MessageTypeL1BatchDeposit
)

// RollupStatusType represents the status of a rollup.
type RollupStatusType int

// Constants for RollupStatusType.
const (
	RollupStatusTypeUnknown   RollupStatusType = iota
	RollupStatusTypeFinalized                  // only batch finalized status is used.
)

// MessageQueueEventType represents the type of message queue event.
type MessageQueueEventType int

// Constants for MessageQueueEventType.
const (
	MessageQueueEventTypeUnknown MessageQueueEventType = iota
	MessageQueueEventTypeQueueTransaction
	MessageQueueEventTypeDequeueTransaction
	MessageQueueEventTypeDropTransaction
	MessageQueueEventTypeResetDequeuedTransaction
)

// BatchStatusType represents the type of batch status.
type BatchStatusType int

// Constants for BatchStatusType.
const (
	BatchStatusTypeUnknown BatchStatusType = iota
	BatchStatusTypeCommitted
	BatchStatusTypeReverted
	BatchStatusTypeFinalized
)

// UpdateStatusType represents the whether batch info is updated in message table.
type UpdateStatusType int

// Constants for UpdateStatusType.
const (
	UpdateStatusTypeUnupdated UpdateStatusType = iota
	UpdateStatusTypeUpdated
)
