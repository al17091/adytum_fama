"use client";

import React from "react";
import { Shield, Globe, Map, RefreshCw, Layers } from "lucide-react";
import { formatJST } from "../lib/time";

interface HeaderProps {
  window: "24h" | "7d";
  onWindowChange: (w: "24h" | "7d") => void;
  mapMode: "3d" | "2d";
  onMapModeChange: (m: "3d" | "2d") => void;
  generatedAt: string;
  loading: boolean;
  onRefresh: () => void;
}

export const Header: React.FC<HeaderProps> = ({
  window,
  onWindowChange,
  mapMode,
  onMapModeChange,
  generatedAt,
  loading,
  onRefresh,
}) => {
  const formattedTime = formatJST(generatedAt);

  return (
    <header className="border-b border-slate-800/80 bg-[#040711]/90 backdrop-blur-md sticky top-0 z-50 px-4 lg:px-8 py-3 flex flex-wrap items-center justify-between gap-4">
      {/* Brand & Subtitle */}
      <div className="flex items-center gap-3">
        <div className="p-2 rounded-lg bg-cyan-950/60 border border-cyan-500/30 text-cyan-400">
          <Shield className="w-5 h-5 animate-pulse" />
        </div>
        <div>
          <div className="flex items-center gap-2">
            <h1 className="text-lg font-bold tracking-wider text-slate-100 uppercase">
              Adytum<span className="text-cyan-400">Fama</span>
            </h1>
            <span className="text-[10px] uppercase font-mono px-2 py-0.5 rounded bg-slate-800/80 border border-slate-700 text-slate-300">
              Perimeter Telemetry
            </span>
          </div>
          <p className="text-xs text-slate-400 font-mono">
            Edge & Gateway Multi-Layer Defense Observability
          </p>
        </div>
      </div>

      {/* Center Controls: 24H vs 7D Toggle */}
      <div className="flex items-center bg-slate-900/90 border border-slate-800 p-1 rounded-lg">
        <button
          onClick={() => onWindowChange("24h")}
          className={`px-3 py-1 text-xs font-mono font-medium rounded transition-colors ${
            window === "24h"
              ? "bg-cyan-500/20 text-cyan-300 border border-cyan-500/40 shadow-sm"
              : "text-slate-400 hover:text-slate-200"
          }`}
        >
          LAST 24 HOURS
        </button>
        <button
          onClick={() => onWindowChange("7d")}
          className={`px-3 py-1 text-xs font-mono font-medium rounded transition-colors ${
            window === "7d"
              ? "bg-cyan-500/20 text-cyan-300 border border-cyan-500/40 shadow-sm"
              : "text-slate-400 hover:text-slate-200"
          }`}
        >
          LAST 7 DAYS
        </button>
      </div>

      {/* Right Controls: 3D/2D Toggle & Snapshot Badge */}
      <div className="flex items-center gap-3">
        {/* Map Mode Toggle */}
        <div className="flex items-center bg-slate-900/90 border border-slate-800 p-1 rounded-lg">
          <button
            onClick={() => onMapModeChange("3d")}
            title="3D Orthographic Cyber Globe"
            className={`p-1.5 rounded transition-colors flex items-center gap-1.5 text-xs font-mono ${
              mapMode === "3d"
                ? "bg-purple-500/20 text-purple-300 border border-purple-500/40"
                : "text-slate-400 hover:text-slate-200"
            }`}
          >
            <Globe className="w-3.5 h-3.5" />
            <span className="hidden sm:inline">3D GLOBE</span>
          </button>
          <button
            onClick={() => onMapModeChange("2d")}
            title="2D Planar Threat Map"
            className={`p-1.5 rounded transition-colors flex items-center gap-1.5 text-xs font-mono ${
              mapMode === "2d"
                ? "bg-purple-500/20 text-purple-300 border border-purple-500/40"
                : "text-slate-400 hover:text-slate-200"
            }`}
          >
            <Map className="w-3.5 h-3.5" />
            <span className="hidden sm:inline">2D MAP</span>
          </button>
        </div>

        {/* Log Explorer Link */}
        <a
          href="/guardinfo/events"
          className="flex items-center gap-1.5 px-3 py-1.5 rounded-lg bg-slate-900/90 border border-slate-800 hover:border-slate-700 text-slate-300 hover:text-cyan-300 text-xs font-mono transition-colors shadow-sm"
          title="Open Historical Log Explorer"
        >
          <Layers className="w-3.5 h-3.5 text-cyan-400" />
          <span className="hidden md:inline">Log Explorer</span>
        </a>

        {/* Snapshot Time Badge */}
        <div className="flex items-center gap-2 px-3 py-1 rounded-lg bg-slate-900/80 border border-slate-800 text-xs font-mono">
          <span className="w-2 h-2 rounded-full bg-emerald-400 animate-ping" />
          <span className="text-slate-400 hidden md:inline">Snapshot:</span>
          <span className="text-slate-200 font-semibold">{formattedTime}</span>
          <button
            onClick={onRefresh}
            disabled={loading}
            title="Refresh snapshot"
            className="text-slate-400 hover:text-cyan-400 ml-1 transition-colors disabled:opacity-50"
          >
            <RefreshCw className={`w-3.5 h-3.5 ${loading ? "animate-spin text-cyan-400" : ""}`} />
          </button>
        </div>
      </div>
    </header>
  );
};
