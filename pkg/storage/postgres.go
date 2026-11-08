package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/al17091/adytum_fama/pkg/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// DB manages PostgreSQL connections and partitioned storage
type DB struct {
	pool *pgxpool.Pool
}

// NewDB initializes connection pool to CloudNativePG PostgreSQL
func NewDB(ctx context.Context, dsn string) (*DB, error) {
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("invalid postgres dsn: %w", err)
	}

	config.MaxConns = 10
	config.MinConns = 2
	config.MaxConnLifetime = 30 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to postgres: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping postgres: %w", err)
	}

	return &DB{pool: pool}, nil
}

// Close terminates connection pool
func (db *DB) Close() {
	if db.pool != nil {
		db.pool.Close()
	}
}

// InitSchema sets up partitioned events table, traffic stats, and snapshot tables
func (db *DB) InitSchema(ctx context.Context) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS security_block_events (
			event_id VARCHAR(64) NOT NULL,
			observed_at TIMESTAMPTZ NOT NULL,
			layer VARCHAR(32) NOT NULL,
			is_blocked BOOLEAN NOT NULL DEFAULT true,
			gateway_node VARCHAR(32),
			src_ip VARCHAR(45) NOT NULL,
			country_code VARCHAR(4),
			country_name VARCHAR(64),
			city_name VARCHAR(64),
			latitude REAL,
			longitude REAL,
			asn INT,
			as_org VARCHAR(128),
			reverse_dns VARCHAR(255),
			actor_type VARCHAR(32),
			actor_name VARCHAR(64),
			classification VARCHAR(16) NOT NULL DEFAULT '-',
			http_method VARCHAR(16),
			http_protocol VARCHAR(16),
			tls_protocol VARCHAR(16),
			request_host VARCHAR(255),
			request_uri TEXT,
			user_agent TEXT,
			threat_category VARCHAR(64),
			ja3_fingerprint VARCHAR(64),
			ja4_fingerprint VARCHAR(64),
			waf_rule_ids TEXT[],
			waf_anomaly_score INT DEFAULT 0,
			PRIMARY KEY (event_id, observed_at)
		) PARTITION BY RANGE (observed_at);`,

		`CREATE TABLE IF NOT EXISTS hourly_traffic_stats (
			bucket_time TIMESTAMPTZ PRIMARY KEY,
			cf_requests BIGINT NOT NULL DEFAULT 0,
			cf_bytes_downloaded BIGINT NOT NULL DEFAULT 0,
			aws_waf_blocked BIGINT NOT NULL DEFAULT 0,
			foramen_444_drops BIGINT NOT NULL DEFAULT 0,
			foramen_modsec_blocks BIGINT NOT NULL DEFAULT 0,
			retis_total_requests BIGINT NOT NULL DEFAULT 0,
			retis_444_drops BIGINT NOT NULL DEFAULT 0,
			retis_modsec_blocks BIGINT NOT NULL DEFAULT 0,
			total_requests BIGINT NOT NULL DEFAULT 0,
			mitigated_requests BIGINT NOT NULL DEFAULT 0,
			legitimate_requests BIGINT NOT NULL DEFAULT 0,
			malicious_ratio DOUBLE PRECISION NOT NULL DEFAULT 0.0,
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);`,

		`CREATE TABLE IF NOT EXISTS dashboard_snapshots (
			window_type VARCHAR(16) PRIMARY KEY, -- '24h' or '7d'
			payload JSONB NOT NULL,
			generated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);`,

		`ALTER TABLE security_block_events ADD COLUMN IF NOT EXISTS classification VARCHAR(16) NOT NULL DEFAULT '-';`,
		`ALTER TABLE security_block_events ADD COLUMN IF NOT EXISTS ja3_fingerprint VARCHAR(64);`,
		`ALTER TABLE security_block_events ADD COLUMN IF NOT EXISTS ja4_fingerprint VARCHAR(64);`,
	}

	for _, q := range queries {
		if _, err := db.pool.Exec(ctx, q); err != nil {
			return fmt.Errorf("failed executing schema query: %w", err)
		}
	}

	// Pre-create partitions for today, past 8 days, and next 2 days
	now := time.Now().UTC()
	for i := -8; i <= 2; i++ {
		target := now.AddDate(0, 0, i)
		if err := db.EnsurePartitionForDate(ctx, target); err != nil {
			return err
		}
	}

	return nil
}

// EnsurePartitionForDate creates a daily partition table if not exists
func (db *DB) EnsurePartitionForDate(ctx context.Context, targetDate time.Time) error {
	utc := targetDate.UTC()
	start := time.Date(utc.Year(), utc.Month(), utc.Day(), 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 0, 1)

	partName := fmt.Sprintf("security_block_events_%s", start.Format("2006_01_02"))
	query := fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s
		PARTITION OF security_block_events
		FOR VALUES FROM ('%s') TO ('%s');`,
		partName, start.Format("2006-01-02 15:04:05-07"), end.Format("2006-01-02 15:04:05-07"),
	)

	_, err := db.pool.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to create partition %s: %w", partName, err)
	}
	return nil
}

