package fetcher

import "sync/atomic"

// SyncInfo is a struct that stores synchronization information shared between L1 fetcher and L2 fetcher.
type SyncInfo struct {
	l2ScanHeight uint64
}

// SetL2ScanHeight is a method that sets the value of l2ScanHeight in SyncInfo.
func (s *SyncInfo) SetL2ScanHeight(height uint64) {
	atomic.StoreUint64(&s.l2ScanHeight, height)
}

// GetL2ScanHeight is a method that retrieves the value of l2ScanHeight in SyncInfo.
func (s *SyncInfo) GetL2ScanHeight() uint64 {
	return atomic.LoadUint64(&s.l2ScanHeight)
}
