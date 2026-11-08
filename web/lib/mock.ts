import { SnapshotPayload } from "../types";

export function getFallbackSnapshot(window: "24h" | "7d"): SnapshotPayload {
  const count = window === "24h" ? 24 : 168;
  const now = new Date();
  const hourly_series = [];

  for (let i = count - 1; i >= 0; i--) {
    const t = new Date(now.getTime() - i * 3600 * 1000);
    const cf = 400 + Math.floor(Math.sin(i / 3) * 150) + Math.floor(Math.random() * 80);
    const waf = Math.floor(cf * 0.12) + Math.floor(Math.random() * 15);
    const f444 = 40 + Math.floor(Math.random() * 25);
    const fMod = 3 + Math.floor(Math.random() * 5);
    const retisTotal = 120 + Math.floor(Math.random() * 40);
    const retis444 = 15 + Math.floor(Math.random() * 10);
    const retisMod = 1 + Math.floor(Math.random() * 3);

    const total = cf + f444 + retisTotal;
    const mitigated = waf + f444 + fMod + retis444 + retisMod;
    const legit = Math.max(0, total - mitigated);
    const ratio = total > 0 ? (mitigated / total) * 100 : 0;

    hourly_series.push({
      bucket_time: t.toISOString(),
      cf_requests: cf,
      cf_bytes_downloaded: cf * 12400,
      aws_waf_blocked: waf,
      foramen_444_drops: f444,
      foramen_modsec_blocks: fMod,
      retis_total_requests: retisTotal,
      retis_444_drops: retis444,
      retis_modsec_blocks: retisMod,
      total_requests: total,
      mitigated_requests: mitigated,
      legitimate_requests: legit,
      malicious_ratio: parseFloat(ratio.toFixed(2)),
    });
  }

  const totalReqs = hourly_series.reduce((sum, s) => sum + s.total_requests, 0);
  const mitigatedReqs = hourly_series.reduce((sum, s) => sum + s.mitigated_requests, 0);
  const legitReqs = hourly_series.reduce((sum, s) => sum + s.legitimate_requests, 0);
  const wafTotal = hourly_series.reduce((sum, s) => sum + s.aws_waf_blocked, 0);
  const f444Total = hourly_series.reduce((sum, s) => sum + s.foramen_444_drops, 0);
  const fModTotal = hourly_series.reduce((sum, s) => sum + s.foramen_modsec_blocks, 0);
  const r444Total = hourly_series.reduce((sum, s) => sum + s.retis_444_drops, 0);
  const rModTotal = hourly_series.reduce((sum, s) => sum + s.retis_modsec_blocks, 0);

  return {
    window,
    generated_at: now.toISOString(),
    total_requests: totalReqs,
    mitigated_requests: mitigatedReqs,
    legitimate_requests: legitReqs,
    malicious_ratio: parseFloat(((mitigatedReqs / totalReqs) * 100).toFixed(1)),
    layer_breakdown: {
      aws_waf_blocked: wafTotal,
      foramen_444_drops: f444Total,
      foramen_modsec_blocks: fModTotal,
      retis_444_drops: r444Total,
      retis_modsec_blocks: rModTotal,
    },
    hourly_series,
    top_countries: [
      { country_code: "US", country_name: "United States", count: Math.floor(mitigatedReqs * 0.42), percentage: 42.1 },
      { country_code: "CN", country_name: "China", count: Math.floor(mitigatedReqs * 0.22), percentage: 22.4 },
      { country_code: "DE", country_name: "Germany", count: Math.floor(mitigatedReqs * 0.11), percentage: 11.2 },
      { country_code: "NL", country_name: "Netherlands", count: Math.floor(mitigatedReqs * 0.08), percentage: 8.5 },
      { country_code: "SG", country_name: "Singapore", count: Math.floor(mitigatedReqs * 0.05), percentage: 5.3 },
    ],
    top_actors: [
      { actor_type: "RESEARCH_SCANNER", actor_name: "Censys", count: Math.floor(mitigatedReqs * 0.28) },
      { actor_type: "RESEARCH_SCANNER", actor_name: "Shadowserver", count: Math.floor(mitigatedReqs * 0.18) },
      { actor_type: "BOT_SCANNER", actor_name: "Automated Attack / Bot", count: Math.floor(mitigatedReqs * 0.35) },
      { actor_type: "RESEARCH_SCANNER", actor_name: "Shodan", count: Math.floor(mitigatedReqs * 0.09) },
    ],
    recent_events: [
      {
        observed_at: new Date(now.getTime() - 2 * 60000).toISOString(),
        layer: "MODSEC_BLOCK",
        src_ip: "198.51.100.42",
        country_code: "US",
        country_name: "United States",
        city_name: "Ashburn",
        latitude: 39.0437,
        longitude: -77.4875,
        asn: 14618,
        as_org: "AMAZON-AES",
        actor_type: "BOT_SCANNER",
        actor_name: "Automated Attack / Bot",
        http_method: "POST",
        threat_category: "SQLI",
        user_agent: "sqlmap/1.7.2#stable (https://sqlmap.org)",
      },
      {
        observed_at: new Date(now.getTime() - 8 * 60000).toISOString(),
        layer: "HTTP_444",
        src_ip: "203.0.113.88",
        country_code: "DE",
        country_name: "Germany",
        city_name: "Frankfurt",
        latitude: 50.1109,
        longitude: 8.6821,
        asn: 24940,
        as_org: "Hetzner Online GmbH",
        actor_type: "RESEARCH_SCANNER",
        actor_name: "Censys",
        http_method: "GET",
        threat_category: "RAW_IP_RECON",
        user_agent: "Mozilla/5.0 (compatible; CensysInspect/1.1; +https://about.censys.io/)",
      },
      {
        observed_at: new Date(now.getTime() - 15 * 60000).toISOString(),
        layer: "AWS_WAF_SAMPLE",
        src_ip: "192.0.2.140",
        country_code: "SG",
        country_name: "Singapore",
        city_name: "Singapore",
        latitude: 1.3521,
        longitude: 103.8198,
        asn: 16509,
        as_org: "AMAZON-02",
        actor_type: "BOT_SCANNER",
        actor_name: "Automated Attack / Bot",
        http_method: "GET",
        threat_category: "DOTFILE",
        user_agent: "Mozilla/5.0 zgrab/0.x",
      },
      {
        observed_at: new Date(now.getTime() - 22 * 60000).toISOString(),
        layer: "HTTP_444",
        src_ip: "198.51.100.201",
        country_code: "CN",
        country_name: "China",
        city_name: "Beijing",
        latitude: 39.9042,
        longitude: 116.4074,
        asn: 4134,
        as_org: "CHINANET-BACKBONE",
        actor_type: "BOT_SCANNER",
        actor_name: "Automated Attack / Bot",
        http_method: "GET",
        threat_category: "CMS_PROBE",
        user_agent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64)",
      },
      {
        observed_at: new Date(now.getTime() - 31 * 60000).toISOString(),
        layer: "MODSEC_BLOCK",
        src_ip: "203.0.113.15",
        country_code: "NL",
        country_name: "Netherlands",
        city_name: "Amsterdam",
        latitude: 52.3676,
        longitude: 4.9041,
        asn: 49981,
        as_org: "WorldStream B.V.",
        actor_type: "RESEARCH_SCANNER",
        actor_name: "Shadowserver",
        http_method: "GET",
        threat_category: "WAF_EXPLOIT_ATTEMPT",
        user_agent: "Mozilla/5.0 (compatible; Shadowserver.org_DNS_Scanner; +https://www.shadowserver.org/)",
      },
    ],
  };
}
