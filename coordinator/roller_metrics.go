package coordinator

import (
	"time"

	geth_metrics "github.com/scroll-tech/go-ethereum/metrics"
)

type rollerMetrics struct {
	rollerProofsVerifiedSuccessTimeTimer   geth_metrics.Timer
	rollerProofsVerifiedFailedTimeTimer    geth_metrics.Timer
	rollerProofsGeneratedFailedTimeTimer   geth_metrics.Timer
	rollerProofsLastAssignedTimestampGauge geth_metrics.Gauge
	rollerProofsLastFinishedTimestampGauge geth_metrics.Gauge
}

func (m *Manager) updateMetricRollerProofsLastFinishedTimestampGauge(pk string) {
	if node, ok := m.rollerPool.Get(pk); ok {
		rMs := node.(*rollerNode).metrics
		if rMs != nil {
			rMs.rollerProofsLastFinishedTimestampGauge.Update(time.Now().Unix())
		}
	}
}

func (m *Manager) updateMetricRollerProofsLastAssignedTimestampGauge(pk string) {
	if node, ok := m.rollerPool.Get(pk); ok {
		rMs := node.(*rollerNode).metrics
		if rMs != nil {
			rMs.rollerProofsLastAssignedTimestampGauge.Update(time.Now().Unix())
		}
	}
}

func (m *Manager) updateMetricRollerProofsVerifiedSuccessTimeTimer(pk string, d time.Duration) {
	if node, ok := m.rollerPool.Get(pk); ok {
		rMs := node.(*rollerNode).metrics
		if rMs != nil {
			rMs.rollerProofsVerifiedSuccessTimeTimer.Update(d)
		}
	}
}

func (m *Manager) updateMetricRollerProofsVerifiedFailedTimeTimer(pk string, d time.Duration) {
	if node, ok := m.rollerPool.Get(pk); ok {
		rMs := node.(*rollerNode).metrics
		if rMs != nil {
			rMs.rollerProofsVerifiedFailedTimeTimer.Update(d)
		}
	}
}

func (m *Manager) updateMetricRollerProofsGeneratedFailedTimeTimer(pk string, d time.Duration) {
	if node, ok := m.rollerPool.Get(pk); ok {
		rMs := node.(*rollerNode).metrics
		if rMs != nil {
			rMs.rollerProofsGeneratedFailedTimeTimer.Update(d)
		}
	}
}
