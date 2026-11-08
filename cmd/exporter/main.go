package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/al17091/adytum_fama/pkg/awsauth"
	"github.com/al17091/adytum_fama/pkg/datalake"
	"github.com/al17091/adytum_fama/pkg/storage"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	log.Println("[INFO] Starting AdytumFama daily S3 Parquet exporter and backfill job...")

	// 1. Connect to PostgreSQL
	pgDSN := getEnv("POSTGRES_DSN", "postgres://adytum:adytum@localhost:5432/adytum_fama?sslmode=disable")
	db, err := storage.NewDB(ctx, pgDSN)
	if err != nil {
		log.Fatalf("[FATAL] Database connection failed: %v", err)
	}
	defer db.Close()

	// 2. Initialize AWS S3 Uploader
	bucketName := getEnv("S3_DATALAKE_BUCKET", "")
	if bucketName == "" {
		log.Fatalf("[FATAL] S3_DATALAKE_BUCKET environment variable is required")
	}

	awsCfg, err := awsauth.LoadAWSConfig(ctx, awsauth.ConfigOptions{})
	if err != nil {
		log.Fatalf("[FATAL] Failed loading AWS config: %v", err)
	}

	s3Client := s3.NewFromConfig(awsCfg)
	uploader := datalake.NewUploader(s3Client, bucketName)

	// 3. Scan the past 7 days (from yesterday back to 7 days ago)
	now := time.Now().UTC()
	yesterday := now.AddDate(0, 0, -1)

	for i := 0; i < 7; i++ {
		targetDate := yesterday.AddDate(0, 0, -i)
		dateStr := targetDate.Format("2006-01-02")

		exists, err := uploader.CheckDayExists(ctx, targetDate)
		if err != nil {
			log.Printf("[WARN] Failed checking S3 existence for %s: %v", dateStr, err)
			continue
		}

		if exists {
			log.Printf("[INFO] Partition for %s already exists in S3. Skipping.", dateStr)
			continue
		}

		log.Printf("[INFO] Partition for %s is missing from S3. Generating Parquet from PostgreSQL...", dateStr)
		events, err := db.GetEventsForDate(ctx, targetDate)
		if err != nil {
			log.Printf("[ERROR] Failed fetching events for %s: %v", dateStr, err)
			continue
		}

		if len(events) == 0 {
			log.Printf("[INFO] Zero events recorded on %s. Writing empty marker partition.", dateStr)
		}

		parquetBytes, err := datalake.EncodeBlockEventsToBytes(events)
		if err != nil {
			log.Printf("[ERROR] Failed encoding Parquet for %s: %v", dateStr, err)
			continue
		}

		s3URI, err := uploader.UploadDailyParquet(ctx, targetDate, parquetBytes)
		if err != nil {
			log.Printf("[ERROR] Failed uploading Parquet to S3 for %s: %v", dateStr, err)
			continue
		}

		log.Printf("[SUCCESS] Successfully archived %d events for %s to %s (Size: %d bytes)",
			len(events), dateStr, s3URI, len(parquetBytes),
		)
	}

	log.Println("[INFO] AdytumFama S3 Parquet export job completed.")
}
