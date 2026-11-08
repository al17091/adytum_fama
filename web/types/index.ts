export interface PublicBlockEvent {
  observed_at: string;
  layer: string; // MODSEC_BLOCK, MODSEC_DETECT, HTTP_444, AWS_WAF_SAMPLE, APACHE_403
  src_ip: string;
  country_code: string;
  country_name: string;
  city_name: string;
  latitude: number;
  longitude: number;
  asn: number;
  as_org: string;
  actor_type: string;
  actor_name: string;
  classification: string; // MALICIOUS, SUSPICIOUS, -
  http_method: string;
  request_uri: string;
  threat_category: string;
  user_agent: string;
}

export interface HourlyTrafficStats {
  bucket_time: string;
  cf_requests: number;
  cf_bytes_downloaded: number;
  aws_waf_blocked: number;
  foramen_444_drops: number;
  foramen_modsec_blocks: number;
  retis_total_requests: number;
  retis_444_drops: number;
  retis_modsec_blocks: number;
  total_requests: number;
  mitigated_requests: number;
  legitimate_requests: number;
  malicious_ratio: number;
}

export interface LayerBreakdown {
  aws_waf_blocked: number;
  foramen_444_drops: number;
  foramen_modsec_blocks: number;
  retis_444_drops: number;
  retis_modsec_blocks: number;
}

export interface CountryStat {
  country_code: string;
  country_name: string;
  count: number;
  percentage: number;
}

export interface ActorStat {
  actor_type: string;
  actor_name: string;
  count: number;
}

export interface SnapshotPayload {
  window: "24h" | "7d";
  generated_at: string;
  total_requests: number;
  mitigated_requests: number;
  legitimate_requests: number;
  malicious_ratio: number;
  layer_breakdown: LayerBreakdown;
  hourly_series: HourlyTrafficStats[];
  top_countries: CountryStat[];
  top_actors: ActorStat[];
  recent_events: PublicBlockEvent[];
}
