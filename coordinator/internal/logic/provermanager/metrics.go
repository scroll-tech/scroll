package provermanager

import (
	"time"

	gethMetrics "github.com/scroll-tech/go-ethereum/metrics"
)

type proverMetrics struct {
	proverProofsVerifiedSuccessTimeTimer   gethMetrics.Timer
	proverProofsVerifiedFailedTimeTimer    gethMetrics.Timer
	proverProofsGeneratedFailedTimeTimer   gethMetrics.Timer
	proverProofsLastAssignedTimestampGauge gethMetrics.Gauge
	proverProofsLastFinishedTimestampGauge gethMetrics.Gauge
}

func (r *proverManager) UpdateMetricProverProofsLastFinishedTimestampGauge(pk string) {
	if node, ok := r.proverPool.Get(pk); ok {
		rMs := node.(*proverNode).metrics
		if rMs != nil {
			rMs.proverProofsLastFinishedTimestampGauge.Update(time.Now().Unix())
		}
	}
}

func (r *proverManager) UpdateMetricProverProofsLastAssignedTimestampGauge(pk string) {
	if node, ok := r.proverPool.Get(pk); ok {
		rMs := node.(*proverNode).metrics
		if rMs != nil {
			rMs.proverProofsLastAssignedTimestampGauge.Update(time.Now().Unix())
		}
	}
}

func (r *proverManager) UpdateMetricProverProofsVerifiedSuccessTimeTimer(pk string, d time.Duration) {
	if node, ok := r.proverPool.Get(pk); ok {
		rMs := node.(*proverNode).metrics
		if rMs != nil {
			rMs.proverProofsVerifiedSuccessTimeTimer.Update(d)
		}
	}
}

func (r *proverManager) UpdateMetricProverProofsVerifiedFailedTimeTimer(pk string, d time.Duration) {
	if node, ok := r.proverPool.Get(pk); ok {
		rMs := node.(*proverNode).metrics
		if rMs != nil {
			rMs.proverProofsVerifiedFailedTimeTimer.Update(d)
		}
	}
}

func (r *proverManager) UpdateMetricProverProofsGeneratedFailedTimeTimer(pk string, d time.Duration) {
	if node, ok := r.proverPool.Get(pk); ok {
		rMs := node.(*proverNode).metrics
		if rMs != nil {
			rMs.proverProofsGeneratedFailedTimeTimer.Update(d)
		}
	}
}
