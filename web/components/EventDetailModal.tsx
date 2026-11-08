"use client";

import React, { useState, useEffect } from "react";
import {
  X,
  Copy,
  Check,
  ShieldAlert,
  Globe2,
  Terminal,
  Fingerprint,
  FileCode,
  Network,
} from "lucide-react";
import { PublicBlockEvent } from "../types";

interface EventDetailModalProps {
  event: PublicBlockEvent | null;
  onClose: () => void;
}

export const EventDetailModal: React.FC<EventDetailModalProps> = ({
  event,
  onClose,
}) => {
  const [copiedKey, setCopiedKey] = useState<string | null>(null);

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === "Escape") {
        onClose();
      }
    };
    if (event) {
      window.addEventListener("keydown", handleKeyDown);
    }
    return () => window.removeEventListener("keydown", handleKeyDown);
  }, [event, onClose]);

  if (!event) return null;

  const handleCopy = (key: string, text: string) => {
    navigator.clipboard.writeText(text);
    setCopiedKey(key);
    setTimeout(() => {
      setCopiedKey(null);
    }, 2000);
  };

  const isModSec = event.layer === "MODSEC_BLOCK" || event.layer === "MODSEC_DETECT";
  const is444 = event.layer === "HTTP_444";
  const is403 = event.layer === "HTTP_403";
  const isSuspiciousUrl = event.layer === "SUSPICIOUS_URL";
  const isAnomaly = event.layer === "ANOMALY_SPIKE";
  const isWaf = event.layer === "AWS_WAF_SAMPLE";

  const layerName = event.layer === "MODSEC_BLOCK"
    ? "ModSecurity Block"
    : event.layer === "MODSEC_DETECT"
    ? "ModSecurity Detect"
    : is444
    ? "Nginx HTTP 444 Drop"
    : is403
    ? "HTTP 403 Forbidden"
    : isSuspiciousUrl
    ? "Suspicious URL Rule"
    : isAnomaly
    ? "Anomaly Spike"
    : isWaf
    ? "AWS WAF Edge Sample"
    : event.layer;

  const badgeColor = event.layer === "MODSEC_BLOCK"
    ? "bg-rose-950/70 border-rose-500/60 text-rose-300"
    : event.layer === "MODSEC_DETECT"
    ? "bg-indigo-950/70 border-indigo-400/60 text-indigo-300"
    : is444
    ? "bg-teal-950/70 border-teal-400/60 text-teal-300"
    : is403
    ? "bg-orange-950/70 border-orange-400/60 text-orange-300"
    : isSuspiciousUrl
    ? "bg-red-950/70 border-red-500/60 text-red-300"
    : isAnomaly
    ? "bg-amber-950/70 border-amber-500/60 text-amber-300"
    : isWaf
    ? "bg-purple-950/70 border-purple-500/60 text-purple-300"
    : "bg-slate-800 border-slate-700 text-slate-300";

  const utcTime = new Date(event.observed_at).toISOString();
  const localTime = new Date(event.observed_at).toLocaleString("ja-JP", {
    timeZoneName: "short",
  });

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-slate-950/80 backdrop-blur-sm animate-fade-in"
      onClick={onClose}
    >
      <div
        className="w-full max-w-3xl max-h-[90vh] bg-[#070b18] border border-slate-700/80 rounded-2xl shadow-2xl flex flex-col overflow-hidden text-slate-100"
        onClick={(e) => e.stopPropagation()}
      >
        {/* Modal Header */}
        <div className="px-6 py-4 border-b border-slate-800 flex items-center justify-between gap-4 bg-slate-900/50">
          <div className="flex items-center gap-3">
            <div className="p-2 rounded-lg bg-cyan-950/60 border border-cyan-500/30 text-cyan-400">
              <ShieldAlert className="w-5 h-5" />
            </div>
            <div>
              <div className="flex items-center gap-2">
                <h3 className="text-base font-bold text-slate-100 tracking-wide">
                  Perimeter Intercept Inspector
                </h3>
                <span className={`px-2 py-0.5 rounded text-[10px] font-bold border ${badgeColor}`}>
                  {layerName}
                </span>
                {event.classification === "MALICIOUS" ? (
                  <span className="px-2 py-0.5 rounded text-[10px] font-bold bg-rose-950/70 border border-rose-500/60 text-rose-300">
                    MALICIOUS
                  </span>
                ) : event.classification === "SUSPICIOUS" ? (
                  <span className="px-2 py-0.5 rounded text-[10px] font-bold bg-amber-950/70 border border-amber-500/60 text-amber-300">
                    SUSPICIOUS
                  </span>
                ) : (
                  <span className="px-2 py-0.5 rounded text-[10px] font-mono text-slate-400 bg-slate-900 border border-slate-800">
                    -
                  </span>
                )}
              </div>
              <p className="text-xs text-slate-400 font-mono mt-0.5">
                Observed: {localTime} ({utcTime})
              </p>
            </div>
          </div>
          <button
            onClick={onClose}
            className="p-1.5 rounded-lg text-slate-400 hover:text-slate-100 hover:bg-slate-800 transition-colors"
            title="Close (Esc)"
          >
            <X className="w-5 h-5" />
          </button>
        </div>

        {/* Modal Body */}
        <div className="p-6 overflow-y-auto flex flex-col gap-5 text-xs font-mono">
          {/* Key Identifiers Grid */}
          <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-4 gap-3">
            {/* IP Address */}
            <div className="p-3 rounded-xl bg-slate-900/70 border border-slate-800/80 flex flex-col gap-1">
              <span className="text-[10px] text-slate-400 uppercase tracking-wider">Observed IP</span>
              <div className="flex items-center justify-between gap-1">
                <code className="text-sm font-bold text-slate-100 select-all">
                  {event.src_ip}
                </code>
                <button
                  onClick={() => handleCopy("ip", event.src_ip)}
                  className="p-1 text-slate-400 hover:text-cyan-400 transition-colors"
                  title="Copy IP"
                >
                  {copiedKey === "ip" ? <Check className="w-3.5 h-3.5 text-emerald-400" /> : <Copy className="w-3.5 h-3.5" />}
                </button>
              </div>
            </div>

            {/* Threat Vector */}
            <div className="p-3 rounded-xl bg-slate-900/70 border border-slate-800/80 flex flex-col gap-1">
              <span className="text-[10px] text-slate-400 uppercase tracking-wider">Threat Vector</span>
              <span className="text-xs font-bold text-cyan-300 truncate">
                {event.threat_category || "RAW_IP_RECON"}
              </span>
            </div>

            {/* Geolocation */}
            <div className="p-3 rounded-xl bg-slate-900/70 border border-slate-800/80 flex flex-col gap-1">
              <span className="text-[10px] text-slate-400 uppercase tracking-wider">Origin Location</span>
              <span className="text-xs font-semibold text-slate-200 truncate">
                [{event.country_code || "XX"}] {event.city_name ? `${event.city_name}, ` : ""}{event.country_name || "Unknown"}
              </span>
            </div>

            {/* Network / ASN */}
            <div className="p-3 rounded-xl bg-slate-900/70 border border-slate-800/80 flex flex-col gap-1">
              <span className="text-[10px] text-slate-400 uppercase tracking-wider">Autonomous System</span>
              <span className="text-xs font-semibold text-slate-200 truncate" title={`AS${event.asn || 0} ${event.as_org || "Unknown"}`}>
                AS{event.asn || 0}
              </span>
            </div>
          </div>

          {/* Target Path (Full Request URI) */}
          <div className="p-4 rounded-xl bg-slate-900/70 border border-slate-800/80 flex flex-col gap-2">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2 text-slate-300 font-bold">
                <FileCode className="w-4 h-4 text-cyan-400" />
                <span>Target Request URI (Full String)</span>
              </div>
              <div className="flex items-center gap-2">
                <span className="px-2 py-0.5 rounded text-[10px] font-bold bg-slate-800 text-slate-200 border border-slate-700">
                  {event.http_method || "GET"}
                </span>
                <button
                  onClick={() => handleCopy("uri", event.request_uri || "/")}
                  className="flex items-center gap-1 text-[11px] text-slate-400 hover:text-cyan-400 px-2 py-0.5 rounded bg-slate-800/80 border border-slate-700 transition-colors"
                >
                  {copiedKey === "uri" ? (
                    <>
                      <Check className="w-3 h-3 text-emerald-400" />
                      <span className="text-emerald-400">Copied</span>
                    </>
                  ) : (
                    <>
                      <Copy className="w-3 h-3" />
                      <span>Copy URI</span>
                    </>
                  )}
                </button>
              </div>
            </div>
            <pre className="p-3 rounded-lg bg-slate-950 border border-slate-800 text-cyan-300 text-xs font-mono break-all whitespace-pre-wrap select-all overflow-x-auto leading-relaxed">
              {event.request_uri || "/"}
            </pre>
          </div>

          {/* User-Agent (Full String) */}
          <div className="p-4 rounded-xl bg-slate-900/70 border border-slate-800/80 flex flex-col gap-2">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2 text-slate-300 font-bold">
                <Terminal className="w-4 h-4 text-cyan-400" />
                <span>User-Agent Header (Full String)</span>
              </div>
              <button
                onClick={() => handleCopy("ua", event.user_agent || "-")}
                className="flex items-center gap-1 text-[11px] text-slate-400 hover:text-cyan-400 px-2 py-0.5 rounded bg-slate-800/80 border border-slate-700 transition-colors"
              >
                {copiedKey === "ua" ? (
                  <>
                    <Check className="w-3 h-3 text-emerald-400" />
                    <span className="text-emerald-400">Copied</span>
                  </>
                ) : (
                  <>
                    <Copy className="w-3 h-3" />
                    <span>Copy User-Agent</span>
                  </>
                )}
              </button>
            </div>
            <pre className="p-3 rounded-lg bg-slate-950 border border-slate-800 text-slate-300 text-xs font-mono break-all whitespace-pre-wrap select-all overflow-x-auto leading-relaxed">
              {event.user_agent || "-"}
            </pre>
          </div>

          {/* Origin Network & Organization Details */}
          <div className="p-4 rounded-xl bg-slate-900/70 border border-slate-800/80 flex flex-col gap-3">
            <div className="flex items-center gap-2 text-slate-300 font-bold">
              <Network className="w-4 h-4 text-cyan-400" />
              <span>ASN & Organization Details</span>
            </div>
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <div>
                <span className="text-[10px] text-slate-400 uppercase">AS Number</span>
                <p className="text-slate-100 font-semibold mt-0.5">
                  {event.asn ? `AS${event.asn}` : "None / Unassigned"}
                </p>
              </div>
              <div>
                <span className="text-[10px] text-slate-400 uppercase">Organization (Full Name)</span>
                <p className="text-slate-100 font-semibold mt-0.5 break-words">
                  {event.as_org || "Unknown"}
                </p>
              </div>
              <div>
                <span className="text-[10px] text-slate-400 uppercase">Scanner / Bot Classification</span>
                <p className="text-slate-100 font-semibold mt-0.5">
                  {event.actor_type && event.actor_type !== "UNKNOWN"
                    ? `${event.actor_type} (${event.actor_name || "Unknown"})`
                    : "Unclassified Probe"}
                </p>
              </div>
              <div>
                <span className="text-[10px] text-slate-400 uppercase">Geo Coordinates</span>
                <p className="text-slate-100 font-semibold mt-0.5">
                  {event.latitude && event.longitude
                    ? `${event.latitude.toFixed(4)}, ${event.longitude.toFixed(4)}`
                    : "Unavailable"}
                </p>
              </div>
            </div>
          </div>

          {/* TLS Client Fingerprints (JA4 / JA3) */}
          <div className="p-4 rounded-xl bg-slate-900/70 border border-slate-800/80 flex flex-col gap-3">
            <div className="flex items-center gap-2 text-slate-300 font-bold">
              <Fingerprint className="w-4 h-4 text-cyan-400" />
              <span>TLS Client Fingerprints (CloudFront Origin Telemetry)</span>
            </div>
            <div className="grid grid-cols-1 sm:grid-cols-2 gap-4">
              <div className="flex flex-col gap-1">
                <div className="flex items-center justify-between">
                  <span className="text-[10px] text-slate-400 uppercase">JA4 Fingerprint</span>
                  {event.ja4_fingerprint && (
                    <button
                      onClick={() => handleCopy("ja4", event.ja4_fingerprint!)}
                      className="text-[10px] text-slate-400 hover:text-cyan-400"
                    >
                      {copiedKey === "ja4" ? "Copied" : "Copy"}
                    </button>
                  )}
                </div>
                <code className="p-2 rounded bg-slate-950 border border-slate-800 text-teal-300 text-xs break-all select-all">
                  {event.ja4_fingerprint || "Not captured (direct gateway access / non-CloudFront)"}
                </code>
              </div>
              <div className="flex flex-col gap-1">
                <div className="flex items-center justify-between">
                  <span className="text-[10px] text-slate-400 uppercase">JA3 Fingerprint (MD5)</span>
                  {event.ja3_fingerprint && (
                    <button
                      onClick={() => handleCopy("ja3", event.ja3_fingerprint!)}
                      className="text-[10px] text-slate-400 hover:text-cyan-400"
                    >
                      {copiedKey === "ja3" ? "Copied" : "Copy"}
                    </button>
                  )}
                </div>
                <code className="p-2 rounded bg-slate-950 border border-slate-800 text-slate-300 text-xs break-all select-all">
                  {event.ja3_fingerprint || "Not captured (direct gateway access / non-CloudFront)"}
                </code>
              </div>
            </div>
          </div>
        </div>

        {/* Modal Footer */}
        <div className="px-6 py-3 border-t border-slate-800 flex items-center justify-between bg-slate-900/50">
          <span className="text-[10px] text-slate-400">
            Observed IP is rendered strictly as plain text (format-detection disabled).
          </span>
          <button
            onClick={onClose}
            className="px-4 py-1.5 rounded-lg bg-slate-800 hover:bg-slate-700 text-xs font-semibold text-slate-200 transition-colors"
          >
            Close
          </button>
        </div>
      </div>
    </div>
  );
};

