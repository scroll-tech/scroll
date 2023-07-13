package rollermanager

import (
	"time"

	gethMetrics "github.com/scroll-tech/go-ethereum/metrics"
)

type rollerMetrics struct {
	rollerProofsVerifiedSuccessTimeTimer   gethMetrics.Timer
	rollerProofsVerifiedFailedTimeTimer    gethMetrics.Timer
	rollerProofsGeneratedFailedTimeTimer   gethMetrics.Timer
	rollerProofsLastAssignedTimestampGauge gethMetrics.Gauge
	rollerProofsLastFinishedTimestampGauge gethMetrics.Gauge
}

func (r *rollerManager) UpdateMetricRollerProofsLastFinishedTimestampGauge(pk string) {
	if node, ok := r.rollerPool.Get(pk); ok {
		rMs := node.(*rollerNode).rollerMetrics
		if rMs != nil {
			rMs.rollerProofsLastFinishedTimestampGauge.Update(time.Now().Unix())
		}
	}
}

func (r *rollerManager) UpdateMetricRollerProofsLastAssignedTimestampGauge(pk string) {
	if node, ok := r.rollerPool.Get(pk); ok {
		rMs := node.(*rollerNode).rollerMetrics
		if rMs != nil {
			rMs.rollerProofsLastAssignedTimestampGauge.Update(time.Now().Unix())
		}
	}
}

func (r *rollerManager) UpdateMetricRollerProofsVerifiedSuccessTimeTimer(pk string, d time.Duration) {
	if node, ok := r.rollerPool.Get(pk); ok {
		rMs := node.(*rollerNode).rollerMetrics
		if rMs != nil {
			rMs.rollerProofsVerifiedSuccessTimeTimer.Update(d)
		}
	}
}

func (r *rollerManager) UpdateMetricRollerProofsVerifiedFailedTimeTimer(pk string, d time.Duration) {
	if node, ok := r.rollerPool.Get(pk); ok {
		rMs := node.(*rollerNode).rollerMetrics
		if rMs != nil {
			rMs.rollerProofsVerifiedFailedTimeTimer.Update(d)
		}
	}
}

func (r *rollerManager) UpdateMetricRollerProofsGeneratedFailedTimeTimer(pk string, d time.Duration) {
	if node, ok := r.rollerPool.Get(pk); ok {
		rMs := node.(*rollerNode).rollerMetrics
		if rMs != nil {
			rMs.rollerProofsGeneratedFailedTimeTimer.Update(d)
		}
	}
}
