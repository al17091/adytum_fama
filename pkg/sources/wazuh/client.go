package wazuh

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/al17091/adytum_fama/pkg/models"
)

var (
	// CGNAT (RFC 6598) 100.64.0.0/10 (e.g. Tailscale internal IPs)
	cgnatNet = &net.IPNet{
		IP:   net.IPv4(100, 64, 0, 0),
		Mask: net.CIDRMask(10, 32),
	}
	// Additional reserved / documentation CIDRs
	reservedNets = []*net.IPNet{
		{IP: net.IPv4(192, 0, 2, 0), Mask: net.CIDRMask(24, 32)},    // TEST-NET-1
		{IP: net.IPv4(198, 51, 100, 0), Mask: net.CIDRMask(24, 32)}, // TEST-NET-2
		{IP: net.IPv4(203, 0, 113, 0), Mask: net.CIDRMask(24, 32)},  // TEST-NET-3
		{IP: net.IPv4(198, 18, 0, 0), Mask: net.CIDRMask(15, 32)},    // Benchmarking
		{IP: net.IPv4(240, 0, 0, 0), Mask: net.CIDRMask(4, 32)},     // Class E reserved
	}

	// Regex matching Nginx combined access log: "METHOD /uri HTTP/x" STATUS BYTES "REFERER" "USER-AGENT"
	nginxCombinedRegex = regexp.MustCompile(`"([A-Z]+)\s+([^"\s]+)\s+HTTP/[^"]+"\s+\d+\s+\d+\s+"[^"]*"\s+"([^"]*)"`)
	// Regex matching ModSecurity request line: request: "METHOD /uri HTTP/x"
	modsecRequestRegex = regexp.MustCompile(`request:\s+"([A-Z]+)\s+([^"\s]+)`)
	// Regex matching ModSecurity URI tag: [uri "/path"]
	modsecURITagRegex = regexp.MustCompile(`\[uri\s+"([^"]+)"\]`)
	// Regex matching CloudFront JA3/JA4 fingerprints in enriched Nginx logs
	ja3Regex = regexp.MustCompile(`\bja3="([^"\s]+)"`)
	ja4Regex = regexp.MustCompile(`\bja4="([^"\s]+)"`)
)

// IsReservedOrPrivateIP checks if an IP is private, link-local, loopback, or in reserved CIDRs
func IsReservedOrPrivateIP(ipStr string) bool {
	ip := net.ParseIP(strings.TrimSpace(ipStr))
	if ip == nil {
		return true // invalid or empty IP is treated as non-public
	}

	if ip.IsPrivate() || ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsMulticast() || ip.IsUnspecified() {
		return true
	}

	if cgnatNet.Contains(ip) {
		return true
	}

	for _, rn := range reservedNets {
		if rn.Contains(ip) {
			return true
		}
	}

	return false
}

// Config holds connection settings for Wazuh Indexer (OpenSearch)
type Config struct {
	Endpoint string // e.g. "https://wazuh-indexer.wazuh.svc:9200"
	Username string
	Password string
}

// Client manages queries to Wazuh Indexer
type Client struct {
	endpoint string
	username string
	password string
	client   *http.Client
}

// NewClient creates a Wazuh Indexer client with TLS configuration
func NewClient(cfg Config) *Client {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true, // Internal Kubernetes step-ca TLS
		},
	}

	return &Client{
		endpoint: strings.TrimRight(cfg.Endpoint, "/"),
		username: cfg.Username,
		password: cfg.Password,
		client:   &http.Client{Transport: tr, Timeout: 30 * time.Second},
	}
}

// Name returns the identifier of this connector
func (c *Client) Name() string {
	return "wazuh"
}

// FetchEvents implements sources.EventConnector
func (c *Client) FetchEvents(ctx context.Context, window time.Duration) ([]models.BlockEvent, error) {
	minutes := int(window.Minutes())
	if minutes <= 0 {
		minutes = 60
	}
	return c.FetchRecentEvents(ctx, minutes)
}

