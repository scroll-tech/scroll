package blob_uploader

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

type blobUploaderMetrics struct {
	rollupBlobUploaderUploadToS3SuccessTotal prometheus.Counter
	rollupBlobUploaderUploadToS3FailedTotal  prometheus.Counter
	rollupBlobUploaderUploadToArweaveSuccessTotal prometheus.Counter
	rollupBlobUploaderUploadToArweaveFailedTotal  prometheus.Counter
}

var (
	initBlobUploaderMetricsOnce sync.Once
	blobUploaderMetric          *blobUploaderMetrics
)

func initBlobUploaderMetrics(reg prometheus.Registerer) *blobUploaderMetrics {
	initBlobUploaderMetricsOnce.Do(func() {
		blobUploaderMetric = &blobUploaderMetrics{
			rollupBlobUploaderUploadToS3SuccessTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
				Name: "rollup_blob_uploader_upload_to_s3_success_total",
				Help: "The total number of upload blob to S3 runs success total",
			}),
			rollupBlobUploaderUploadToS3FailedTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
				Name: "rollup_blob_uploader_upload_to_s3_failed_total",
				Help: "The total number of upload blob to S3 runs failed total",
			}),
			rollupBlobUploaderUploadToArweaveSuccessTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
				Name: "rollup_blob_uploader_upload_to_arweave_success_total",
				Help: "The total number of upload blob to arweave runs success total",
			}),
			rollupBlobUploaderUploadToArweaveFailedTotal: promauto.With(reg).NewCounter(prometheus.CounterOpts{
				Name: "rollup_blob_uploader_upload_to_arweave_failed_total",
				Help: "The total number of upload blob to arweave runs failed total",
			}),
		}
	})
	return blobUploaderMetric
}
