"use client";

import React from "react";
import {
  BarChart3,
  Globe2,
  Bot,
  ListFilter,
  ShieldAlert,
  Search,
} from "lucide-react";
import {
  HourlyTrafficStats,
  CountryStat,
  ActorStat,
  PublicBlockEvent,
} from "../types";

interface AnalyticsGridProps {
  hourlySeries: HourlyTrafficStats[];
  topCountries: CountryStat[];
  topActors: ActorStat[];
  recentEvents: PublicBlockEvent[];
  targetCountry: string | null;
  onTargetCountryChange: (c: string | null) => void;
}

export const AnalyticsGrid: React.FC<AnalyticsGridProps> = ({
  hourlySeries,
  topCountries,
  topActors,
  recentEvents,
  targetCountry,
  onTargetCountryChange,
}) => {
  // Find max hourly total for proportional scaling
  const maxHourly = Math.max(
    1,
    ...hourlySeries.map((s) => s.total_requests)
  );

  return (
    <section className="flex flex-col gap-6 w-full max-w-7xl mx-auto px-4 lg:px-8 py-8">
      {/* Row 1: Hourly Traffic & Defense Stacked Bar Chart */}
      <div className="p-5 rounded-2xl bg-slate-900/80 border border-slate-800/80 backdrop-blur-md shadow-xl flex flex-col gap-4">
        <div className="flex items-center justify-between flex-wrap gap-2">
          <div className="flex items-center gap-2">
            <BarChart3 className="w-4 h-4 text-cyan-400" />
            <h2 className="text-sm font-bold uppercase tracking-wider text-slate-200">
              Hourly Ingress & Threat Interception Trend
            </h2>
          </div>
          {/* Legend */}
          <div className="flex items-center gap-3 text-[11px] font-mono flex-wrap">
            <span className="flex items-center gap-1.5">
              <span className="w-2.5 h-2.5 rounded-sm bg-cyan-400" />
              <span className="text-slate-300">Legitimate</span>
            </span>
            <span className="flex items-center gap-1.5">
              <span className="w-2.5 h-2.5 rounded-sm bg-purple-500" />
              <span className="text-slate-300">AWS WAF</span>
            </span>
            <span className="flex items-center gap-1.5">
              <span className="w-2.5 h-2.5 rounded-sm bg-teal-400" />
              <span className="text-slate-300">Nginx 444</span>
            </span>
            <span className="flex items-center gap-1.5">
              <span className="w-2.5 h-2.5 rounded-sm bg-rose-500" />
              <span className="text-slate-300">ModSec CRS</span>
            </span>
          </div>
        </div>

        {/* Stacked Bars */}
        <div className="w-full h-44 flex items-end gap-1 sm:gap-1.5 pt-4 pb-2 border-b border-slate-800 overflow-x-auto">
          {hourlySeries.map((stat, idx) => {
            const hTotal = stat.total_requests || 1;
            const barHeightPct = Math.min(100, Math.max(6, (hTotal / maxHourly) * 100));

            // Proportions of segments
            const legitPct = (stat.legitimate_requests / hTotal) * 100;
            const wafPct = (stat.aws_waf_blocked / hTotal) * 100;
            const dropPct =
              ((stat.foramen_444_drops + stat.retis_444_drops) / hTotal) * 100;
            const modsecPct =
              ((stat.foramen_modsec_blocks + stat.retis_modsec_blocks) / hTotal) * 100;

            const timeLabel = new Date(stat.bucket_time).getHours() + ":00";

            return (
              <div
                key={idx}
                className="flex-1 min-w-[12px] h-full flex flex-col justify-end group relative"
              >
                {/* Tooltip on hover */}
                <div className="absolute bottom-full mb-2 left-1/2 -translate-x-1/2 hidden group-hover:flex flex-col p-2 bg-slate-950 border border-slate-700 rounded-lg text-[10px] font-mono text-slate-200 z-30 whitespace-nowrap shadow-xl">
                  <div className="text-cyan-400 font-bold mb-1">
                    {new Date(stat.bucket_time).toLocaleString("ja-JP", {
                      month: "2-digit",
                      day: "2-digit",
                      hour: "2-digit",
                      minute: "2-digit",
                    })}
                  </div>
                  <div>Total: {stat.total_requests.toLocaleString()}</div>
                  <div className="text-cyan-300">
                    Legitimate: {stat.legitimate_requests.toLocaleString()}
                  </div>
                  <div className="text-purple-300">
                    AWS WAF: {stat.aws_waf_blocked.toLocaleString()}
                  </div>
                  <div className="text-teal-300">
                    Nginx 444:{" "}
                    {(stat.foramen_444_drops + stat.retis_444_drops).toLocaleString()}
                  </div>
                  <div className="text-rose-400">
                    ModSec:{" "}
                    {(stat.foramen_modsec_blocks + stat.retis_modsec_blocks).toLocaleString()}
                  </div>
                  <div className="text-amber-400 font-bold mt-0.5">
                    Hostile: {stat.malicious_ratio}%
                  </div>
                </div>

                {/* Stacked bar container */}
                <div
                  className="w-full rounded-t overflow-hidden flex flex-col-reverse transition-all duration-300 group-hover:brightness-125"
                  style={{ height: `${barHeightPct}%` }}
                >
                  <div className="w-full bg-cyan-400/80" style={{ height: `${legitPct}%` }} />
                  <div className="w-full bg-purple-500" style={{ height: `${wafPct}%` }} />
                  <div className="w-full bg-teal-400" style={{ height: `${dropPct}%` }} />
                  <div className="w-full bg-rose-500" style={{ height: `${modsecPct}%` }} />
                </div>
              </div>
            );
          })}
        </div>
      </div>

      {/* Row 2: Top Attacking Countries & Top Scanner Actors */}
      <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
        {/* Top Countries */}
        <div className="p-5 rounded-2xl bg-slate-900/80 border border-slate-800/80 backdrop-blur-md shadow-xl flex flex-col gap-3">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <Globe2 className="w-4 h-4 text-cyan-400" />
              <h3 className="text-sm font-bold uppercase tracking-wider text-slate-200">
                Top Threat Origin Origins (GeoIP)
              </h3>
            </div>
            {targetCountry && (
              <button
                onClick={() => onTargetCountryChange(null)}
                className="text-[11px] font-mono text-cyan-400 hover:underline"
              >
                Clear Focus
              </button>
            )}
          </div>

          <div className="flex flex-col gap-2 mt-1">
            {topCountries.slice(0, 6).map((c, i) => (
              <button
                key={c.country_code}
                onClick={() =>
                  onTargetCountryChange(
                    targetCountry === c.country_code ? null : c.country_code
                  )
                }
                className={`flex items-center justify-between p-2 rounded-lg text-xs font-mono transition-all text-left ${
                  targetCountry === c.country_code
                    ? "bg-cyan-950/60 border border-cyan-500/60"
                    : "bg-slate-800/40 hover:bg-slate-800/70 border border-transparent"
                }`}
              >
                <div className="flex items-center gap-2.5 flex-1 min-w-0 pr-3">
                  <span className="text-slate-400 w-4 font-bold text-right">{i + 1}.</span>
                  <span className="px-1.5 py-0.5 rounded bg-slate-800 text-slate-200 font-bold text-[10px]">
                    {c.country_code}
                  </span>
                  <span className="truncate text-slate-200 font-medium">
                    {c.country_name || "Unknown"}
                  </span>
                </div>

                <div className="flex items-center gap-3">
                  <span className="text-slate-400 text-[11px]">
                    {c.count.toLocaleString()} hits
                  </span>
                  <div className="w-16 h-1.5 bg-slate-800 rounded-full overflow-hidden hidden sm:block">
                    <div
                      className="h-full bg-cyan-400 rounded-full"
                      style={{ width: `${Math.min(100, c.percentage)}%` }}
                    />
                  </div>
                  <span className="text-cyan-300 font-bold w-10 text-right">
                    {c.percentage.toFixed(1)}%
                  </span>
                </div>
              </button>
            ))}
          </div>
        </div>

        {/* Top Scanner Actors */}
        <div className="p-5 rounded-2xl bg-slate-900/80 border border-slate-800/80 backdrop-blur-md shadow-xl flex flex-col gap-3">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-2">
              <Bot className="w-4 h-4 text-purple-400" />
              <h3 className="text-sm font-bold uppercase tracking-wider text-slate-200">
                Scanner & Automated Bot Actors
              </h3>
            </div>
            <span className="text-[11px] font-mono text-slate-400">rDNS & Signature</span>
          </div>

          <div className="flex flex-col gap-2 mt-1">
            {topActors.slice(0, 6).map((actor, i) => {
              const isResearch = actor.actor_type === "RESEARCH_SCANNER";
              return (
                <div
                  key={i}
                  className="flex items-center justify-between p-2 rounded-lg bg-slate-800/40 border border-slate-700/40 text-xs font-mono"
                >
                  <div className="flex items-center gap-2.5">
                    <span className="text-slate-400 w-4 font-bold text-right">{i + 1}.</span>
                    <span className="text-slate-100 font-semibold">{actor.actor_name}</span>
                    <span
                      className={`text-[9px] uppercase px-1.5 py-0.5 rounded font-bold border ${
                        isResearch
                          ? "bg-cyan-950/70 border-cyan-500/50 text-cyan-300"
                          : "bg-rose-950/70 border-rose-500/50 text-rose-300"
                      }`}
                    >
                      {isResearch ? "Research Scanner" : "Automated Bot"}
                    </span>
                  </div>
                  <span className="font-bold text-slate-300">
                    {actor.count.toLocaleString()} reqs
                  </span>
                </div>
              );
            })}
          </div>
        </div>
      </div>

      {/* Row 3: Observed Hostile IP Ranking & Recent Raw Block Log Table */}
      <div className="p-5 rounded-2xl bg-slate-900/80 border border-slate-800/80 backdrop-blur-md shadow-xl flex flex-col gap-4">
        <div className="flex items-center justify-between flex-wrap gap-2">
          <div className="flex items-center gap-2">
            <ShieldAlert className="w-4 h-4 text-rose-400" />
            <h3 className="text-sm font-bold uppercase tracking-wider text-slate-200">
              Recent Mitigated Hostile Requests (Plain Text IP / Strictly Non-Linked)
            </h3>
          </div>
          <span className="text-[10px] font-mono px-2 py-0.5 rounded bg-slate-800 text-slate-400">
            format-detection: none
          </span>
        </div>

        <div className="overflow-x-auto">
          <table className="w-full text-left text-xs font-mono">
            <thead>
              <tr className="border-b border-slate-800 text-slate-400 text-[11px] uppercase tracking-wider">
                <th className="py-2.5 px-3">Time</th>
                <th className="py-2.5 px-3">Defense Layer</th>
                <th className="py-2.5 px-3">Attacker IP</th>
                <th className="py-2.5 px-3">Location</th>
                <th className="py-2.5 px-3">ASN / Org</th>
                <th className="py-2.5 px-3">Threat Category</th>
                <th className="py-2.5 px-3">User-Agent</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-slate-800/60">
              {recentEvents.slice(0, 15).map((evt, idx) => {
                const isModSec = evt.layer === "MODSEC_BLOCK";
                const is444 = evt.layer === "HTTP_444";
                const isWaf = evt.layer === "AWS_WAF_SAMPLE";

                const badgeColor = isModSec
                  ? "bg-rose-950/60 border-rose-500/60 text-rose-300"
                  : is444
                  ? "bg-teal-950/60 border-teal-400/60 text-teal-300"
                  : isWaf
                  ? "bg-purple-950/60 border-purple-500/60 text-purple-300"
                  : "bg-slate-800 border-slate-700 text-slate-300";

                const layerName = isModSec
                  ? "ModSec CRS"
                  : is444
                  ? "HTTP 444 Drop"
                  : isWaf
                  ? "AWS WAF Edge"
                  : evt.layer;

                const timeStr = new Date(evt.observed_at).toLocaleTimeString("ja-JP", {
                  hour12: false,
                });

                return (
                  <tr key={idx} className="hover:bg-slate-800/30 transition-colors">
                    {/* Timestamp */}
                    <td className="py-2.5 px-3 whitespace-nowrap text-slate-400">
                      {timeStr}
                    </td>

                    {/* Defense Layer */}
                    <td className="py-2.5 px-3 whitespace-nowrap">
                      <span
                        className={`px-2 py-0.5 rounded text-[10px] font-bold border ${badgeColor}`}
                      >
                        {layerName}
                      </span>
                    </td>

                    {/* Attacker IP: Rendered strictly in <code>, NEVER as an anchor! */}
                    <td className="py-2.5 px-3 whitespace-nowrap">
                      <code className="text-slate-100 font-bold bg-slate-950 px-1.5 py-0.5 rounded border border-slate-800 select-text">
                        {evt.src_ip}
                      </code>
                    </td>

                    {/* Location */}
                    <td className="py-2.5 px-3 whitespace-nowrap text-slate-300">
                      <span className="font-bold text-slate-400 mr-1.5">
                        [{evt.country_code || "XX"}]
                      </span>
                      <span>{evt.city_name || evt.country_name || "Unknown"}</span>
                    </td>

                    {/* ASN & Org */}
                    <td className="py-2.5 px-3 whitespace-nowrap text-slate-400 max-w-[180px] truncate">
                      {evt.asn ? `AS${evt.asn} ` : ""}
                      {evt.as_org || "Unknown"}
                    </td>

                    {/* Threat Category */}
                    <td className="py-2.5 px-3 whitespace-nowrap">
                      <span className="text-rose-400 font-semibold">
                        {evt.threat_category || "ATTACK"}
                      </span>
                    </td>

                    {/* User Agent */}
                    <td
                      className="py-2.5 px-3 max-w-[220px] truncate text-slate-400"
                      title={evt.user_agent}
                    >
                      {evt.user_agent || "-"}
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      </div>
    </section>
  );
};
