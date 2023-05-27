package collector

type AggTaskCollector struct {
	CollectorType string
}

func NewAggTaskCollector() *AggTaskCollector {

}

func (atc *AggTaskCollector) Name() string {
	return AggTaskCollectorName
}

func (atc *AggTaskCollector) Collect() error {

}

func (atc *AggTaskCollector) Recover() {

}
