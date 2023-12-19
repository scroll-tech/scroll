package fetcher

import "sync/atomic"

// SyncInfo is a struct that stores synchronization information shared between L1 fetcher and L2 fetcher.
type SyncInfo struct {
	l2SyncHeight uint64
}

// SetL2SyncHeight is a method that sets the value of l2SyncHeight in SyncInfo.
func (s *SyncInfo) SetL2SyncHeight(height uint64) {
	atomic.StoreUint64(&s.l2SyncHeight, height)
}

// GetL2SyncHeight is a method that retrieves the value of l2SyncHeight in SyncInfo.
func (s *SyncInfo) GetL2SyncHeight() uint64 {
	return atomic.LoadUint64(&s.l2SyncHeight)
}
