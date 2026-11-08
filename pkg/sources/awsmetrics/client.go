package awsmetrics

import (
	"context"
	"fmt"
	"time"

	"github.com/al17091/adytum_fama/pkg/models"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/cloudwatch"
	cwtypes "github.com/aws/aws-sdk-go-v2/service/cloudwatch/types"
	"github.com/aws/aws-sdk-go-v2/service/wafv2"
	waftypes "github.com/aws/aws-sdk-go-v2/service/wafv2/types"
)

// MetricsClient handles CloudWatch and AWS WAF telemetry retrieval in us-east-1
type MetricsClient struct {
	cwClient  *cloudwatch.Client
	wafClient *wafv2.Client
}

// CloudFrontMetrics contains aggregate metrics for the hour
type CloudFrontMetrics struct {
	Requests        int64
	BytesDownloaded int64
	WAFBlocked      int64
}

// NewMetricsClient creates a client targeting us-east-1 where CloudFront/Global WAF metrics reside
func NewMetricsClient(cfg aws.Config) *MetricsClient {
	// CloudFront and Global WAF metrics are published strictly to us-east-1
	eastCfg := cfg.Copy()
	eastCfg.Region = "us-east-1"

	return &MetricsClient{
		cwClient:  cloudwatch.NewFromConfig(eastCfg),
		wafClient: wafv2.NewFromConfig(eastCfg),
	}
}

// FetchHourlyCloudFrontMetrics queries CloudWatch for CloudFront distribution and WAF counts
func (c *MetricsClient) FetchHourlyCloudFrontMetrics(ctx context.Context, distID, webACLName string, start, end time.Time) (CloudFrontMetrics, error) {
	metrics := CloudFrontMetrics{}

	// Single GetMetricData call packing CloudFront requests, bytes, and WAF blocked count
	queries := []cwtypes.MetricDataQuery{
		{
			Id: aws.String("cf_reqs"),
			MetricStat: &cwtypes.MetricStat{
				Metric: &cwtypes.Metric{
					Namespace:  aws.String("AWS/CloudFront"),
					MetricName: aws.String("Requests"),
					Dimensions: []cwtypes.Dimension{
						{Name: aws.String("DistributionId"), Value: aws.String(distID)},
						{Name: aws.String("Region"), Value: aws.String("Global")},
					},
				},
				Period: aws.Int32(3600),
				Stat:   aws.String("Sum"),
			},
		},
		{
			Id: aws.String("cf_bytes"),
			MetricStat: &cwtypes.MetricStat{
				Metric: &cwtypes.Metric{
					Namespace:  aws.String("AWS/CloudFront"),
					MetricName: aws.String("BytesDownloaded"),
					Dimensions: []cwtypes.Dimension{
						{Name: aws.String("DistributionId"), Value: aws.String(distID)},
						{Name: aws.String("Region"), Value: aws.String("Global")},
					},
				},
				Period: aws.Int32(3600),
				Stat:   aws.String("Sum"),
			},
		},
		{
			Id: aws.String("waf_blocked"),
			MetricStat: &cwtypes.MetricStat{
				Metric: &cwtypes.Metric{
					Namespace:  aws.String("AWS/WAFV2"),
					MetricName: aws.String("BlockedRequests"),
					Dimensions: []cwtypes.Dimension{
						{Name: aws.String("WebACL"), Value: aws.String(webACLName)},
						{Name: aws.String("Region"), Value: aws.String("Global")},
						{Name: aws.String("Rule"), Value: aws.String("ALL")},
					},
				},
				Period: aws.Int32(3600),
				Stat:   aws.String("Sum"),
			},
		},
	}

	output, err := c.cwClient.GetMetricData(ctx, &cloudwatch.GetMetricDataInput{
		StartTime:         aws.Time(start),
		EndTime:           aws.Time(end),
		MetricDataQueries: queries,
	})
	if err != nil {
		return metrics, fmt.Errorf("failed fetching cloudwatch metrics: %w", err)
	}

	for _, result := range output.MetricDataResults {
		sum := int64(0)
		for _, val := range result.Values {
			sum += int64(val)
		}

		switch *result.Id {
		case "cf_reqs":
			metrics.Requests = sum
		case "cf_bytes":
			metrics.BytesDownloaded = sum
		case "waf_blocked":
			metrics.WAFBlocked = sum
		}
	}

	return metrics, nil
}

// FetchSampledBlockedRequests retrieves sample malicious requests blocked at the CloudFront edge
func (c *MetricsClient) FetchSampledBlockedRequests(ctx context.Context, webACLArn string, windowMinutes int) ([]models.BlockEvent, error) {
	end := time.Now().UTC()
	start := end.Add(-time.Duration(windowMinutes) * time.Minute)

	out, err := c.wafClient.GetSampledRequests(ctx, &wafv2.GetSampledRequestsInput{
		WebAclArn:      aws.String(webACLArn),
		RuleMetricName: aws.String("ALL"),
		Scope:          waftypes.ScopeCloudfront,
		TimeWindow: &waftypes.TimeWindow{
			StartTime: aws.Time(start),
			EndTime:   aws.Time(end),
		},
		MaxItems: aws.Int64(500),
	})
	if err != nil {
		return nil, fmt.Errorf("failed fetching sampled WAF requests: %w", err)
	}

	var events []models.BlockEvent
	for _, sample := range out.SampledRequests {
		if sample.Action != nil && *sample.Action != "BLOCK" {
			continue // Only capture blocked requests
		}

		e := models.BlockEvent{
			EventID:        fmt.Sprintf("waf-%d", sample.Timestamp.UnixNano()),
			ObservedAt:     *sample.Timestamp,
			Layer:          "AWS_WAF_SAMPLE",
			IsBlocked:      true,
			GatewayNode:    "edge-cloudfront",
			ThreatCategory: "EDGE_WAF_BLOCK",
		}

		if sample.Request != nil {
			if sample.Request.ClientIP != nil {
				e.SrcIP = *sample.Request.ClientIP
			}
			if sample.Request.Country != nil {
				e.CountryCode = *sample.Request.Country
			}
			if sample.Request.Method != nil {
				e.HTTPMethod = *sample.Request.Method
			}
			if sample.Request.URI != nil {
				e.RequestURI = *sample.Request.URI
			}

			// Extract User-Agent header
			for _, h := range sample.Request.Headers {
				if h.Name != nil && h.Value != nil && *h.Name == "user-agent" {
					e.UserAgent = *h.Value
					break
				}
			}
		}

		if sample.RuleNameWithinRuleGroup != nil {
			e.WAFRuleIDs = []string{*sample.RuleNameWithinRuleGroup}
		}

		if e.SrcIP != "" {
			events = append(events, e)
		}
	}

	return events, nil
}
