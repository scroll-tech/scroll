package blob_uploader

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type blobUploaderMetrics struct {
	rollupBlobUploaderUploadToS3Total      prometheus.Counter
}

var (
	initBlobUploaderMetricsOnce sync.Once
	l1RelayerMetric         *blobUploaderMetrics
)

func initblobUploaderMetrics(reg prometheus.Registerer) *blobUploaderMetrics {
	initBlobUploaderMetricsOnce.Do(func() {
		l1RelayerMetric = &blobUploaderMetrics{
			rollupBlobUploaderUploadToS3Total: promauto.With(reg).NewCounter(prometheus.CounterOpts{
				Name: "rollup_blob_uploader_upload_to_s3_total",
				Help: "The total number of upload blob to S3 run total",
			}),
		}
	})
	return l1RelayerMetric
}
