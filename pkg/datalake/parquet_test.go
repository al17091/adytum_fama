package datalake

import (
	"bytes"
	"testing"
	"time"

	"github.com/al17091/adytum_fama/pkg/models"
	"github.com/parquet-go/parquet-go"
)

func TestWriteBlockEventsParquet(t *testing.T) {
	now := time.Now().Truncate(time.Millisecond)
	sampleEvents := []models.BlockEvent{
		{
			EventID:         "evt-001",
			ObservedAt:      now,
			Layer:           "HTTP_444",
			IsBlocked:       true,
			SrcIP:           "198.51.100.1",
			CountryCode:     "US",
			CountryName:     "United States",
			CityName:        "San Jose",
			Latitude:        37.3382,
			Longitude:       -121.8863,
			ASN:             15169,
			ASOrg:           "Google LLC",
			ReverseDNS:      "crawl-198-51-100-1.googlebot.com",
			ActorType:       "SEARCH_ENGINE",
			ActorName:       "Googlebot",
			HTTPMethod:      "GET",
			HTTPProtocol:    "HTTP/2.0",
			TLSProtocol:     "TLSv1.3",
			RequestHost:     "origin.engineerlab.jp",
			RequestURI:      "/robots.txt",
			UserAgent:       "Googlebot/2.1",
			ThreatCategory:  "RAW_IP_RECON",
			WAFRuleIDs:      []string{"100100"},
			WAFAnomalyScore: 0,
		},
		{
			EventID:         "evt-002",
			ObservedAt:      now.Add(10 * time.Minute),
			Layer:           "MODSEC_BLOCK",
			IsBlocked:       true,
			SrcIP:           "203.0.113.5",
			CountryCode:     "JP",
			CountryName:     "Japan",
			CityName:        "Tokyo",
			Latitude:        35.6895,
			Longitude:       139.6917,
			ASN:             2514,
			ASOrg:           "NTT Communications",
			ReverseDNS:      "",
			ActorType:       "BOT_SCANNER",
			ActorName:       "Automated Security Probe",
			HTTPMethod:      "POST",
			HTTPProtocol:    "HTTP/1.1",
			TLSProtocol:     "TLSv1.2",
			RequestHost:     "100.64.0.12",
			RequestURI:      "/api/v1/login?admin=1%27%20OR%201=1--",
			UserAgent:       "sqlmap/1.7.2#stable",
			ThreatCategory:  "SQLI",
			WAFRuleIDs:      []string{"942100", "949110"},
			WAFAnomalyScore: 10,
		},
	}

	parquetBytes, err := EncodeBlockEventsToBytes(sampleEvents)
	if err != nil {
		t.Fatalf("EncodeBlockEventsToBytes failed: %v", err)
	}

	if len(parquetBytes) == 0 {
		t.Fatal("generated parquet bytes is empty")
	}

	// Read back and verify using parquet-go reader
	reader := parquet.NewGenericReader[models.BlockEvent](bytes.NewReader(parquetBytes))
	defer reader.Close()

	readEvents := make([]models.BlockEvent, len(sampleEvents))
	n, err := reader.Read(readEvents)
	if err != nil && err.Error() != "EOF" {
		t.Fatalf("failed to read back parquet: %v", err)
	}

	if n != 2 {
		t.Fatalf("expected 2 read events, got %d", n)
	}

	if readEvents[0].EventID != "evt-001" || readEvents[1].EventID != "evt-002" {
		t.Errorf("read events mismatch: %+v", readEvents)
	}
}
