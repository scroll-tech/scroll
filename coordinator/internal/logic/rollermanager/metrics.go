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

func (r *proverManager) UpdateMetricRollerProofsLastFinishedTimestampGauge(pk string) {
	if node, ok := r.proverPool.Get(pk); ok {
		rMs := node.(*proverNode).metrics
		if rMs != nil {
			rMs.proverProofsLastFinishedTimestampGauge.Update(time.Now().Unix())
		}
	}
}

func (r *proverManager) UpdateMetricRollerProofsLastAssignedTimestampGauge(pk string) {
	if node, ok := r.proverPool.Get(pk); ok {
		rMs := node.(*proverNode).metrics
		if rMs != nil {
			rMs.proverProofsLastAssignedTimestampGauge.Update(time.Now().Unix())
		}
	}
}

func (r *proverManager) UpdateMetricRollerProofsVerifiedSuccessTimeTimer(pk string, d time.Duration) {
	if node, ok := r.proverPool.Get(pk); ok {
		rMs := node.(*proverNode).metrics
		if rMs != nil {
			rMs.proverProofsVerifiedSuccessTimeTimer.Update(d)
		}
	}
}

func (r *proverManager) UpdateMetricRollerProofsVerifiedFailedTimeTimer(pk string, d time.Duration) {
	if node, ok := r.proverPool.Get(pk); ok {
		rMs := node.(*proverNode).metrics
		if rMs != nil {
			rMs.proverProofsVerifiedFailedTimeTimer.Update(d)
		}
	}
}

func (r *proverManager) UpdateMetricRollerProofsGeneratedFailedTimeTimer(pk string, d time.Duration) {
	if node, ok := r.proverPool.Get(pk); ok {
		rMs := node.(*proverNode).metrics
		if rMs != nil {
			rMs.proverProofsGeneratedFailedTimeTimer.Update(d)
		}
	}
}
