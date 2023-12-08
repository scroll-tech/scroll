package fetcher

import "sync/atomic"

type SyncInfo struct {
	l2ScanHeight uint64
}

func (s *SyncInfo) SetL2ScanHeight(height uint64) {
	atomic.StoreUint64(&s.l2ScanHeight, height)
}

func (s *SyncInfo) GetL2ScanHeight() uint64 {
	return atomic.LoadUint64(&s.l2ScanHeight)
}