// FetchRecentEvents retrieves perimeter block/detect events for the past windowMinutes,
// filtering out internal/private IP traffic and irrelevant 503s (31123/31163).
func (c *Client) FetchRecentEvents(ctx context.Context, windowMinutes int) ([]models.BlockEvent, error) {
	timeLimit := time.Now().UTC().Add(-time.Duration(windowMinutes) * time.Minute).Format(time.RFC3339)

	// OpenSearch Query DSL matching:
	// - 100100 (Nginx 444 Drop)
	// - 100401 (ModSecurity Detect)
	// - 100402 (ModSecurity Block)
	// - 31333  (ModSecurity rejected query)
	// - 31516  (Suspicious URL access)
	// - 31151  (Multiple 400 error codes from same IP)
	// - 31101 + data.id=403 (HTTP 403 access denial)
	queryDSL := map[string]interface{}{
		"size": 5000,
		"query": map[string]interface{}{
			"bool": map[string]interface{}{
				"filter": []map[string]interface{}{
					{
						"range": map[string]interface{}{
							"timestamp": map[string]interface{}{
								"gte": timeLimit,
							},
						},
					},
					{
						"bool": map[string]interface{}{
							"should": []map[string]interface{}{
								{
									"terms": map[string]interface{}{
										"rule.id": []string{
											"100100", "100401", "100402",
											"31333", "31516", "31151",
										},
									},
								},
								{
									"bool": map[string]interface{}{
										"filter": []map[string]interface{}{
											{"term": map[string]interface{}{"rule.id": "31101"}},
											{"term": map[string]interface{}{"data.id": "403"}},
										},
									},
								},
							},
							"minimum_should_match": 1,
						},
					},
				},
				"must_not": []map[string]interface{}{
					{
						"terms": map[string]interface{}{
							"rule.id": []string{"31123", "31163"},
						},
					},
					{"prefix": map[string]interface{}{"data.srcip": "10."}},
					{"prefix": map[string]interface{}{"data.srcip": "100."}},
					{"prefix": map[string]interface{}{"data.srcip": "127."}},
					{"prefix": map[string]interface{}{"data.srcip": "172.16."}},
					{"prefix": map[string]interface{}{"data.srcip": "172.17."}},
					{"prefix": map[string]interface{}{"data.srcip": "172.18."}},
					{"prefix": map[string]interface{}{"data.srcip": "172.19."}},
					{"prefix": map[string]interface{}{"data.srcip": "172.20."}},
					{"prefix": map[string]interface{}{"data.srcip": "172.21."}},
					{"prefix": map[string]interface{}{"data.srcip": "172.22."}},
					{"prefix": map[string]interface{}{"data.srcip": "172.23."}},
					{"prefix": map[string]interface{}{"data.srcip": "172.24."}},
					{"prefix": map[string]interface{}{"data.srcip": "172.25."}},
					{"prefix": map[string]interface{}{"data.srcip": "172.26."}},
					{"prefix": map[string]interface{}{"data.srcip": "172.27."}},
					{"prefix": map[string]interface{}{"data.srcip": "172.28."}},
					{"prefix": map[string]interface{}{"data.srcip": "172.29."}},
					{"prefix": map[string]interface{}{"data.srcip": "172.30."}},
					{"prefix": map[string]interface{}{"data.srcip": "172.31."}},
					{"prefix": map[string]interface{}{"data.srcip": "192.168."}},
				},
			},
		},
		"sort": []map[string]interface{}{
			{"timestamp": map[string]string{"order": "asc"}},
		},
	}

	bodyBytes, err := json.Marshal(queryDSL)
	if err != nil {
		return nil, fmt.Errorf("failed marshaling search query: %w", err)
	}

	searchURL := fmt.Sprintf("%s/wazuh-alerts-*/_search", c.endpoint)
	req, err := http.NewRequestWithContext(ctx, "POST", searchURL, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed creating search request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	if c.username != "" {
		req.SetBasicAuth(c.username, c.password)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("search request failed against %s: %w", searchURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("wazuh search returned non-200 status (%d): %s", resp.StatusCode, string(respBody))
	}

	var searchResp struct {
		Hits struct {
			Hits []struct {
				ID     string                 `json:"_id"`
				Source map[string]interface{} `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&searchResp); err != nil {
		return nil, fmt.Errorf("failed decoding search response: %w", err)
	}

	var events []models.BlockEvent
	for _, hit := range searchResp.Hits.Hits {
		evt, ok := parseWazuhHit(hit.ID, hit.Source)
		if ok {
			events = append(events, evt)
		}
	}

	return events, nil
}

func parseWazuhHit(hitID string, src map[string]interface{}) (models.BlockEvent, bool) {
	evt := models.BlockEvent{
		EventID:        hitID,
		IsBlocked:      true,
		Classification: "-",
	}

	// 1. Timestamp
	if tsStr, ok := src["timestamp"].(string); ok {
		if t, err := time.Parse(time.RFC3339, tsStr); err == nil {
			evt.ObservedAt = t
		}
	}
	if evt.ObservedAt.IsZero() {
		evt.ObservedAt = time.Now().UTC()
	}

	// 2. Rule and Layer mapping
	ruleID := ""
	if rule, ok := src["rule"].(map[string]interface{}); ok {
		if id, ok := rule["id"].(string); ok {
			ruleID = id
		}
	}

	switch ruleID {
	case "31333":
		evt.Layer = "MODSEC_BLOCK"
		evt.ThreatCategory = "WAF_EXPLOIT_ATTEMPT"
		evt.Classification = "MALICIOUS"
		evt.IsBlocked = true
	case "100402":
		evt.Layer = "MODSEC_BLOCK"
		evt.ThreatCategory = "WAF_EXPLOIT_ATTEMPT"
		evt.Classification = "MALICIOUS"
		evt.IsBlocked = true
	case "31516":
		evt.Layer = "SUSPICIOUS_URL"
		evt.ThreatCategory = "PATH_TRAVERSAL_RECON"
		evt.Classification = "MALICIOUS"
		evt.IsBlocked = true
	case "31151":
		evt.Layer = "ANOMALY_SPIKE"
		evt.ThreatCategory = "REPEATED_PROBE"
		evt.Classification = "SUSPICIOUS"
		evt.IsBlocked = true
	case "100100":
		evt.Layer = "HTTP_444"
		evt.ThreatCategory = "RAW_IP_RECON"
		evt.Classification = "-"
		evt.IsBlocked = true
	case "100401":
		evt.Layer = "MODSEC_DETECT"
		evt.ThreatCategory = "SUSPICIOUS_PROBE"
		evt.Classification = "-"
		evt.IsBlocked = false
	case "31101":
		evt.Layer = "HTTP_403"
		evt.ThreatCategory = "ACCESS_DENIED"
		evt.Classification = "-"
		evt.IsBlocked = true
	default:
		return evt, false
	}

	// 3. Agent / Gateway node
	if agent, ok := src["agent"].(map[string]interface{}); ok {
		if name, ok := agent["name"].(string); ok {
			evt.GatewayNode = name
		}
	}

	// 4. Source IP
	if data, ok := src["data"].(map[string]interface{}); ok {
		if srcip, ok := data["srcip"].(string); ok && srcip != "" {
			evt.SrcIP = srcip
		}
		if url, ok := data["url"].(string); ok {
			evt.RequestURI = url
		}
		if proto, ok := data["protocol"].(string); ok {
			evt.HTTPMethod = proto
		}
		if ua, ok := data["user_agent"].(string); ok && ua != "" {
			evt.UserAgent = ua
		} else if ua, ok := data["http_user_agent"].(string); ok && ua != "" {
			evt.UserAgent = ua
		}
		if ja3, ok := data["ja3"].(string); ok && ja3 != "" && ja3 != "-" {
			evt.JA3Fingerprint = ja3
		}
		if ja4, ok := data["ja4"].(string); ok && ja4 != "" && ja4 != "-" {
			evt.JA4Fingerprint = ja4
		}
	}
	if evt.SrcIP == "" {
		if srcip, ok := src["srcip"].(string); ok {
			evt.SrcIP = srcip
		}
	}

	// 5. User-Agent and RequestURI enrichment from full_log if missing
	fullLog, _ := src["full_log"].(string)
	if fullLog != "" {
		if submatch := nginxCombinedRegex.FindStringSubmatch(fullLog); len(submatch) >= 4 {
			if evt.HTTPMethod == "" {
				evt.HTTPMethod = submatch[1]
			}
			if evt.RequestURI == "" {
				evt.RequestURI = submatch[2]
			}
			if evt.UserAgent == "" {
				evt.UserAgent = submatch[3]
			}
		}
		if evt.RequestURI == "" {
			if submatch := modsecRequestRegex.FindStringSubmatch(fullLog); len(submatch) >= 3 {
				if evt.HTTPMethod == "" {
					evt.HTTPMethod = submatch[1]
				}
				evt.RequestURI = submatch[2]
			} else if submatch := modsecURITagRegex.FindStringSubmatch(fullLog); len(submatch) >= 2 {
				evt.RequestURI = submatch[1]
			}
		}

		if evt.JA3Fingerprint == "" {
			if submatch := ja3Regex.FindStringSubmatch(fullLog); len(submatch) >= 2 && submatch[1] != "-" {
				evt.JA3Fingerprint = submatch[1]
			}
		}
		if evt.JA4Fingerprint == "" {
			if submatch := ja4Regex.FindStringSubmatch(fullLog); len(submatch) >= 2 && submatch[1] != "-" {
				evt.JA4Fingerprint = submatch[1]
			}
		}
	}

	// Drop if no IP identified, or if IP is private, link-local, or reserved
	if evt.SrcIP == "" || IsReservedOrPrivateIP(evt.SrcIP) {
		return evt, false
	}

	if ruleID != "" {
		evt.WAFRuleIDs = []string{ruleID}
	}

	return evt, true
}
