import React from "react";
import { ShieldAlert, ArrowLeft } from "lucide-react";

export default function NotFound() {
  return (
    <div className="min-h-screen bg-[#040711] text-slate-100 flex flex-col items-center justify-center p-6 font-mono">
      <div className="max-w-md w-full p-8 rounded-2xl bg-slate-900/80 border border-slate-800/80 backdrop-blur-md shadow-2xl flex flex-col items-center text-center gap-4">
        <div className="p-3 rounded-xl bg-rose-950/60 border border-rose-500/40 text-rose-400">
          <ShieldAlert className="w-8 h-8" />
        </div>
        <div className="flex flex-col gap-1">
          <span className="text-xs uppercase tracking-widest text-rose-400 font-bold">
            HTTP 404 • Perimeter Route Not Found
          </span>
          <h1 className="text-xl font-bold text-slate-100">
            Telemetry Endpoint Unavailable
          </h1>
        </div>
        <p className="text-xs text-slate-400 leading-relaxed">
          The requested perimeter telemetry path does not exist or has been rotated out of the active hot-tier window.
        </p>
        <a
          href="/guardinfo/"
          className="mt-2 inline-flex items-center gap-2 px-4 py-2 rounded-lg bg-cyan-500/20 hover:bg-cyan-500/30 border border-cyan-500/40 text-cyan-300 text-xs font-bold transition-colors"
        >
          <ArrowLeft className="w-4 h-4" />
          <span>Return to Telemetry Dashboard</span>
        </a>
      </div>
    </div>
  );
}
