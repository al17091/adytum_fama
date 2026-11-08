package models

import (
	"time"
)

// BlockEvent represents a single mitigated security event (ModSec, Nginx 444, AWS WAF sample)
// exactly matching the Apache Parquet schema and PostgreSQL storage model.
type BlockEvent struct {
	EventID         string    `json:"event_id" parquet:"event_id"`
	ObservedAt      time.Time `json:"observed_at" parquet:"observed_at,timestamp(millisecond)"`
	Layer           string    `json:"layer" parquet:"layer"`                         // MODSEC_BLOCK, MODSEC_DETECT, HTTP_444, AWS_WAF_SAMPLE
	IsBlocked       bool      `json:"is_blocked" parquet:"is_blocked"`
	GatewayNode     string    `json:"gateway_node,omitempty" parquet:"gateway_node"` // foramen, retis, edge (masked in public API)
	SrcIP           string    `json:"src_ip" parquet:"src_ip"`
	CountryCode     string    `json:"country_code" parquet:"country_code"`
	CountryName     string    `json:"country_name" parquet:"country_name"`
	CityName        string    `json:"city_name" parquet:"city_name"`
	Latitude        float32   `json:"latitude" parquet:"latitude"`
	Longitude       float32   `json:"longitude" parquet:"longitude"`
	ASN             uint32    `json:"asn" parquet:"asn"`
	ASOrg           string    `json:"as_org" parquet:"as_org"`
	ReverseDNS      string    `json:"reverse_dns" parquet:"reverse_dns"`
	ActorType       string    `json:"actor_type" parquet:"actor_type"` // RESEARCH_SCANNER, BOT_SCANNER, UNKNOWN
	ActorName       string    `json:"actor_name" parquet:"actor_name"` // Censys, Shodan, Shadowserver, etc.
	Classification  string    `json:"classification" parquet:"classification"` // MALICIOUS, SUSPICIOUS, -
	HTTPMethod      string    `json:"http_method" parquet:"http_method"`
	HTTPProtocol    string    `json:"http_protocol" parquet:"http_protocol"` // HTTP/1.1, HTTP/2.0, HTTP/3.0
	TLSProtocol     string    `json:"tls_protocol" parquet:"tls_protocol"`   // TLSv1.2, TLSv1.3
	RequestHost     string    `json:"request_host,omitempty" parquet:"request_host"`
	RequestURI      string    `json:"request_uri" parquet:"request_uri"`
	UserAgent       string    `json:"user_agent" parquet:"user_agent"`
	ThreatCategory  string    `json:"threat_category" parquet:"threat_category"` // SQLI, XSS, DOTFILE, CMS_PROBE, RAW_IP_RECON
	JA3Fingerprint  string    `json:"ja3_fingerprint,omitempty" parquet:"ja3_fingerprint"`
	JA4Fingerprint  string    `json:"ja4_fingerprint,omitempty" parquet:"ja4_fingerprint"`
	WAFRuleIDs      []string  `json:"waf_rule_ids" parquet:"waf_rule_ids,list"`
	WAFAnomalyScore int32     `json:"waf_anomaly_score" parquet:"waf_anomaly_score"`
}

// HourlyTrafficStats holds hourly ingress totals and mitigation counts
type HourlyTrafficStats struct {
	BucketTime          time.Time `json:"bucket_time" parquet:"bucket_time,timestamp(millisecond)"`
	CFRequests          int64     `json:"cf_requests" parquet:"cf_requests"`
	CFBytesDownloaded   int64     `json:"cf_bytes_downloaded" parquet:"cf_bytes_downloaded"`
	AWSWAFBlocked       int64     `json:"aws_waf_blocked" parquet:"aws_waf_blocked"`
	Foramen444Drops     int64     `json:"foramen_444_drops" parquet:"foramen_444_drops"`
	ForamenModSecBlocks int64     `json:"foramen_modsec_blocks" parquet:"foramen_modsec_blocks"`
	RetisTotalRequests  int64     `json:"retis_total_requests" parquet:"retis_total_requests"`
	Retis444Drops       int64     `json:"retis_444_drops" parquet:"retis_444_drops"`
	RetisModSecBlocks   int64     `json:"retis_modsec_blocks" parquet:"retis_modsec_blocks"`
	TotalRequests       int64     `json:"total_requests" parquet:"total_requests"`
	MitigatedRequests   int64     `json:"mitigated_requests" parquet:"mitigated_requests"`
	LegitimateRequests  int64     `json:"legitimate_requests" parquet:"legitimate_requests"`
	MaliciousRatio      float64   `json:"malicious_ratio" parquet:"malicious_ratio"`
}

// LayerBreakdown summarizes counts per mitigation layer
type LayerBreakdown struct {
	AWSWAFBlocked       int64 `json:"aws_waf_blocked"`
	Foramen444Drops     int64 `json:"foramen_444_drops"`
	ForamenModSecBlocks int64 `json:"foramen_modsec_blocks"`
	Retis444Drops       int64 `json:"retis_444_drops"`
	RetisModSecBlocks   int64 `json:"retis_modsec_blocks"`
}

// CountryStat summarizes attacks by country
type CountryStat struct {
	CountryCode string  `json:"country_code"`
	CountryName string  `json:"country_name"`
	Count       int64   `json:"count"`
	Percentage  float64 `json:"percentage"`
}

// ActorStat summarizes attacks by scanner/bot actor
type ActorStat struct {
	ActorType  string `json:"actor_type"`
	ActorName  string `json:"actor_name"`
	Count      int64  `json:"count"`
}

// PublicBlockEvent masks internal details for /guardinfo public consumption
type PublicBlockEvent struct {
	ObservedAt     time.Time `json:"observed_at"`
	Layer          string    `json:"layer"`
	SrcIP          string    `json:"src_ip"` // Raw full IP (plain text, never linked in UI)
	CountryCode    string    `json:"country_code"`
	CountryName    string    `json:"country_name"`
	CityName       string    `json:"city_name"`
	Latitude       float32   `json:"latitude"`
	Longitude      float32   `json:"longitude"`
	ASN            uint32    `json:"asn"`
	ASOrg          string    `json:"as_org"`
	ActorType      string    `json:"actor_type"`
	ActorName      string    `json:"actor_name"`
	Classification string    `json:"classification"` // MALICIOUS, SUSPICIOUS, -
	HTTPMethod     string    `json:"http_method"`
	RequestURI     string    `json:"request_uri"`
	ThreatCategory string    `json:"threat_category"`
	JA3Fingerprint string    `json:"ja3_fingerprint,omitempty"`
	JA4Fingerprint string    `json:"ja4_fingerprint,omitempty"`
	UserAgent      string    `json:"user_agent"`
}

// SnapshotPayload is the pre-aggregated JSON payload served to /guardinfo
type SnapshotPayload struct {
	Window             string               `json:"window"` // 24h or 7d
	GeneratedAt        time.Time            `json:"generated_at"`
	TotalRequests      int64                `json:"total_requests"`
	MitigatedRequests  int64                `json:"mitigated_requests"`
	LegitimateRequests int64                `json:"legitimate_requests"`
	MaliciousRatio     float64              `json:"malicious_ratio"`
	LayerBreakdown     LayerBreakdown       `json:"layer_breakdown"`
	HourlySeries       []HourlyTrafficStats `json:"hourly_series"`
	TopCountries       []CountryStat        `json:"top_countries"`
	TopActors          []ActorStat          `json:"top_actors"`
	RecentEvents       []PublicBlockEvent   `json:"recent_events"`
}
