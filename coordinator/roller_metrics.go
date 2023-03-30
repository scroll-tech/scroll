package coordinator

import (
	"time"

	"github.com/scroll-tech/go-ethereum/log"
	geth_metrics "github.com/scroll-tech/go-ethereum/metrics"
)

type rollerMetrics struct {
	rollerProofsProvingSuccessTimeTimer    geth_metrics.Timer
	rollerProofsProvingFailedTimeTimer     geth_metrics.Timer
	rollerProofsSuccessTotalCounter        geth_metrics.Counter
	rollerProofsFailedTotalCounter         geth_metrics.Counter
	rollerProofsLastAssignedTimestampGauge geth_metrics.Gauge
	rollerProofsLastFinishedTimestampGauge geth_metrics.Gauge
}

func (m *Manager) updateMetricRollerProofsSuccessTotal(pk string) {
	if node, ok := m.rollerPool.Get(pk); ok {
		rMs := node.(*rollerNode).rollerMetrics
		if rMs != nil {
			rMs.rollerProofsSuccessTotalCounter.Inc(1)
		} else {
			log.Error("successTotal metric is nil", "roller pk", pk)
		}
	}
}

func (m *Manager) updateMetricRollerProofsFailedTotal(pk string) {
	if node, ok := m.rollerPool.Get(pk); ok {
		rMs := node.(*rollerNode).rollerMetrics
		if rMs != nil {
			rMs.rollerProofsFailedTotalCounter.Inc(1)
		} else {
			log.Error("failedTotal metric is nil", "roller pk", pk)
		}
	}
}

func (m *Manager) updateMetricRollerProofsLastFinishedTimestamp(pk string) {
	if node, ok := m.rollerPool.Get(pk); ok {
		rMs := node.(*rollerNode).rollerMetrics
		if rMs != nil {
			rMs.rollerProofsLastFinishedTimestampGauge.Update(time.Now().Unix())
		} else {
			log.Error("lastFinishedTimestamp metric is nil", "roller pk", pk)
		}
	}
}

func (m *Manager) updateMetricRollerProofsLastAssignedTimestamp(pk string) {
	if node, ok := m.rollerPool.Get(pk); ok {
		rMs := node.(*rollerNode).rollerMetrics
		if rMs != nil {
			rMs.rollerProofsLastAssignedTimestampGauge.Update(time.Now().Unix())
		} else {
			log.Error("lastAssignedTimestamp metric is nil", "roller pk", pk)
		}
	}
}

func (m *Manager) updateMetricRollerProvingSuccessTimeTimer(pk string, d time.Duration) {
	if node, ok := m.rollerPool.Get(pk); ok {
		rMs := node.(*rollerNode).rollerMetrics
		if rMs != nil {
			rMs.rollerProofsProvingSuccessTimeTimer.Update(d)
		} else {
			log.Error("provingSuccessTime metric is nil", "roller pk", pk)
		}
	}
}

func (m *Manager) updateMetricRollerProvingFailedTimeTimer(pk string, d time.Duration) {
	if node, ok := m.rollerPool.Get(pk); ok {
		rMs := node.(*rollerNode).rollerMetrics
		if rMs != nil {
			rMs.rollerProofsProvingFailedTimeTimer.Update(d)
		} else {
			log.Error("provingFailedTime metric is nil", "roller pk", pk)
		}
	}
}