// InsertBlockEvents inserts a batch of block events, ignoring duplicates
func (db *DB) InsertBlockEvents(ctx context.Context, events []models.BlockEvent) error {
	if len(events) == 0 {
		return nil
	}

	batch := &pgx.Batch{}
	query := `INSERT INTO security_block_events (
		event_id, observed_at, layer, is_blocked, gateway_node, src_ip,
		country_code, country_name, city_name, latitude, longitude,
		asn, as_org, reverse_dns, actor_type, actor_name, classification,
		http_method, http_protocol, tls_protocol, request_host,
		request_uri, user_agent, threat_category, ja3_fingerprint, ja4_fingerprint,
		waf_rule_ids, waf_anomaly_score
	) VALUES (
		$1, $2, $3, $4, $5, $6,
		$7, $8, $9, $10, $11,
		$12, $13, $14, $15, $16, $17,
		$18, $19, $20, $21,
		$22, $23, $24, $25, $26,
		$27, $28
	) ON CONFLICT (event_id, observed_at) DO NOTHING;`

	for _, e := range events {
		batch.Queue(query,
			e.EventID, e.ObservedAt, e.Layer, e.IsBlocked, e.GatewayNode, e.SrcIP,
			e.CountryCode, e.CountryName, e.CityName, e.Latitude, e.Longitude,
			e.ASN, e.ASOrg, e.ReverseDNS, e.ActorType, e.ActorName, e.Classification,
			e.HTTPMethod, e.HTTPProtocol, e.TLSProtocol, e.RequestHost,
			e.RequestURI, e.UserAgent, e.ThreatCategory, e.JA3Fingerprint, e.JA4Fingerprint,
			e.WAFRuleIDs, e.WAFAnomalyScore,
		)
	}

	br := db.pool.SendBatch(ctx, batch)
	defer br.Close()

	for range events {
		if _, err := br.Exec(); err != nil {
			return fmt.Errorf("failed inserting block event batch: %w", err)
		}
	}

	return nil
}

