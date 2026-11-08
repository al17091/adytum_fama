package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/al17091/adytum_fama/pkg/awsauth"
	"github.com/al17091/adytum_fama/pkg/dedupe"
	"github.com/al17091/adytum_fama/pkg/geoip"
	"github.com/al17091/adytum_fama/pkg/models"
	"github.com/al17091/adytum_fama/pkg/rdns"
	"github.com/al17091/adytum_fama/pkg/sources"
	"github.com/al17091/adytum_fama/pkg/sources/apache"
	"github.com/al17091/adytum_fama/pkg/sources/awsmetrics"
	"github.com/al17091/adytum_fama/pkg/sources/prometheus"
	"github.com/al17091/adytum_fama/pkg/sources/wazuh"
	"github.com/al17091/adytum_fama/pkg/storage"
)

func getEnv(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	log.Println("[INFO] Starting AdytumFama hourly telemetry collector...")

	// 1. Connect to PostgreSQL
	pgDSN := getEnv("POSTGRES_DSN", "postgres://adytum:adytum@localhost:5432/adytum_fama?sslmode=disable")
	db, err := storage.NewDB(ctx, pgDSN)
	if err != nil {
		log.Fatalf("[FATAL] Database connection failed: %v", err)
	}
	defer db.Close()

	if err := db.InitSchema(ctx); err != nil {
		log.Fatalf("[FATAL] Schema initialization failed: %v", err)
	}

	// 2. Initialize GeoIP & rDNS
	cityPath := getEnv("GEOIP_CITY_PATH", "/usr/share/GeoIP/GeoLite2-City.mmdb")
	asnPath := getEnv("GEOIP_ASN_PATH", "/usr/share/GeoIP/GeoLite2-ASN.mmdb")
	geoResolver, err := geoip.NewResolver(cityPath, asnPath)
	if err != nil {
		log.Printf("[WARN] GeoIP resolver partial init (%v). Continuing without full GeoIP.", err)
	} else {
		defer geoResolver.Close()
	}

	rdnsTagger := rdns.NewTagger()

	// 3. Register Event Connectors (Wazuh, AWS WAF, Apache, etc.)
	var eventConnectors []sources.EventConnector

	// Wazuh Connector
	wazuhEndpoint := getEnv("WAZUH_ENDPOINT", "https://wazuh-indexer.wazuh.svc:9200")
	wazuhUser := getEnv("WAZUH_USERNAME", "adytum_fama_reader")
	wazuhPass := getEnv("WAZUH_PASSWORD", "")
	wazuhClient := wazuh.NewClient(wazuh.Config{
		Endpoint: wazuhEndpoint,
		Username: wazuhUser,
		Password: wazuhPass,
	})
	eventConnectors = append(eventConnectors, wazuhClient)

	// AWS WAF Connector (if AWS auth & WAF WebACL configured)
	awsCfg, err := awsauth.LoadAWSConfig(ctx, awsauth.ConfigOptions{})
	var awsMetricsClient *awsmetrics.MetricsClient
	if err == nil {
		awsMetricsClient = awsmetrics.NewMetricsClient(awsCfg)
		webACLArn := getEnv("WAF_WEBACL_ARN", "")
		if webACLArn != "" {
			eventConnectors = append(eventConnectors, awsmetrics.NewWAFEventConnector(awsMetricsClient, webACLArn))
		}
	} else {
		log.Printf("[WARN] AWS SDK init skipped or failed: %v", err)
	}

	// Optional Apache Connector (if log path configured)
	apacheLogPath := getEnv("APACHE_LOG_PATH", "")
	if apacheLogPath != "" {
		eventConnectors = append(eventConnectors, apache.NewConnector(apache.Config{
			LogPath:     apacheLogPath,
			GatewayNode: getEnv("APACHE_GATEWAY_NODE", "apache-gateway"),
		}))
	}

	// 4. Ingest events uniformly across all registered connectors
	collectionWindow := 70 * time.Minute
	var allEvents []models.BlockEvent

	for _, conn := range eventConnectors {
		log.Printf("[INFO] Fetching events from connector [%s]...", conn.Name())
		evts, err := conn.FetchEvents(ctx, collectionWindow)
		if err != nil {
			log.Printf("[WARN] Connector [%s] failed: %v", conn.Name(), err)
			continue
		}
		log.Printf("[INFO] Connector [%s] retrieved %d events.", conn.Name(), len(evts))
		allEvents = append(allEvents, evts...)
	}

	// 5. Fetch Traffic Metrics (Prometheus + CloudWatch)
	promEndpoint := getEnv("PROMETHEUS_ENDPOINT", "http://kube-prometheus-stack-prometheus.monitoring.svc:9090")
	promClient := prometheus.NewClient(prometheus.Config{Endpoint: promEndpoint})

	log.Println("[INFO] Fetching gateway request totals from Prometheus...")
	promTotals, err := promClient.FetchHourlyGatewayTotals(ctx)
	if err != nil {
		log.Printf("[WARN] Failed to fetch Prometheus gateway totals: %v", err)
	}

	var cfMetrics awsmetrics.CloudFrontMetrics
	if awsMetricsClient != nil {
		distID := getEnv("CLOUDFRONT_DISTRIBUTION_ID", "")
		webACLName := getEnv("WAF_WEBACL_NAME", "WWW-WebACL")
		now := time.Now().UTC()
		oneHourAgo := now.Add(-1 * time.Hour)

		if distID != "" && webACLName != "" {
			cfMetrics, err = awsMetricsClient.FetchHourlyCloudFrontMetrics(ctx, distID, webACLName, oneHourAgo, now)
			if err != nil {
				log.Printf("[WARN] Failed to fetch CloudWatch metrics: %v", err)
			}
		}
	}

	// 6. Enrich events with GeoIP and rDNS
	for i := range allEvents {
		e := &allEvents[i]

		// GeoIP enrichment
		if geoResolver != nil && e.SrcIP != "" {
			geo := geoResolver.Lookup(e.SrcIP)
			if e.CountryCode == "" {
				e.CountryCode = geo.CountryCode
			}
			e.CountryName = geo.CountryName
			e.CityName = geo.CityName
			e.Latitude = geo.Latitude
			e.Longitude = geo.Longitude
			e.ASN = geo.ASN
			e.ASOrg = geo.ASOrg
		}

		// rDNS & scanner classification
		if e.SrcIP != "" {
			actor := rdnsTagger.Lookup(ctx, e.SrcIP, e.UserAgent)
			e.ReverseDNS = actor.ReverseDNS
			e.ActorType = actor.ActorType
			e.ActorName = actor.ActorName
		}
	}

	// 7. Store block events into PostgreSQL (sliding hot partition)
	if len(allEvents) > 0 {
		log.Printf("[INFO] Saving %d enriched events into PostgreSQL...", len(allEvents))
		if err := db.InsertBlockEvents(ctx, allEvents); err != nil {
			log.Printf("[ERROR] Failed to insert block events: %v", err)
		}
	}

	// 8. Compute hourly metrics using deduplication formula
	hourlyBucket := time.Now().UTC().Truncate(time.Hour)
	foramen444 := int64(0)
	foramenModSec := int64(0)
	retis444 := int64(0)
	retisModSec := int64(0)

	for _, e := range allEvents {
		switch e.Layer {
		case "HTTP_444":
			if e.GatewayNode == "retis" {
				retis444++
			} else {
				foramen444++
			}
		case "MODSEC_BLOCK", "APACHE_403":
			if e.GatewayNode == "retis" {
				retisModSec++
			} else {
				foramenModSec++
			}
		}
	}

	hourlyStats := models.HourlyTrafficStats{
		BucketTime:          hourlyBucket,
		CFRequests:          cfMetrics.Requests,
		CFBytesDownloaded:   cfMetrics.BytesDownloaded,
		AWSWAFBlocked:       cfMetrics.WAFBlocked,
		Foramen444Drops:     foramen444,
		ForamenModSecBlocks: foramenModSec,
		RetisTotalRequests:  promTotals.RetisTotal,
		Retis444Drops:       retis444,
		RetisModSecBlocks:   retisModSec,
	}

	dedupe.CalculateHourlyMetrics(&hourlyStats)
	log.Printf("[INFO] Hourly stats: Total=%d, Mitigated=%d, Legitimate=%d, HostileRatio=%.2f%%",
		hourlyStats.TotalRequests, hourlyStats.MitigatedRequests, hourlyStats.LegitimateRequests, hourlyStats.MaliciousRatio,
	)

	if err := db.UpsertHourlyStats(ctx, hourlyStats); err != nil {
		log.Printf("[ERROR] Failed to upsert hourly stats: %v", err)
	}

	log.Println("[INFO] AdytumFama hourly collector completed successfully.")
}
