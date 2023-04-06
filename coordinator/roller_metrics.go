package coordinator

import (
	"time"

	"github.com/scroll-tech/go-ethereum/log"
	geth_metrics "github.com/scroll-tech/go-ethereum/metrics"
)

type rollerMetrics struct {
	rollerProofsVerifiedSuccessTimeTimer   geth_metrics.Timer
	rollerProofsVerifiedFailedTimeTimer    geth_metrics.Timer
	rollerProofsGenerationFailedTimeTimer  geth_metrics.Timer
	rollerProofsLastAssignedTimestampGauge geth_metrics.Gauge
	rollerProofsLastFinishedTimestampGauge geth_metrics.Gauge
}

func (m *Manager) updateMetricRollerProofsLastFinishedTimestampGauge(pk string) {
	if node, ok := m.rollerPool.Get(pk); ok {
		rMs := node.(*rollerNode).rollerMetrics
		if rMs != nil {
			rMs.rollerProofsLastFinishedTimestampGauge.Update(time.Now().Unix())
		} else {
			log.Error("lastFinishedTimestamp metric is nil", "roller pk", pk)
		}
	}
}

func (m *Manager) updateMetricRollerProofsLastAssignedTimestampGauge(pk string) {
	if node, ok := m.rollerPool.Get(pk); ok {
		rMs := node.(*rollerNode).rollerMetrics
		if rMs != nil {
			rMs.rollerProofsLastAssignedTimestampGauge.Update(time.Now().Unix())
		} else {
			log.Error("lastAssignedTimestamp metric is nil", "roller pk", pk)
		}
	}
}

func (m *Manager) updateMetricRollerProofsVerifiedSuccessTimeTimer(pk string, d time.Duration) {
	if node, ok := m.rollerPool.Get(pk); ok {
		rMs := node.(*rollerNode).rollerMetrics
		if rMs != nil {
			rMs.rollerProofsVerifiedSuccessTimeTimer.Update(d)
		} else {
			log.Error("provingSuccessTime metric is nil", "roller pk", pk)
		}
	}
}

func (m *Manager) updateMetricRollerProvingFailedTimeTimer(pk string, d time.Duration) {
	if node, ok := m.rollerPool.Get(pk); ok {
		rMs := node.(*rollerNode).rollerMetrics
		if rMs != nil {
			rMs.rollerProofsVerifiedFailedTimeTimer.Update(d)
		} else {
			log.Error("provingFailedTime metric is nil", "roller pk", pk)
		}
	}
}
