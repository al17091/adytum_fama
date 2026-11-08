package apache

import (
	"strings"
	"testing"
	"time"
)

func TestApacheConnectorParseLogs(t *testing.T) {
	logContent := `198.51.100.22 - - [08/Nov/2026:10:00:15 +0000] "GET / HTTP/1.1" 200 4520 "-" "Mozilla/5.0"
203.0.113.50 - - [08/Nov/2026:10:05:00 +0000] "GET /.env HTTP/1.1" 403 289 "-" "curl/7.68.0"
198.51.100.99 - - [08/Nov/2026:10:10:30 +0000] "POST /wp-login.php HTTP/1.1" 403 512 "-" "python-requests/2.28.1"
192.0.2.10 - - [08/Nov/2026:08:00:00 +0000] "GET /old-probe HTTP/1.1" 403 200 "-" "nikto"
`

	conn := NewConnector(Config{
		GatewayNode: "apache-test-node",
	})

	since := time.Date(2026, 11, 8, 9, 0, 0, 0, time.UTC)
	events, err := conn.ParseLogs(strings.NewReader(logContent), since)
	if err != nil {
		t.Fatalf("unexpected error parsing logs: %v", err)
	}

	// 200 OK should be skipped, old-probe (08:00) should be filtered out by `since` (09:00).
	// Exactly 2 events should be returned.
	if len(events) != 2 {
		t.Fatalf("expected 2 block events, got %d", len(events))
	}

	// First event: .env probe
	e1 := events[0]
	if e1.SrcIP != "203.0.113.50" {
		t.Errorf("expected SrcIP 203.0.113.50, got %s", e1.SrcIP)
	}
	if e1.ThreatCategory != "DOTFILE" {
		t.Errorf("expected ThreatCategory DOTFILE, got %s", e1.ThreatCategory)
	}
	if e1.Layer != "APACHE_403" {
		t.Errorf("expected Layer APACHE_403, got %s", e1.Layer)
	}
	if e1.GatewayNode != "apache-test-node" {
		t.Errorf("expected GatewayNode apache-test-node, got %s", e1.GatewayNode)
	}

	// Second event: wp-login probe
	e2 := events[1]
	if e2.SrcIP != "198.51.100.99" {
		t.Errorf("expected SrcIP 198.51.100.99, got %s", e2.SrcIP)
	}
	if e2.ThreatCategory != "CMS_PROBE" {
		t.Errorf("expected ThreatCategory CMS_PROBE, got %s", e2.ThreatCategory)
	}
}
