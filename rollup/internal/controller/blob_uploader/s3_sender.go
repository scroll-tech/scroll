package blob_uploader

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"scroll-tech/rollup/internal/config"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3 Uploader is responsible for uploading data to AWS S3.
type S3Uploader struct {
	client  *s3.Client
	bucket  string
	region  string
	timeout time.Duration
}

func NewS3Uploader(cfg *config.AWSS3Config) (*S3Uploader, error) {
	// load AWS config
	var opts []func(*awsconfig.LoadOptions) error
	opts = append(opts, awsconfig.WithRegion(cfg.Region))

	// if AccessKey && SecretKey provided, use it
	if cfg.AccessKey != "" && cfg.SecretKey != "" {
		opts = append(opts, awsconfig.WithCredentialsProvider(
			credentials.NewStaticCredentialsProvider(
				cfg.AccessKey,
				cfg.SecretKey,
				"",
			)),
		)
	}

	awsCfg, err := awsconfig.LoadDefaultConfig(context.Background(), opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to load default config: %w", err)
	}

	return &S3Uploader{
		client:  s3.NewFromConfig(awsCfg),
		bucket:  cfg.Bucket,
		region:  cfg.Region,
		timeout: 30 * time.Second,
	}, nil
}

// UploadData uploads data to s3 bucket
func (u *S3Uploader) UploadData(ctx context.Context, data []byte, objectKey string) error {
	uploadCtx, cancel := context.WithTimeout(ctx, u.timeout)
	defer cancel()

	_, err := u.client.PutObject(uploadCtx, &s3.PutObjectInput{
		Bucket:      aws.String(u.bucket),
		Key:         aws.String(objectKey),
		Body:        bytes.NewReader(data),
		ContentType: aws.String("application/octet-stream"),
	})
	return err
}
