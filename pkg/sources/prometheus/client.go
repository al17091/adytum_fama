package prometheus

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Config holds Prometheus endpoint configuration
type Config struct {
	Endpoint string // e.g. "http://prometheus-operated.monitoring.svc:9090"
}

// Client manages queries to Prometheus
type Client struct {
	endpoint string
	client   *http.Client
}

// GatewayRequestTotals holds hourly increase totals for each gateway
type GatewayRequestTotals struct {
	ForamenTotal int64
	RetisTotal   int64
}

// NewClient creates a Prometheus client
func NewClient(cfg Config) *Client {
	return &Client{
		endpoint: strings.TrimRight(cfg.Endpoint, "/"),
		client:   &http.Client{Timeout: 15 * time.Second},
	}
}

// FetchHourlyGatewayTotals retrieves total requests processed by each gateway during the last hour
func (c *Client) FetchHourlyGatewayTotals(ctx context.Context) (GatewayRequestTotals, error) {
	totals := GatewayRequestTotals{}

	promQL := `increase(nginx_http_requests_total{job="nginx-gateway-exporter"}[1h])`
	queryURL := fmt.Sprintf("%s/api/v1/query?query=%s", c.endpoint, url.QueryEscape(promQL))

	req, err := http.NewRequestWithContext(ctx, "GET", queryURL, nil)
	if err != nil {
		return totals, fmt.Errorf("failed creating prometheus request: %w", err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return totals, fmt.Errorf("prometheus request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return totals, fmt.Errorf("prometheus returned status %d: %s", resp.StatusCode, string(body))
	}

	var promResp struct {
		Status string `json:"status"`
		Data   struct {
			ResultType string `json:"resultType"`
			Result     []struct {
				Metric map[string]string `json:"metric"`
				Value  []interface{}     `json:"value"` // [timestamp, "value_string"]
			} `json:"result"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&promResp); err != nil {
		return totals, fmt.Errorf("failed decoding prometheus response: %w", err)
	}

	for _, res := range promResp.Data.Result {
		gateway := res.Metric["gateway"]
		if len(res.Value) < 2 {
			continue
		}

		valStr, ok := res.Value[1].(string)
		if !ok {
			continue
		}

		valFloat, err := strconv.ParseFloat(valStr, 64)
		if err != nil {
			continue
		}
		valInt := int64(valFloat)

		switch gateway {
		case "foramen":
			totals.ForamenTotal += valInt
		case "retis":
			totals.RetisTotal += valInt
		}
	}

	return totals, nil
}