// UpsertHourlyStats stores calculated hourly metrics
func (db *DB) UpsertHourlyStats(ctx context.Context, stats models.HourlyTrafficStats) error {
	query := `INSERT INTO hourly_traffic_stats (
		bucket_time, cf_requests, cf_bytes_downloaded, aws_waf_blocked,
		foramen_444_drops, foramen_modsec_blocks, retis_total_requests,
		retis_444_drops, retis_modsec_blocks, total_requests,
		mitigated_requests, legitimate_requests, malicious_ratio, updated_at
	) VALUES (
		$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, NOW()
	) ON CONFLICT (bucket_time) DO UPDATE SET
		cf_requests = EXCLUDED.cf_requests,
		cf_bytes_downloaded = EXCLUDED.cf_bytes_downloaded,
		aws_waf_blocked = EXCLUDED.aws_waf_blocked,
		foramen_444_drops = EXCLUDED.foramen_444_drops,
		foramen_modsec_blocks = EXCLUDED.foramen_modsec_blocks,
		retis_total_requests = EXCLUDED.retis_total_requests,
		retis_444_drops = EXCLUDED.retis_444_drops,
		retis_modsec_blocks = EXCLUDED.retis_modsec_blocks,
		total_requests = EXCLUDED.total_requests,
		mitigated_requests = EXCLUDED.mitigated_requests,
		legitimate_requests = EXCLUDED.legitimate_requests,
		malicious_ratio = EXCLUDED.malicious_ratio,
		updated_at = NOW();`

	_, err := db.pool.Exec(ctx, query,
		stats.BucketTime, stats.CFRequests, stats.CFBytesDownloaded, stats.AWSWAFBlocked,
		stats.Foramen444Drops, stats.ForamenModSecBlocks, stats.RetisTotalRequests,
		stats.Retis444Drops, stats.RetisModSecBlocks, stats.TotalRequests,
		stats.MitigatedRequests, stats.LegitimateRequests, stats.MaliciousRatio,
	)
	if err != nil {
		return fmt.Errorf("failed upserting hourly stats: %w", err)
	}
	return nil
}

// GetEventsForDate extracts all events within a UTC day for Parquet compaction
func (db *DB) GetEventsForDate(ctx context.Context, date time.Time) ([]models.BlockEvent, error) {
	utc := date.UTC()
	start := time.Date(utc.Year(), utc.Month(), utc.Day(), 0, 0, 0, 0, time.UTC)
	end := start.AddDate(0, 0, 1)

	query := `SELECT
		event_id, observed_at, layer, is_blocked, COALESCE(gateway_node, ''),
		src_ip, COALESCE(country_code, 'XX'), COALESCE(country_name, 'Unknown'),
		COALESCE(city_name, 'Unknown'), COALESCE(latitude, 0), COALESCE(longitude, 0),
		COALESCE(asn, 0), COALESCE(as_org, 'Unknown'), COALESCE(reverse_dns, ''),
		COALESCE(actor_type, 'UNKNOWN'), COALESCE(actor_name, 'Unknown'),
		COALESCE(classification, '-'),
		COALESCE(http_method, ''), COALESCE(http_protocol, ''), COALESCE(tls_protocol, ''),
		COALESCE(request_host, ''), COALESCE(request_uri, ''), COALESCE(user_agent, ''),
		COALESCE(threat_category, ''), COALESCE(ja3_fingerprint, ''), COALESCE(ja4_fingerprint, ''),
		COALESCE(waf_rule_ids, ARRAY[]::TEXT[]), COALESCE(waf_anomaly_score, 0)
	FROM security_block_events
	WHERE observed_at >= $1 AND observed_at < $2
	ORDER BY observed_at ASC;`

	rows, err := db.pool.Query(ctx, query, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed querying events for %s: %w", start.Format("2006-01-02"), err)
	}
	defer rows.Close()

	var events []models.BlockEvent
	for rows.Next() {
		var e models.BlockEvent
		err := rows.Scan(
			&e.EventID, &e.ObservedAt, &e.Layer, &e.IsBlocked, &e.GatewayNode,
			&e.SrcIP, &e.CountryCode, &e.CountryName, &e.CityName, &e.Latitude, &e.Longitude,
			&e.ASN, &e.ASOrg, &e.ReverseDNS, &e.ActorType, &e.ActorName,
			&e.Classification,
			&e.HTTPMethod, &e.HTTPProtocol, &e.TLSProtocol, &e.RequestHost,
			&e.RequestURI, &e.UserAgent, &e.ThreatCategory,
			&e.JA3Fingerprint, &e.JA4Fingerprint,
			&e.WAFRuleIDs, &e.WAFAnomalyScore,
		)
		if err != nil {
			return nil, fmt.Errorf("failed scanning event row: %w", err)
		}
		events = append(events, e)
	}

	return events, nil
}

