"use client";

import React from "react";
import { Shield, Layers, ExternalLink } from "lucide-react";

export const Footer: React.FC = () => {
  return (
    <footer className="border-t border-slate-800/80 bg-[#02050c] mt-auto py-10 px-4 lg:px-8 text-xs font-mono text-slate-400">
      <div className="max-w-7xl mx-auto flex flex-col gap-8">
        {/* Defense Architecture Summary */}
        <div className="grid grid-cols-1 md:grid-cols-3 gap-6 p-5 rounded-2xl bg-slate-900/40 border border-slate-800/60">
          <div>
            <div className="flex items-center gap-2 text-purple-400 font-bold mb-1.5">
              <span className="w-2 h-2 rounded-full bg-purple-500 shadow-[0_0_8px_#a855f7]" />
              Layer 1: AWS Edge Perimeter
            </div>
            <p className="text-[11px] text-slate-400 leading-relaxed">
              Global CloudFront edge locations and AWS WAF (WWW-WebACL) mitigate layer-3/4 DDoS
              spikes, known IP reputation threats, and rate abuse before reaching the Tokyo gateway.
            </p>
          </div>

          <div>
            <div className="flex items-center gap-2 text-cyan-400 font-bold mb-1.5">
              <span className="w-2 h-2 rounded-full bg-cyan-400 shadow-[0_0_8px_#00f5d4]" />
              Layer 2: Gateway Nginx Silent Drop
            </div>
            <p className="text-[11px] text-slate-400 leading-relaxed">
              Dedicated reverse proxies (foramen & retis) immediately terminate unauthorized direct-IP
              reconnaissance and invalid Host headers with TCP resets via Nginx HTTP 444.
            </p>
          </div>

          <div>
            <div className="flex items-center gap-2 text-rose-400 font-bold mb-1.5">
              <span className="w-2 h-2 rounded-full bg-rose-500 shadow-[0_0_8px_#ff2a5f]" />
              Layer 3: ModSecurity L7 WAF
            </div>
            <p className="text-[11px] text-slate-400 leading-relaxed">
              OWASP Core Rule Set (CRS v4) inspects incoming payloads for SQL injection, XSS, remote code execution,
              and dotfile reconnaissance, dropping hostile exploits with HTTP 403.
            </p>
          </div>
        </div>

        {/* Disclaimer & Attributions */}
        <div className="flex flex-col md:flex-row items-center justify-between gap-4 border-t border-slate-800/60 pt-6 text-[11px]">
          <div className="flex items-center gap-2 text-slate-400">
            <Shield className="w-4 h-4 text-cyan-400" />
            <span>
              AdytumFama Perimeter Telemetry Engine • MIT License • Copyright &copy; 2026 Koki Higashi
            </span>
          </div>

          {/* MaxMind Attribution */}
          <div className="text-slate-400 text-center md:text-right">
            <span>This product includes GeoLite2 data created by MaxMind, available from </span>
            <a
              href="https://www.maxmind.com"
              target="_blank"
              rel="noopener noreferrer"
              className="text-cyan-400 hover:underline inline-flex items-center gap-0.5"
            >
              maxmind.com <ExternalLink className="w-3 h-3" />
            </a>
          </div>
        </div>
      </div>
    </footer>
  );
};
