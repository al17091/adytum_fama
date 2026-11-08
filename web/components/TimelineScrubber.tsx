"use client";

import React, { useEffect, useState } from "react";
import { Play, Pause, RotateCcw, FastForward } from "lucide-react";
import { HourlyTrafficStats } from "../types";
import { formatJST } from "../lib/time";

interface TimelineScrubberProps {
  hourlySeries: HourlyTrafficStats[];
  currentIndex: number | null;
  onIndexChange: (index: number | null) => void;
}

export const TimelineScrubber: React.FC<TimelineScrubberProps> = ({
  hourlySeries,
  currentIndex,
  onIndexChange,
}) => {
  const [isPlaying, setIsPlaying] = useState(false);
  const [speed, setSpeed] = useState<number>(1); // 1x, 2x, 4x

  const totalSteps = hourlySeries.length;
  const activeIdx = currentIndex !== null ? currentIndex : totalSteps - 1;
  const activeIdxRef = React.useRef(activeIdx);
  activeIdxRef.current = activeIdx;

  // Auto-advance loop when playing
  useEffect(() => {
    if (!isPlaying || totalSteps === 0) return;

    const intervalMs = Math.max(200, 1000 / speed);
    const timer = setInterval(() => {
      const next = (activeIdxRef.current + 1) % totalSteps;
      onIndexChange(next);
    }, intervalMs);

    return () => clearInterval(timer);
  }, [isPlaying, speed, totalSteps, onIndexChange]);

  const currentStat = hourlySeries[activeIdx];
  const formattedBucket = formatJST(currentStat?.bucket_time);

  return (
    <div className="w-full bg-slate-900/90 backdrop-blur-md border border-slate-800/90 rounded-xl p-3 shadow-xl flex flex-col gap-2">
      <div className="flex items-center justify-between gap-4">
        {/* Play / Pause / Reset controls */}
        <div className="flex items-center gap-2">
          <button
            onClick={() => setIsPlaying(!isPlaying)}
            className="p-1.5 rounded-lg bg-cyan-500/20 text-cyan-300 border border-cyan-500/40 hover:bg-cyan-500/30 transition-colors"
            title={isPlaying ? "Pause replay" : "Play time-lapse replay"}
          >
            {isPlaying ? <Pause className="w-4 h-4" /> : <Play className="w-4 h-4" />}
          </button>

          <button
            onClick={() => {
              setIsPlaying(false);
              onIndexChange(null);
            }}
            className="p-1.5 rounded-lg bg-slate-800 text-slate-300 hover:text-slate-100 transition-colors"
            title="Reset to latest hour"
          >
            <RotateCcw className="w-4 h-4" />
          </button>

          {/* Speed Toggle */}
          <button
            onClick={() => setSpeed((s) => (s === 1 ? 2 : s === 2 ? 4 : 1))}
            className="px-2 py-1 rounded bg-slate-800 text-slate-300 hover:text-cyan-400 text-xs font-mono transition-colors flex items-center gap-1"
            title="Replay speed"
          >
            <FastForward className="w-3 h-3" />
            <span>{speed}x</span>
          </button>
        </div>

        {/* Current Replay Timestamp */}
        <div className="text-xs font-mono flex items-center gap-2">
          <span className="text-slate-400 hidden sm:inline">Time-Lapse Replay:</span>
          <span className="px-2 py-0.5 rounded bg-cyan-950/60 border border-cyan-500/30 text-cyan-300 font-bold">
            {formattedBucket}
          </span>
          {currentIndex === null && (
            <span className="text-[10px] uppercase font-mono text-slate-400">(Latest)</span>
          )}
        </div>
      </div>

      {/* Slider Bar */}
      <div className="flex items-center gap-3">
        <input
          type="range"
          min={0}
          max={Math.max(0, totalSteps - 1)}
          value={activeIdx}
          onChange={(e) => {
            setIsPlaying(false);
            onIndexChange(parseInt(e.target.value, 10));
          }}
          className="w-full accent-cyan-400 h-1.5 bg-slate-800 rounded-lg cursor-pointer"
        />
        <span className="text-[11px] font-mono text-slate-400 whitespace-nowrap">
          {activeIdx + 1} / {totalSteps}h
        </span>
      </div>
    </div>
  );
};