// SaveSnapshot caches the pre-built JSON snapshot for instant /guardinfo delivery
func (db *DB) SaveSnapshot(ctx context.Context, window string, payload models.SnapshotPayload) error {
	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("failed marshaling snapshot json: %w", err)
	}

	query := `INSERT INTO dashboard_snapshots (window_type, payload, generated_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (window_type) DO UPDATE SET
			payload = EXCLUDED.payload,
			generated_at = NOW();`

	_, err = db.pool.Exec(ctx, query, window, jsonBytes)
	if err != nil {
		return fmt.Errorf("failed saving snapshot (%s): %w", window, err)
	}
	return nil
}

// GetSnapshot retrieves the cached JSON payload
func (db *DB) GetSnapshot(ctx context.Context, window string) (*models.SnapshotPayload, error) {
	var jsonBytes []byte
	query := `SELECT payload FROM dashboard_snapshots WHERE window_type = $1;`

	err := db.pool.QueryRow(ctx, query, window).Scan(&jsonBytes)
	if err != nil {
		return nil, err
	}

	var payload models.SnapshotPayload
	if err := json.Unmarshal(jsonBytes, &payload); err != nil {
		return nil, fmt.Errorf("failed unmarshaling snapshot payload: %w", err)
	}
	return &payload, nil
}

