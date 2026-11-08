package datalake

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"github.com/al17091/adytum_fama/pkg/models"
	"github.com/parquet-go/parquet-go"
)

// WriteBlockEventsParquet encodes slice of BlockEvents into a Parquet byte stream with Zstd compression
func WriteBlockEventsParquet(w io.Writer, events []models.BlockEvent) error {
	schema := parquet.SchemaOf(new(models.BlockEvent))

	writer := parquet.NewGenericWriter[models.BlockEvent](
		w,
		schema,
		parquet.Compression(&parquet.Zstd),
	)

	if _, err := writer.Write(events); err != nil {
		_ = writer.Close()
		return fmt.Errorf("failed to write parquet rows: %w", err)
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("failed to close parquet writer: %w", err)
	}

	return nil
}

// EncodeBlockEventsToBytes returns compressed Parquet bytes for uploading
func EncodeBlockEventsToBytes(events []models.BlockEvent) ([]byte, error) {
	var buf bytes.Buffer
	if err := WriteBlockEventsParquet(&buf, events); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// SaveBlockEventsParquet writes Parquet rows directly to a local file
func SaveBlockEventsParquet(filePath string, events []models.BlockEvent) error {
	f, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("failed to create file (%s): %w", filePath, err)
	}
	defer f.Close()

	return WriteBlockEventsParquet(f, events)
}
