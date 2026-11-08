"use client";

import React, { useState, useEffect, useCallback } from "react";
import { Header } from "../components/Header";
import { ThreatMap } from "../components/ThreatMap";
import { KPIOverlay } from "../components/KPIOverlay";
import { TimelineScrubber } from "../components/TimelineScrubber";
import { AnalyticsGrid } from "../components/AnalyticsGrid";
import { Footer } from "../components/Footer";
import { SnapshotPayload } from "../types";
import { getFallbackSnapshot } from "../lib/mock";

export default function GuardInfoPage() {
  const [windowType, setWindowType] = useState<"24h" | "7d">("24h");
  const [mapMode, setMapMode] = useState<"3d" | "2d">("3d");
  const [filterLayer, setFilterLayer] = useState<string | null>(null);
  const [targetCountry, setTargetCountry] = useState<string | null>(null);
  const [replayIndex, setReplayIndex] = useState<number | null>(null);

  const [snapshot, setSnapshot] = useState<SnapshotPayload>(() =>
    getFallbackSnapshot("24h")
  );
  const [loading, setLoading] = useState(false);

  // Fetch pre-aggregated snapshot JSON
  const fetchSnapshot = useCallback(async (win: "24h" | "7d") => {
    setLoading(true);
    try {
      // In production behind Traefik & CloudFront, the API is at /guardinfo/api/snapshot
      const res = await fetch(`/guardinfo/api/snapshot?window=${win}`, {
        headers: { Accept: "application/json" },
        cache: "default",
      });
      if (!res.ok) {
        throw new Error(`API returned HTTP ${res.status}`);
      }
      const data: SnapshotPayload = await res.json();
      setSnapshot(data);
    } catch (err) {
      console.warn("Falling back to local snapshot due to API error/dev mode:", err);
      setSnapshot(getFallbackSnapshot(win));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchSnapshot(windowType);
  }, [windowType, fetchSnapshot]);

  // Derived events when scrubbed in replay mode
  const activeEvents = React.useMemo(() => {
    if (replayIndex === null || !snapshot.hourly_series[replayIndex]) {
      return snapshot.recent_events;
    }
    // In replay mode, select events close to that bucket time
    const targetBucket = new Date(
      snapshot.hourly_series[replayIndex].bucket_time
    ).getTime();
    const matching = snapshot.recent_events.filter((e) => {
      const et = new Date(e.observed_at).getTime();
      return Math.abs(et - targetBucket) <= 3600 * 1000 * 2;
    });
    return matching.length > 0 ? matching : snapshot.recent_events.slice(0, 5);
  }, [replayIndex, snapshot]);

  return (
    <div className="flex flex-col min-h-screen bg-[#040711] text-slate-100 selection:bg-cyan-500/30">
      {/* Top Navigation & Status */}
      <Header
        window={windowType}
        onWindowChange={(w) => {
          setWindowType(w);
          setReplayIndex(null);
        }}
        mapMode={mapMode}
        onMapModeChange={setMapMode}
        generatedAt={snapshot.generated_at}
        loading={loading}
        onRefresh={() => fetchSnapshot(windowType)}
      />

      {/* Main Content Area */}
      <main className="flex-1 flex flex-col">
        {/* Section 1: Hero Threat Map Stage (约 65-75vh) */}
        <section className="relative w-full max-w-7xl mx-auto px-4 lg:px-8 pt-4 pb-2 flex flex-col gap-3">
          <div className="relative w-full min-h-[500px] lg:min-h-[600px] flex items-center justify-center">
            {/* 3D/2D Threat Map Canvas */}
            <ThreatMap
              events={activeEvents}
              mapMode={mapMode}
              filterLayer={filterLayer}
              targetCountry={targetCountry}
              currentTimeIndex={replayIndex}
            />

            {/* Left Floating KPI Overlay */}
            <div className="absolute top-4 left-4 z-20 pointer-events-auto hidden md:block">
              <KPIOverlay
                totalRequests={snapshot.total_requests}
                legitimateRequests={snapshot.legitimate_requests}
                mitigatedRequests={snapshot.mitigated_requests}
                maliciousRatio={snapshot.malicious_ratio}
                layerBreakdown={snapshot.layer_breakdown}
                filterLayer={filterLayer}
                onFilterLayerChange={setFilterLayer}
              />
            </div>
          </div>

          {/* Mobile KPI block */}
          <div className="block md:hidden">
            <KPIOverlay
              totalRequests={snapshot.total_requests}
              legitimateRequests={snapshot.legitimate_requests}
              mitigatedRequests={snapshot.mitigated_requests}
              maliciousRatio={snapshot.malicious_ratio}
              layerBreakdown={snapshot.layer_breakdown}
              filterLayer={filterLayer}
              onFilterLayerChange={setFilterLayer}
            />
          </div>

          {/* Bottom Stage: Timeline Scrubber */}
          <TimelineScrubber
            hourlySeries={snapshot.hourly_series}
            currentIndex={replayIndex}
            onIndexChange={setReplayIndex}
          />
        </section>

        {/* Section 2: Analytics & Log Grid */}
        <AnalyticsGrid
          hourlySeries={snapshot.hourly_series}
          topCountries={snapshot.top_countries}
          topActors={snapshot.top_actors}
          recentEvents={snapshot.recent_events}
          targetCountry={targetCountry}
          onTargetCountryChange={setTargetCountry}
        />
      </main>

      {/* Footer */}
      <Footer />
    </div>
  );
}