// BuildSnapshot aggregates stats from hourly_traffic_stats and security_block_events
func (db *DB) BuildSnapshot(ctx context.Context, window string) (*models.SnapshotPayload, error) {
	duration := 24 * time.Hour
	if window == "7d" {
		duration = 7 * 24 * time.Hour
	}
	since := time.Now().UTC().Add(-duration)

	payload := &models.SnapshotPayload{
		Window:       window,
		GeneratedAt:  time.Now().UTC(),
		HourlySeries: make([]models.HourlyTrafficStats, 0),
		TopCountries: make([]models.CountryStat, 0),
		TopActors:    make([]models.ActorStat, 0),
		RecentEvents: make([]models.PublicBlockEvent, 0),
	}

	// 1. Hourly Series
	seriesQuery := `SELECT
		bucket_time, cf_requests, cf_bytes_downloaded, aws_waf_blocked,
		foramen_444_drops, foramen_modsec_blocks, retis_total_requests,
		retis_444_drops, retis_modsec_blocks, total_requests,
		mitigated_requests, legitimate_requests, malicious_ratio
	FROM hourly_traffic_stats
	WHERE bucket_time >= $1
	ORDER BY bucket_time ASC;`

	rows, err := db.pool.Query(ctx, seriesQuery, since)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var s models.HourlyTrafficStats
			if err := rows.Scan(
				&s.BucketTime, &s.CFRequests, &s.CFBytesDownloaded, &s.AWSWAFBlocked,
				&s.Foramen444Drops, &s.ForamenModSecBlocks, &s.RetisTotalRequests,
				&s.Retis444Drops, &s.RetisModSecBlocks, &s.TotalRequests,
				&s.MitigatedRequests, &s.LegitimateRequests, &s.MaliciousRatio,
			); err == nil {
				payload.HourlySeries = append(payload.HourlySeries, s)
				payload.TotalRequests += s.TotalRequests
				payload.MitigatedRequests += s.MitigatedRequests
				payload.LegitimateRequests += s.LegitimateRequests
				payload.LayerBreakdown.AWSWAFBlocked += s.AWSWAFBlocked
				payload.LayerBreakdown.Foramen444Drops += s.Foramen444Drops
				payload.LayerBreakdown.ForamenModSecBlocks += s.ForamenModSecBlocks
				payload.LayerBreakdown.Retis444Drops += s.Retis444Drops
				payload.LayerBreakdown.RetisModSecBlocks += s.RetisModSecBlocks
			}
		}
		if payload.TotalRequests < payload.MitigatedRequests {
			payload.TotalRequests = payload.MitigatedRequests
		}
		if payload.TotalRequests > 0 {
			ratio := float64(payload.MitigatedRequests) / float64(payload.TotalRequests) * 100.0
			if ratio > 100.0 {
				ratio = 100.0
			}
			payload.MaliciousRatio = math.Round(ratio*100) / 100.0
		}
	}

	// 2. Top Countries
	countryQuery := `SELECT
		COALESCE(country_code, 'XX'),
		COALESCE(country_name, 'Unknown'),
		COUNT(*) as c
	FROM security_block_events
	WHERE observed_at >= $1
	GROUP BY country_code, country_name
	ORDER BY c DESC
	LIMIT 10;`

	cRows, err := db.pool.Query(ctx, countryQuery, since)
	if err == nil {
		defer cRows.Close()
		for cRows.Next() {
			var cs models.CountryStat
			if err := cRows.Scan(&cs.CountryCode, &cs.CountryName, &cs.Count); err == nil {
				if payload.MitigatedRequests > 0 {
					cs.Percentage = float64(cs.Count) / float64(payload.MitigatedRequests) * 100.0
				}
				payload.TopCountries = append(payload.TopCountries, cs)
			}
		}
	}

	// 3. Top Actors
	actorQuery := `SELECT
		COALESCE(actor_type, 'UNKNOWN'),
		COALESCE(actor_name, 'Unknown'),
		COUNT(*) as c
	FROM security_block_events
	WHERE observed_at >= $1 AND actor_type != 'UNKNOWN'
	GROUP BY actor_type, actor_name
	ORDER BY c DESC
	LIMIT 10;`

	aRows, err := db.pool.Query(ctx, actorQuery, since)
	if err == nil {
		defer aRows.Close()
		for aRows.Next() {
			var as models.ActorStat
			if err := aRows.Scan(&as.ActorType, &as.ActorName, &as.Count); err == nil {
				payload.TopActors = append(payload.TopActors, as)
			}
		}
	}

	// 4. Recent Events (masked for /guardinfo public consumption)
	eventQuery := `SELECT
		observed_at, layer, src_ip,
		COALESCE(country_code, 'XX'), COALESCE(country_name, 'Unknown'),
		COALESCE(city_name, 'Unknown'), COALESCE(latitude, 0), COALESCE(longitude, 0),
		COALESCE(asn, 0), COALESCE(as_org, 'Unknown'),
		COALESCE(actor_type, 'UNKNOWN'), COALESCE(actor_name, 'Unknown'),
		COALESCE(classification, '-'),
		COALESCE(http_method, ''), COALESCE(request_uri, '/'), COALESCE(threat_category, ''),
		COALESCE(ja3_fingerprint, ''), COALESCE(ja4_fingerprint, ''),
		COALESCE(user_agent, '')
	FROM security_block_events
	WHERE observed_at >= $1
	ORDER BY observed_at DESC
	LIMIT 100;`

	eRows, err := db.pool.Query(ctx, eventQuery, since)
	if err == nil {
		defer eRows.Close()
		for eRows.Next() {
			var pe models.PublicBlockEvent
			if err := eRows.Scan(
				&pe.ObservedAt, &pe.Layer, &pe.SrcIP,
				&pe.CountryCode, &pe.CountryName,
				&pe.CityName, &pe.Latitude, &pe.Longitude,
				&pe.ASN, &pe.ASOrg,
				&pe.ActorType, &pe.ActorName,
				&pe.Classification,
				&pe.HTTPMethod, &pe.RequestURI, &pe.ThreatCategory,
				&pe.JA3Fingerprint, &pe.JA4Fingerprint,
				&pe.UserAgent,
			); err == nil {
				payload.RecentEvents = append(payload.RecentEvents, pe)
			}
		}
	}

	return payload, nil
}

// EventFilter defines search and pagination parameters for the log explorer
type EventFilter struct {
	Since          time.Time
	Until          time.Time
	Layer          string // optional filter by layer
	Classification string // optional filter by classification
	Search         string // keyword search across ip, uri, user_agent, as_org
	Limit          int
	Offset         int
}

// QueryEventsResult holds paginated results from QueryBlockEvents
type QueryEventsResult struct {
	Total  int64                     `json:"total"`
	Events []models.PublicBlockEvent `json:"events"`
	Limit  int                       `json:"limit"`
	Offset int                       `json:"offset"`
}

