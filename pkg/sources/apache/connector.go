package apache

import (
	"bufio"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/al17091/adytum_fama/pkg/models"
)

// Combined log pattern: 127.0.0.1 - - [10/Oct/2000:13:55:36 -0700] "GET /apache_pb.gif HTTP/1.0" 200 2326 "http://www.example.com/start.html" "Mozilla/4.08 [en] (Win98; I ;Nav)"
var combinedLogRegex = regexp.MustCompile(`^(\S+)\s+\S+\s+\S+\s+\[([^\]]+)\]\s+"([^"]*)"\s+(\d{3})\s+(\S+)\s+"([^"]*)"\s+"([^"]*)"`)

// Config defines settings for Apache HTTP Server log ingestion
type Config struct {
	LogPath     string
	GatewayNode string
}

// Connector ingests Apache access and security logs into AdytumFama
type Connector struct {
	logPath     string
	gatewayNode string
}

// NewConnector creates a new Apache log event connector
func NewConnector(cfg Config) *Connector {
	node := cfg.GatewayNode
	if node == "" {
		node = "apache-origin"
	}
	return &Connector{
		logPath:     cfg.LogPath,
		gatewayNode: node,
	}
}

// Name returns connector identifier
func (c *Connector) Name() string {
	return "apache"
}

// FetchEvents implements sources.EventConnector
func (c *Connector) FetchEvents(ctx context.Context, window time.Duration) ([]models.BlockEvent, error) {
	if c.logPath == "" {
		return nil, nil
	}

	file, err := os.Open(c.logPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("failed opening apache log file: %w", err)
	}
	defer file.Close()

	since := time.Now().UTC().Add(-window)
	return c.ParseLogs(file, since)
}

// ParseLogs scans log lines from an io.Reader and returns BlockEvents observed since the given time
func (c *Connector) ParseLogs(r io.Reader, since time.Time) ([]models.BlockEvent, error) {
	var events []models.BlockEvent
	scanner := bufio.NewScanner(r)

	// Support lines up to 64KB (e.g. large URLs / headers)
	buf := make([]byte, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if len(line) == 0 || line[0] == '#' {
			continue
		}

		evt, ok := c.parseLine(line, since)
		if ok {
			events = append(events, evt)
		}
	}

	if err := scanner.Err(); err != nil {
		return events, fmt.Errorf("error reading apache log stream: %w", err)
	}

	return events, nil
}

func (c *Connector) parseLine(line string, since time.Time) (models.BlockEvent, bool) {
	matches := combinedLogRegex.FindStringSubmatch(line)
	if len(matches) < 8 {
		return models.BlockEvent{}, false
	}

	srcIP := matches[1]
	timeStr := matches[2]
	requestLine := matches[3]
	statusStr := matches[4]
	// bytesStr := matches[5]
	// referer := matches[6]
	userAgent := matches[7]

	status, err := strconv.Atoi(statusStr)
	if err != nil {
		return models.BlockEvent{}, false
	}

	// Only capture mitigation events: 403 Forbidden, 400 Bad Request, 444 Drop
	if status != http.StatusForbidden && status != http.StatusBadRequest && status != 444 {
		return models.BlockEvent{}, false
	}

	// Parse Apache timestamp: "02/Jan/2006:15:04:05 -0700"
	t, err := time.Parse("02/Jan/2006:15:04:05 -0700", timeStr)
	if err != nil {
		t = time.Now().UTC()
	} else {
		t = t.UTC()
	}

	// Filter by window
	if !since.IsZero() && t.Before(since) {
		return models.BlockEvent{}, false
	}

	// Parse request line: "GET /admin/.env HTTP/1.1"
	reqParts := strings.Split(requestLine, " ")
	method := "GET"
	uri := ""
	proto := "HTTP/1.1"
	if len(reqParts) >= 1 {
		method = reqParts[0]
	}
	if len(reqParts) >= 2 {
		uri = reqParts[1]
	}
	if len(reqParts) >= 3 {
		proto = reqParts[2]
	}

	threatCategory := classifyThreat(uri, userAgent, status)

	hash := sha256.Sum256([]byte(line))
	eventID := fmt.Sprintf("apache-%s", hex.EncodeToString(hash[:16]))

	layer := "APACHE_403"
	if status == 444 {
		layer = "HTTP_444"
	}

	evt := models.BlockEvent{
		EventID:        eventID,
		ObservedAt:     t,
		Layer:          layer,
		IsBlocked:      true,
		GatewayNode:    c.gatewayNode,
		SrcIP:          srcIP,
		HTTPMethod:     method,
		HTTPProtocol:   proto,
		RequestURI:     uri,
		UserAgent:      userAgent,
		ThreatCategory: threatCategory,
	}

	return evt, true
}

func classifyThreat(uri, userAgent string, status int) string {
	lowerURI := strings.ToLower(uri)
	lowerUA := strings.ToLower(userAgent)

	if strings.Contains(lowerURI, ".env") || strings.Contains(lowerURI, ".git") || strings.Contains(lowerURI, ".aws") {
		return "DOTFILE"
	}
	if strings.Contains(lowerURI, "wp-login") || strings.Contains(lowerURI, "xmlrpc.php") || strings.Contains(lowerURI, "/phpmyadmin") {
		return "CMS_PROBE"
	}
	if strings.Contains(lowerURI, "union") || strings.Contains(lowerURI, "select") || strings.Contains(lowerURI, "sleep(") {
		return "SQLI"
	}
	if strings.Contains(lowerURI, "<script") || strings.Contains(lowerURI, "javascript:") {
		return "XSS"
	}
	if strings.Contains(lowerUA, "sqlmap") || strings.Contains(lowerUA, "nikto") || strings.Contains(lowerUA, "nuclei") {
		return "SECURITY_SCANNER"
	}
	if status == 444 {
		return "RAW_IP_RECON"
	}
	return "BLOCKED_REQUEST"
}
