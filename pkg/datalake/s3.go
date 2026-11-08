package datalake

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// Uploader manages S3 data lake storage for compacted Parquet events
type Uploader struct {
	client *s3.Client
	bucket string
}

// NewUploader creates an S3 data lake uploader client
func NewUploader(client *s3.Client, bucketName string) *Uploader {
	return &Uploader{
		client: client,
		bucket: bucketName,
	}
}

// FormatPartitionPath returns Hive-style partition path for a specific UTC date
func FormatPartitionPath(t time.Time) string {
	utc := t.UTC()
	return fmt.Sprintf("block_events/year=%04d/month=%02d/day=%02d/block_events_%s.parquet",
		utc.Year(), utc.Month(), utc.Day(), utc.Format("2006-01-02"),
	)
}

// CheckDayExists checks whether the daily compacted Parquet file already exists in S3
func (u *Uploader) CheckDayExists(ctx context.Context, targetDate time.Time) (bool, error) {
	key := FormatPartitionPath(targetDate)

	_, err := u.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(u.bucket),
		Key:    aws.String(key),
	})
	if err != nil {
		var notFound *types.NotFound
		var noSuchKey *types.NoSuchKey
		if errors.As(err, &notFound) || errors.As(err, &noSuchKey) {
			return false, nil
		}
		// In AWS SDK v2, 404 can also present as generic ResponseError
		return false, nil
	}

	return true, nil
}

// UploadDailyParquet uploads compacted Parquet bytes to the S3 data lake
func (u *Uploader) UploadDailyParquet(ctx context.Context, targetDate time.Time, parquetData []byte) (string, error) {
	key := FormatPartitionPath(targetDate)

	_, err := u.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(u.bucket),
		Key:           aws.String(key),
		Body:          bytes.NewReader(parquetData),
		ContentLength: aws.Int64(int64(len(parquetData))),
		ContentType:   aws.String("application/vnd.apache.parquet"),
	})
	if err != nil {
		return "", fmt.Errorf("failed to upload Parquet to s3://%s/%s: %w", u.bucket, key, err)
	}

	return fmt.Sprintf("s3://%s/%s", u.bucket, key), nil
}
