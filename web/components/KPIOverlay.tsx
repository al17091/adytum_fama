"use client";

import React from "react";
import { Activity, ShieldCheck, ShieldAlert, Cpu, Radio, Zap } from "lucide-react";
import { LayerBreakdown } from "../types";

interface KPIOverlayProps {
  totalRequests: number;
  legitimateRequests: number;
  mitigatedRequests: number;
  maliciousRatio: number;
  layerBreakdown: LayerBreakdown;
  filterLayer: string | null;
  onFilterLayerChange: (layer: string | null) => void;
}

export const KPIOverlay: React.FC<KPIOverlayProps> = ({
  totalRequests,
  legitimateRequests,
  mitigatedRequests,
  maliciousRatio,
  layerBreakdown,
  filterLayer,
  onFilterLayerChange,
}) => {
  const legitPercent =
    totalRequests > 0 ? ((legitimateRequests / totalRequests) * 100).toFixed(1) : "0.0";

  return (
    <div className="flex flex-col gap-3 max-w-sm w-full">
      {/* Total Observed Requests */}
      <div className="p-3.5 rounded-xl bg-slate-900/85 backdrop-blur-md border border-slate-800/90 shadow-xl">
        <div className="flex items-center justify-between text-slate-400 text-xs font-mono mb-1">
          <span className="flex items-center gap-1.5 uppercase tracking-wider">
            <Activity className="w-3.5 h-3.5 text-cyan-400" />
            Total Perimeter Requests
          </span>
          <span className="text-[11px] text-cyan-400/80 font-bold">100% INGRESS</span>
        </div>
        <div className="text-2xl font-bold font-mono tracking-tight text-slate-100">
          {totalRequests.toLocaleString()}{" "}
          <span className="text-xs text-slate-400 font-normal">reqs</span>
        </div>
        {/* Progress bar */}
        <div className="w-full h-1.5 bg-slate-800 rounded-full mt-2 overflow-hidden flex">
          <div
            className="h-full bg-cyan-400 transition-all duration-500"
            style={{ width: `${legitPercent}%` }}
            title={`Legitimate: ${legitPercent}%`}
          />
          <div
            className="h-full bg-rose-500 transition-all duration-500"
            style={{ width: `${maliciousRatio}%` }}
            title={`Hostile: ${maliciousRatio}%`}
          />
        </div>
      </div>

      {/* Legitimate vs Hostile Dual Cards */}
      <div className="grid grid-cols-2 gap-2.5">
        {/* Legitimate Traffic */}
        <div className="p-3 rounded-xl bg-slate-900/80 backdrop-blur-md border border-slate-800/80">
          <div className="flex items-center gap-1.5 text-[11px] text-slate-400 font-mono mb-1">
            <ShieldCheck className="w-3.5 h-3.5 text-cyan-400" />
            <span>Legitimate</span>
          </div>
          <div className="text-lg font-bold font-mono text-cyan-300">
            {legitimateRequests.toLocaleString()}
          </div>
          <div className="text-[10px] font-mono text-slate-400 mt-0.5">
            {legitPercent}% passed
          </div>
        </div>

        {/* Mitigated Malicious Traffic */}
        <div className="p-3 rounded-xl bg-slate-900/80 backdrop-blur-md border border-rose-950/60 shadow-inner">
          <div className="flex items-center gap-1.5 text-[11px] text-rose-400 font-mono mb-1">
            <ShieldAlert className="w-3.5 h-3.5 text-rose-400" />
            <span>Mitigated</span>
          </div>
          <div className="text-lg font-bold font-mono text-rose-400 glow-modsec">
            {mitigatedRequests.toLocaleString()}
          </div>
          <div className="text-[10px] font-mono text-rose-300/80 font-semibold mt-0.5">
            {maliciousRatio}% hostile
          </div>
        </div>
      </div>

      {/* Defense Layer Filter / Breakdown Pills */}
      <div className="p-3 rounded-xl bg-slate-900/85 backdrop-blur-md border border-slate-800/90 flex flex-col gap-2">
        <div className="flex items-center justify-between text-[11px] font-mono text-slate-400">
          <span className="uppercase tracking-wider">Multi-Layer Interception</span>
          {filterLayer && (
            <button
              onClick={() => onFilterLayerChange(null)}
              className="text-cyan-400 hover:underline text-[10px]"
            >
              Reset Filter
            </button>
          )}
        </div>

        <div className="flex flex-col gap-1.5">
          {/* AWS WAF (Edge) */}
          <button
            onClick={() =>
              onFilterLayerChange(filterLayer === "AWS_WAF_SAMPLE" ? null : "AWS_WAF_SAMPLE")
            }
            className={`flex items-center justify-between px-2.5 py-1.5 rounded-lg text-xs font-mono transition-all border ${
              filterLayer === "AWS_WAF_SAMPLE"
                ? "bg-purple-950/70 border-purple-500/80 text-purple-200"
                : "bg-slate-800/40 border-slate-700/50 text-slate-300 hover:border-purple-500/40"
            }`}
          >
            <div className="flex items-center gap-2">
              <span className="w-2 h-2 rounded-full bg-purple-500 shadow-[0_0_8px_#a855f7]" />
              <Zap className="w-3 h-3 text-purple-400" />
              <span>AWS WAF (Edge)</span>
            </div>
            <span className="font-bold text-purple-300">
              {layerBreakdown.aws_waf_blocked.toLocaleString()}
            </span>
          </button>

          {/* Nginx HTTP 444 (Gateway Drop) */}
          <button
            onClick={() =>
              onFilterLayerChange(filterLayer === "HTTP_444" ? null : "HTTP_444")
            }
            className={`flex items-center justify-between px-2.5 py-1.5 rounded-lg text-xs font-mono transition-all border ${
              filterLayer === "HTTP_444"
                ? "bg-cyan-950/70 border-cyan-500/80 text-cyan-200"
                : "bg-slate-800/40 border-slate-700/50 text-slate-300 hover:border-cyan-500/40"
            }`}
          >
            <div className="flex items-center gap-2">
              <span className="w-2 h-2 rounded-full bg-cyan-400 shadow-[0_0_8px_#00f5d4]" />
              <Radio className="w-3 h-3 text-cyan-400" />
              <span>Nginx 444 (Silent Drop)</span>
            </div>
            <span className="font-bold text-cyan-300">
              {(layerBreakdown.foramen_444_drops + layerBreakdown.retis_444_drops).toLocaleString()}
            </span>
          </button>

          {/* ModSecurity CRS (L7 WAF) */}
          <button
            onClick={() =>
              onFilterLayerChange(filterLayer === "MODSEC_BLOCK" ? null : "MODSEC_BLOCK")
            }
            className={`flex items-center justify-between px-2.5 py-1.5 rounded-lg text-xs font-mono transition-all border ${
              filterLayer === "MODSEC_BLOCK"
                ? "bg-rose-950/70 border-rose-500/80 text-rose-200"
                : "bg-slate-800/40 border-slate-700/50 text-slate-300 hover:border-rose-500/40"
            }`}
          >
            <div className="flex items-center gap-2">
              <span className="w-2 h-2 rounded-full bg-rose-500 shadow-[0_0_8px_#ff2a5f]" />
              <Cpu className="w-3 h-3 text-rose-400" />
              <span>ModSecurity CRS (L7 WAF)</span>
            </div>
            <span className="font-bold text-rose-300">
              {(layerBreakdown.foramen_modsec_blocks + layerBreakdown.retis_modsec_blocks).toLocaleString()}
            </span>
          </button>
        </div>
      </div>
    </div>
  );
};
