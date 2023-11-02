package relayer

import "context"

type SendProofServiceServer interface {
	start(ctx context.Context) error
	stop()
	pushProofHash(msg proofHash) error
	handleHistoryProofTxs(ctx context.Context)
}