// QueryBlockEvents retrieves filtered, paginated events across historical days
func (db *DB) QueryBlockEvents(ctx context.Context, f EventFilter) (*QueryEventsResult, error) {
	if f.Limit <= 0 {
		f.Limit = 100
	} else if f.Limit > 500 {
		f.Limit = 500
	}
	if f.Offset < 0 {
		f.Offset = 0
	}
	if f.Since.IsZero() {
		f.Since = time.Now().UTC().Add(-7 * 24 * time.Hour)
	}

	whereClauses := []string{"observed_at >= $1"}
	args := []interface{}{f.Since}
	argIdx := 2

	if !f.Until.IsZero() {
		whereClauses = append(whereClauses, fmt.Sprintf("observed_at <= $%d", argIdx))
		args = append(args, f.Until)
		argIdx++
	}

	if f.Layer != "" && f.Layer != "ALL" {
		whereClauses = append(whereClauses, fmt.Sprintf("layer = $%d", argIdx))
		args = append(args, f.Layer)
		argIdx++
	}

	if f.Classification != "" && f.Classification != "ALL" {
		whereClauses = append(whereClauses, fmt.Sprintf("classification = $%d", argIdx))
		args = append(args, f.Classification)
		argIdx++
	}

	if f.Search != "" {
		s := "%" + strings.TrimSpace(f.Search) + "%"
		whereClauses = append(whereClauses, fmt.Sprintf("(src_ip ILIKE $%d OR request_uri ILIKE $%d OR user_agent ILIKE $%d OR as_org ILIKE $%d)", argIdx, argIdx, argIdx, argIdx))
		args = append(args, s)
		argIdx++
	}

	whereSQL := strings.Join(whereClauses, " AND ")

	// Count matching events
	countSQL := fmt.Sprintf("SELECT COUNT(*) FROM security_block_events WHERE %s", whereSQL)
	var total int64
	if err := db.pool.QueryRow(ctx, countSQL, args...).Scan(&total); err != nil {
		return nil, fmt.Errorf("failed counting filtered events: %w", err)
	}

	// Fetch page
	selectSQL := fmt.Sprintf(`SELECT
		observed_at, layer, src_ip,
		COALESCE(country_code, 'XX'), COALESCE(country_name, 'Unknown'),
		COALESCE(city_name, 'Unknown'), COALESCE(latitude, 0), COALESCE(longitude, 0),
		COALESCE(asn, 0), COALESCE(as_org, 'Unknown'),
		COALESCE(actor_type, 'UNKNOWN'), COALESCE(actor_name, 'Unknown'),
		COALESCE(classification, '-'),
		COALESCE(http_method, ''), COALESCE(request_uri, '/'), COALESCE(threat_category, ''),
		COALESCE(ja3_fingerprint, ''), COALESCE(ja4_fingerprint, ''),
		COALESCE(user_agent, '')
	FROM security_block_events
	WHERE %s
	ORDER BY observed_at DESC
	LIMIT $%d OFFSET $%d`, whereSQL, argIdx, argIdx+1)

	args = append(args, f.Limit, f.Offset)

	rows, err := db.pool.Query(ctx, selectSQL, args...)
	if err != nil {
		return nil, fmt.Errorf("failed querying filtered events: %w", err)
	}
	defer rows.Close()

	events := make([]models.PublicBlockEvent, 0, f.Limit)
	for rows.Next() {
		var pe models.PublicBlockEvent
		if err := rows.Scan(
			&pe.ObservedAt, &pe.Layer, &pe.SrcIP,
			&pe.CountryCode, &pe.CountryName,
			&pe.CityName, &pe.Latitude, &pe.Longitude,
			&pe.ASN, &pe.ASOrg,
			&pe.ActorType, &pe.ActorName,
			&pe.Classification,
			&pe.HTTPMethod, &pe.RequestURI, &pe.ThreatCategory,
			&pe.JA3Fingerprint, &pe.JA4Fingerprint,
			&pe.UserAgent,
		); err == nil {
			events = append(events, pe)
		}
	}

	return &QueryEventsResult{
		Total:  total,
		Events: events,
		Limit:  f.Limit,
		Offset: f.Offset,
	}, nil
}
