"use client";

import React, { useState, useEffect, useCallback } from "react";
import {
  ShieldAlert,
  ArrowLeft,
  Search,
  Filter,
  Calendar,
  RefreshCw,
  Eye,
  ChevronLeft,
  ChevronRight,
  Shield,
  Layers,
} from "lucide-react";
import { PublicBlockEvent } from "../../types";
import { EventDetailModal } from "../../components/EventDetailModal";

interface QueryEventsResponse {
  total: number;
  events: PublicBlockEvent[];
  limit: number;
  offset: number;
}

export default function EventsExplorerPage() {
  const [windowRange, setWindowRange] = useState<"24h" | "3d" | "7d">("7d");
  const [layer, setLayer] = useState<string>("ALL");
  const [classification, setClassification] = useState<string>("ALL");
  const [searchQuery, setSearchQuery] = useState<string>("");
  const [debouncedSearch, setDebouncedSearch] = useState<string>("");

  const [page, setPage] = useState<number>(1);
  const pageSize = 50;

  const [loading, setLoading] = useState<boolean>(true);
  const [error, setError] = useState<string | null>(null);
  const [total, setTotal] = useState<number>(0);
  const [events, setEvents] = useState<PublicBlockEvent[]>([]);
  const [selectedEvent, setSelectedEvent] = useState<PublicBlockEvent | null>(null);

  // Debounce search input
  useEffect(() => {
    const handler = setTimeout(() => {
      setDebouncedSearch(searchQuery);
      setPage(1); // reset to page 1 on new search
    }, 350);
    return () => clearTimeout(handler);
  }, [searchQuery]);

  const fetchEvents = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const offset = (page - 1) * pageSize;
      const params = new URLSearchParams({
        window: windowRange,
        limit: pageSize.toString(),
        offset: offset.toString(),
      });
      if (layer !== "ALL") params.set("layer", layer);
      if (classification !== "ALL") params.set("classification", classification);
      if (debouncedSearch.trim()) params.set("q", debouncedSearch.trim());

      const res = await fetch(`/guardinfo/api/events?${params.toString()}`);
      if (!res.ok) {
        throw new Error(`Failed loading events (HTTP ${res.status})`);
      }
      const data: QueryEventsResponse = await res.json();
      setEvents(data.events || []);
      setTotal(data.total || 0);
    } catch (err: unknown) {
      console.error(err);
      setError(err instanceof Error ? err.message : "Error querying event telemetry");
    } finally {
      setLoading(false);
    }
  }, [windowRange, layer, classification, debouncedSearch, page]);

  useEffect(() => {
    fetchEvents();
  }, [fetchEvents]);

  const totalPages = Math.max(1, Math.ceil(total / pageSize));

  return (
    <div className="flex flex-col min-h-screen bg-[#040711] text-slate-100 selection:bg-cyan-500/30">
      {/* Top Header */}
      <header className="border-b border-slate-800/80 bg-[#040711]/90 backdrop-blur-md sticky top-0 z-40 px-4 lg:px-8 py-3.5 flex flex-wrap items-center justify-between gap-4">
        <div className="flex items-center gap-4">
          <a
            href="/guardinfo/"
            className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-slate-900 border border-slate-800 hover:border-slate-700 text-slate-300 hover:text-cyan-300 text-xs font-mono transition-colors"
          >
            <ArrowLeft className="w-3.5 h-3.5" />
            <span>Dashboard</span>
          </a>

          <div className="h-4 w-px bg-slate-800" />

          <div className="flex items-center gap-2.5">
            <div className="p-1.5 rounded-lg bg-cyan-950/60 border border-cyan-500/30 text-cyan-400">
              <Shield className="w-4 h-4" />
            </div>
            <div>
              <h1 className="text-base font-bold tracking-wider text-slate-100 uppercase">
                Perimeter Event Log Explorer
              </h1>
              <p className="text-[11px] text-slate-400 font-mono">
                Historical Telemetry & Intercept Analysis (Hot Tier / 7-Day Storage)
              </p>
            </div>
          </div>
        </div>

        {/* Refresh button */}
        <button
          onClick={fetchEvents}
          disabled={loading}
          className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-slate-900 hover:bg-slate-800 border border-slate-800 text-xs font-mono text-slate-300 transition-colors disabled:opacity-50"
        >
          <RefreshCw className={`w-3.5 h-3.5 ${loading ? "animate-spin text-cyan-400" : ""}`} />
          <span>Refresh</span>
        </button>
      </header>

      {/* Main Container */}
      <main className="flex-1 max-w-7xl w-full mx-auto px-4 lg:px-8 py-6 flex flex-col gap-6">
        {/* Filters & Search Control Card */}
        <div className="p-5 rounded-2xl bg-slate-900/80 border border-slate-800/80 backdrop-blur-md shadow-xl flex flex-col gap-4">
          <div className="flex flex-wrap items-center justify-between gap-4">
            {/* Time Window Selector */}
            <div className="flex items-center gap-2">
              <span className="text-xs font-mono text-slate-400 uppercase tracking-wider flex items-center gap-1.5">
                <Calendar className="w-3.5 h-3.5 text-cyan-400" /> Time Window:
              </span>
              <div className="flex items-center bg-slate-950 border border-slate-800 p-0.5 rounded-lg">
                {(["24h", "3d", "7d"] as const).map((w) => (
                  <button
                    key={w}
                    onClick={() => {
                      setWindowRange(w);
                      setPage(1);
                    }}
                    className={`px-3 py-1 text-xs font-mono font-medium rounded-md transition-colors ${
                      windowRange === w
                        ? "bg-cyan-500/20 text-cyan-300 border border-cyan-500/40 shadow-sm"
                        : "text-slate-400 hover:text-slate-200"
                    }`}
                  >
                    {w === "24h" ? "24 Hours" : w === "3d" ? "3 Days" : "7 Days"}
                  </button>
                ))}
              </div>
            </div>

            {/* Defense Layer Filter */}
            <div className="flex items-center gap-2">
              <span className="text-xs font-mono text-slate-400 uppercase tracking-wider flex items-center gap-1.5">
                <Layers className="w-3.5 h-3.5 text-purple-400" /> Defense Layer:
              </span>
              <select
                value={layer}
                onChange={(e) => {
                  setLayer(e.target.value);
                  setPage(1);
                }}
                className="bg-slate-950 border border-slate-800 text-slate-200 text-xs font-mono px-3 py-1.5 rounded-lg focus:outline-none focus:border-cyan-500"
              >
                <option value="ALL">All Layers</option>
                <option value="HTTP_444">Nginx HTTP 444 Drop</option>
                <option value="MODSEC_BLOCK">ModSecurity Block</option>
                <option value="SUSPICIOUS_URL">Suspicious URL Rule</option>
                <option value="ANOMALY_SPIKE">Anomaly Spike</option>
                <option value="HTTP_403">HTTP 403 Forbidden</option>
                <option value="MODSEC_DETECT">ModSecurity Detect</option>
              </select>
            </div>

            {/* Classification Filter */}
            <div className="flex items-center gap-2">
              <span className="text-xs font-mono text-slate-400 uppercase tracking-wider flex items-center gap-1.5">
                <Filter className="w-3.5 h-3.5 text-rose-400" /> Classification:
              </span>
              <select
                value={classification}
                onChange={(e) => {
                  setClassification(e.target.value);
                  setPage(1);
                }}
                className="bg-slate-950 border border-slate-800 text-slate-200 text-xs font-mono px-3 py-1.5 rounded-lg focus:outline-none focus:border-cyan-500"
              >
                <option value="ALL">All Classifications</option>
                <option value="MALICIOUS">MALICIOUS</option>
                <option value="SUSPICIOUS">SUSPICIOUS</option>
                <option value="-">- (Recon / Unclassified)</option>
              </select>
            </div>
          </div>

          {/* Search bar */}
          <div className="relative w-full">
            <Search className="w-4 h-4 text-slate-400 absolute left-3.5 top-1/2 -translate-y-1/2" />
            <input
              type="text"
              placeholder="Filter by IP address, request path (/admin, .env), User-Agent, or Autonomous System (Contabo, Amazon)..."
              value={searchQuery}
              onChange={(e) => setSearchQuery(e.target.value)}
              className="w-full bg-slate-950/90 border border-slate-800 rounded-xl pl-10 pr-4 py-2 text-xs font-mono text-slate-200 placeholder:text-slate-500 focus:outline-none focus:border-cyan-500/80 transition-colors"
            />
            {searchQuery && (
              <button
                onClick={() => setSearchQuery("")}
                className="absolute right-3 top-1/2 -translate-y-1/2 text-xs text-slate-400 hover:text-slate-200 font-mono"
              >
                Clear
              </button>
            )}
          </div>
        </div>

        {/* Results Info & Pagination Top */}
        <div className="flex items-center justify-between text-xs font-mono text-slate-400 px-1">
          <div>
            Showing{" "}
            <span className="text-cyan-400 font-bold">
              {total > 0 ? (page - 1) * pageSize + 1 : 0}
            </span>{" "}
            -{" "}
            <span className="text-cyan-400 font-bold">
              {Math.min(page * pageSize, total)}
            </span>{" "}
            of <span className="text-slate-100 font-bold">{total.toLocaleString()}</span> intercepted events
          </div>

          <div className="flex items-center gap-2">
            <span>
              Page {page} of {totalPages}
            </span>
            <button
              onClick={() => setPage((p) => Math.max(1, p - 1))}
              disabled={page <= 1 || loading}
              className="p-1 rounded bg-slate-900 border border-slate-800 text-slate-300 hover:text-cyan-300 disabled:opacity-40 disabled:cursor-not-allowed"
            >
              <ChevronLeft className="w-4 h-4" />
            </button>
            <button
              onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
              disabled={page >= totalPages || loading}
              className="p-1 rounded bg-slate-900 border border-slate-800 text-slate-300 hover:text-cyan-300 disabled:opacity-40 disabled:cursor-not-allowed"
            >
              <ChevronRight className="w-4 h-4" />
            </button>
          </div>
        </div>

        {/* Event Logs Table Card */}
        <div className="p-5 rounded-2xl bg-slate-900/80 border border-slate-800/80 backdrop-blur-md shadow-xl flex flex-col gap-4">
          {error ? (
            <div className="p-6 text-center text-rose-400 font-mono text-xs">
              {error}
            </div>
          ) : loading && events.length === 0 ? (
            <div className="py-16 text-center font-mono text-xs text-slate-400 flex flex-col items-center justify-center gap-2">
              <RefreshCw className="w-6 h-6 animate-spin text-cyan-400" />
              <span>Querying historical telemetry...</span>
            </div>
          ) : events.length === 0 ? (
            <div className="py-16 text-center font-mono text-xs text-slate-400">
              No matching perimeter events found for the selected criteria.
            </div>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full text-left text-xs font-mono">
                <thead>
                  <tr className="border-b border-slate-800 text-slate-400 text-[11px] uppercase tracking-wider">
                    <th className="py-2.5 px-3">Time</th>
                    <th className="py-2.5 px-3">Defense Layer</th>
                    <th className="py-2.5 px-3">Observed IP</th>
                    <th className="py-2.5 px-3">Our Classification</th>
                    <th className="py-2.5 px-3">Location</th>
                    <th className="py-2.5 px-3">Vector</th>
                    <th className="py-2.5 px-3">User-Agent</th>
                    <th className="py-2.5 px-3 text-right">Details</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-800/60">
                  {events.map((evt, idx) => {
                    const isModSec = evt.layer === "MODSEC_BLOCK" || evt.layer === "MODSEC_DETECT";
                    const is444 = evt.layer === "HTTP_444";
                    const is403 = evt.layer === "HTTP_403";
                    const isSuspiciousUrl = evt.layer === "SUSPICIOUS_URL";
                    const isAnomaly = evt.layer === "ANOMALY_SPIKE";
                    const isWaf = evt.layer === "AWS_WAF_SAMPLE";

                    const badgeColor = evt.layer === "MODSEC_BLOCK"
                      ? "bg-rose-950/60 border-rose-500/60 text-rose-300"
                      : evt.layer === "MODSEC_DETECT"
                      ? "bg-indigo-950/60 border-indigo-400/60 text-indigo-300"
                      : is444
                      ? "bg-teal-950/60 border-teal-400/60 text-teal-300"
                      : is403
                      ? "bg-orange-950/60 border-orange-400/60 text-orange-300"
                      : isSuspiciousUrl
                      ? "bg-red-950/60 border-red-500/60 text-red-300"
                      : isAnomaly
                      ? "bg-amber-950/60 border-amber-500/60 text-amber-300"
                      : isWaf
                      ? "bg-purple-950/60 border-purple-500/60 text-purple-300"
                      : "bg-slate-800 border-slate-700 text-slate-300";

                    const layerName = evt.layer === "MODSEC_BLOCK"
                      ? "ModSec Block"
                      : evt.layer === "MODSEC_DETECT"
                      ? "ModSec Detect"
                      : is444
                      ? "HTTP 444 Drop"
                      : is403
                      ? "HTTP 403 Deny"
                      : isSuspiciousUrl
                      ? "Suspicious URL"
                      : isAnomaly
                      ? "Anomaly Spike"
                      : isWaf
                      ? "AWS WAF Edge"
                      : evt.layer;

                    const timeStr = new Date(evt.observed_at).toLocaleString("ja-JP", {
                      month: "2-digit",
                      day: "2-digit",
                      hour: "2-digit",
                      minute: "2-digit",
                      second: "2-digit",
                      hour12: false,
                    });

                    return (
                      <tr
                        key={idx}
                        onClick={() => setSelectedEvent(evt)}
                        className="hover:bg-slate-800/40 cursor-pointer transition-colors group"
                      >
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

                        {/* Observed IP */}
                        <td className="py-2.5 px-3 whitespace-nowrap">
                          <code className="text-slate-100 font-bold bg-slate-950 px-1.5 py-0.5 rounded border border-slate-800 select-text">
                            {evt.src_ip}
                          </code>
                        </td>

                        {/* Our Classification */}
                        <td className="py-2.5 px-3 whitespace-nowrap">
                          <div className="flex flex-col gap-0.5 items-start">
                            {evt.classification === "MALICIOUS" ? (
                              <span className="px-2 py-0.5 rounded text-[10px] font-bold bg-rose-950/70 border border-rose-500/60 text-rose-300">
                                MALICIOUS
                              </span>
                            ) : evt.classification === "SUSPICIOUS" ? (
                              <span className="px-2 py-0.5 rounded text-[10px] font-bold bg-amber-950/70 border border-amber-500/60 text-amber-300">
                                SUSPICIOUS
                              </span>
                            ) : (
                              <span className="text-slate-400 font-mono text-xs px-2 py-0.5 rounded bg-slate-900 border border-slate-800">
                                -
                              </span>
                            )}
                          </div>
                        </td>

                        {/* Location */}
                        <td className="py-2.5 px-3 whitespace-nowrap text-slate-300">
                          <span className="font-bold text-slate-400 mr-1.5">
                            [{evt.country_code || "XX"}]
                          </span>
                          <span>{evt.city_name || evt.country_name || "Unknown"}</span>
                        </td>

                        {/* Vector */}
                        <td className="py-2.5 px-3 whitespace-nowrap">
                          <div className="flex flex-col gap-0.5 items-start">
                            <span className="text-[11px] font-mono text-cyan-400/90 font-medium">
                              {evt.threat_category || "RAW_IP_RECON"}
                            </span>
                            {evt.ja4_fingerprint && (
                              <span
                                className="text-[9px] font-mono px-1 py-0.2 rounded bg-slate-800 text-teal-300 border border-teal-500/30"
                                title={`JA4: ${evt.ja4_fingerprint}`}
                              >
                                JA4: {evt.ja4_fingerprint.slice(0, 10)}...
                              </span>
                            )}
                          </div>
                        </td>

                        {/* User Agent */}
                        <td
                          className="py-2.5 px-3 max-w-[280px] truncate text-slate-400 font-mono text-xs"
                          title={evt.user_agent}
                        >
                          {evt.user_agent || "-"}
                        </td>

                        {/* Action: Details */}
                        <td className="py-2.5 px-3 text-right whitespace-nowrap">
                          <button
                            onClick={(e) => {
                              e.stopPropagation();
                              setSelectedEvent(evt);
                            }}
                            className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-lg bg-slate-800/80 hover:bg-cyan-950/70 border border-slate-700/80 hover:border-cyan-500/50 text-slate-300 hover:text-cyan-300 text-[11px] font-mono transition-colors shadow-sm"
                            title="View Full Event Details"
                          >
                            <Eye className="w-3.5 h-3.5 text-cyan-400" />
                            <span>Inspect</span>
                          </button>
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          )}
        </div>

        {/* Pagination Bottom */}
        {totalPages > 1 && (
          <div className="flex items-center justify-between text-xs font-mono text-slate-400 px-1 pb-8">
            <div>
              Page <span className="text-slate-100 font-bold">{page}</span> of{" "}
              <span className="text-slate-100 font-bold">{totalPages}</span>
            </div>

            <div className="flex items-center gap-2">
              <button
                onClick={() => setPage((p) => Math.max(1, p - 1))}
                disabled={page <= 1 || loading}
                className="flex items-center gap-1 px-3 py-1.5 rounded-lg bg-slate-900 border border-slate-800 text-slate-300 hover:text-cyan-300 disabled:opacity-40 disabled:cursor-not-allowed"
              >
                <ChevronLeft className="w-4 h-4" />
                <span>Previous</span>
              </button>
              <button
                onClick={() => setPage((p) => Math.min(totalPages, p + 1))}
                disabled={page >= totalPages || loading}
                className="flex items-center gap-1 px-3 py-1.5 rounded-lg bg-slate-900 border border-slate-800 text-slate-300 hover:text-cyan-300 disabled:opacity-40 disabled:cursor-not-allowed"
              >
                <span>Next</span>
                <ChevronRight className="w-4 h-4" />
              </button>
            </div>
          </div>
        )}
      </main>

      {/* Event Details Modal */}
      <EventDetailModal
        event={selectedEvent}
        onClose={() => setSelectedEvent(null)}
      />
    </div>
  );
}
