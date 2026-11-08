package wazuh

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/al17091/adytum_fama/pkg/models"
)

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

// FetchRecentEvents retrieves HTTP 444 and ModSecurity block/detect events for the past windowMinutes
func (c *Client) FetchRecentEvents(ctx context.Context, windowMinutes int) ([]models.BlockEvent, error) {
	timeLimit := time.Now().UTC().Add(-time.Duration(windowMinutes) * time.Minute).Format(time.RFC3339)

	// OpenSearch Query DSL matching Nginx 444 (rule 100100) and ModSecurity CRS (100401/100402)
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
						"terms": map[string]interface{}{
							"rule.id": []string{"100100", "100401", "100402"},
						},
					},
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
		EventID:   hitID,
		IsBlocked: true,
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
	case "100100":
		evt.Layer = "HTTP_444"
		evt.ThreatCategory = "RAW_IP_RECON"
	case "100402":
		evt.Layer = "MODSEC_BLOCK"
		evt.ThreatCategory = "WAF_EXPLOIT_ATTEMPT"
	case "100401":
		evt.Layer = "MODSEC_DETECT"
		evt.IsBlocked = false
		evt.ThreatCategory = "SUSPICIOUS_PROBE"
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
	}
	if evt.SrcIP == "" {
		if srcip, ok := src["srcip"].(string); ok {
			evt.SrcIP = srcip
		}
	}

	// Drop if no IP identified
	if evt.SrcIP == "" {
		return evt, false
	}

	if ruleID != "" {
		evt.WAFRuleIDs = []string{ruleID}
	}

	return evt, true
}
