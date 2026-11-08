"use client";

import React, { useRef, useEffect, useState, useMemo } from "react";
import { PublicBlockEvent } from "../types";
import { WORLD_CONTINENTS } from "./worldData";

interface ThreatMapProps {
  events: PublicBlockEvent[];
  mapMode: "3d" | "2d";
  filterLayer: string | null;
  targetCountry: string | null;
  currentTimeIndex: number | null; // For time-lapse replay
}

// Tokyo Edge coordinates (RFC-style abstracted defense perimeter)
const TOKYO_LAT = 35.6895;
const TOKYO_LON = 139.6917;

export const ThreatMap: React.FC<ThreatMapProps> = ({
  events,
  mapMode,
  filterLayer,
  targetCountry,
  currentTimeIndex,
}) => {
  const canvasRef = useRef<HTMLCanvasElement | null>(null);
  const [rotation, setRotation] = useState({ yaw: -0.8, pitch: 0.35 });
  const [isDragging, setIsDragging] = useState(false);
  const dragStart = useRef({ x: 0, y: 0, yaw: 0, pitch: 0 });
  const pulseAnim = useRef(0);

  // Filter events based on active filters
  const visibleEvents = useMemo(() => {
    let filtered = events;
    if (filterLayer) {
      filtered = filtered.filter((e) => e.layer === filterLayer);
    }
    if (targetCountry) {
      filtered = filtered.filter((e) => e.country_code === targetCountry);
    }
    // Limit to max 40 arcs for high visual quality & performance
    return filtered.slice(0, 40);
  }, [events, filterLayer, targetCountry]);

  // Color mapping per layer
  const getLayerColor = (layer: string, alpha: number = 1): string => {
    switch (layer) {
      case "MODSEC_BLOCK":
        return `rgba(255, 42, 95, ${alpha})`; // Crimson
      case "HTTP_444":
        return `rgba(0, 245, 212, ${alpha})`; // Cyan
      case "AWS_WAF_SAMPLE":
        return `rgba(168, 85, 247, ${alpha})`; // Violet
      default:
        return `rgba(56, 189, 248, ${alpha})`; // Sky
    }
  };

  // Mouse / Touch handlers for 3D globe rotation
  const handleMouseDown = (e: React.MouseEvent<HTMLCanvasElement>) => {
    if (mapMode !== "3d") return;
    setIsDragging(true);
    dragStart.current = {
      x: e.clientX,
      y: e.clientY,
      yaw: rotation.yaw,
      pitch: rotation.pitch,
    };
  };

  const handleMouseMove = (e: React.MouseEvent<HTMLCanvasElement>) => {
    if (!isDragging || mapMode !== "3d") return;
    const dx = e.clientX - dragStart.current.x;
    const dy = e.clientY - dragStart.current.y;
    setRotation({
      yaw: dragStart.current.yaw + dx * 0.006,
      pitch: Math.max(-1.2, Math.min(1.2, dragStart.current.pitch + dy * 0.006)),
    });
  };

  const handleMouseUp = () => setIsDragging(false);

  // Render loop
  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;
    const ctx = canvas.getContext("2d");
    if (!ctx) return;

    let animId: number;

    const render = () => {
      pulseAnim.current += 0.015;
      if (pulseAnim.current > 1) pulseAnim.current = 0;

      // Auto-rotation when not dragging
      if (!isDragging && mapMode === "3d") {
        setRotation((prev) => ({
          ...prev,
          yaw: prev.yaw + 0.0012,
        }));
      }

      // Responsive canvas size
      const dpr = window.devicePixelRatio || 1;
      const rect = canvas.getBoundingClientRect();
      const W = rect.width;
      const H = rect.height;

      if (canvas.width !== W * dpr || canvas.height !== H * dpr) {
        canvas.width = W * dpr;
        canvas.height = H * dpr;
      }

      ctx.save();
      ctx.scale(dpr, dpr);
      ctx.clearRect(0, 0, W, H);

      if (mapMode === "3d") {
        render3DGlobe(ctx, W, H, rotation, pulseAnim.current, visibleEvents, getLayerColor);
      } else {
        render2DMap(ctx, W, H, pulseAnim.current, visibleEvents, getLayerColor);
      }

      ctx.restore();
      animId = requestAnimationFrame(render);
    };

    animId = requestAnimationFrame(render);
    return () => cancelAnimationFrame(animId);
  }, [mapMode, rotation, isDragging, visibleEvents]);

  return (
    <div className="relative w-full h-full min-h-[480px] lg:min-h-[580px] bg-gradient-to-b from-[#040711] via-[#070d1e] to-[#040711] rounded-2xl overflow-hidden border border-slate-800/80 shadow-2xl flex items-center justify-center cursor-grab active:cursor-grabbing">
      <canvas
        ref={canvasRef}
        onMouseDown={handleMouseDown}
        onMouseMove={handleMouseMove}
        onMouseUp={handleMouseUp}
        onMouseLeave={handleMouseUp}
        className="w-full h-full block"
      />

      {/* Edge Node Badge (Tokyo) */}
      <div className="absolute top-4 right-4 z-10 px-3 py-1.5 rounded-lg bg-slate-900/80 backdrop-blur-md border border-cyan-500/30 text-xs font-mono flex items-center gap-2">
        <span className="w-2 h-2 rounded-full bg-cyan-400 animate-ping" />
        <span className="text-slate-400">DEFENSE HUB:</span>
        <span className="text-cyan-300 font-bold">TOKYO EDGE [JP]</span>
      </div>

      {/* Active Arcs Count */}
      <div className="absolute bottom-4 left-4 z-10 px-3 py-1.5 rounded-lg bg-slate-900/80 backdrop-blur-md border border-slate-800 text-xs font-mono text-slate-400">
        Active Intercept Vectors:{" "}
        <span className="text-cyan-400 font-bold">{visibleEvents.length}</span>
      </div>
    </div>
  );
};

// 3D Spherical Projection Helper
function project3D(
  lat: number,
  lon: number,
  yaw: number,
  pitch: number,
  R: number,
  cx: number,
  cy: number,
  altRatio: number = 1
) {
  const phi = (lat * Math.PI) / 180;
  const theta = (lon * Math.PI) / 180 + yaw;

  // Sphere Cartesian (Y up, Z towards viewer)
  const x = R * altRatio * Math.cos(phi) * Math.sin(theta);
  const y0 = R * altRatio * Math.sin(phi);
  const z0 = R * altRatio * Math.cos(phi) * Math.cos(theta);

  // Pitch rotation around X axis
  const y = y0 * Math.cos(pitch) - z0 * Math.sin(pitch);
  const z = y0 * Math.sin(pitch) + z0 * Math.cos(pitch);

  return {
    x: cx + x,
    y: cy - y,
    z, // > 0 is front hemisphere
  };
}

// Render 3D Cyber Globe
function render3DGlobe(
  ctx: CanvasRenderingContext2D,
  W: number,
  H: number,
  rotation: { yaw: number; pitch: number },
  animProgress: number,
  events: PublicBlockEvent[],
  getLayerColor: (l: string, a?: number) => string
) {
  const cx = W / 2;
  const cy = H / 2;
  const R = Math.min(W, H) * 0.36;

  // 1. Atmosphere Radial Glow
  const glow = ctx.createRadialGradient(cx, cy, R * 0.85, cx, cy, R * 1.25);
  glow.addColorStop(0, "rgba(6, 182, 212, 0.12)");
  glow.addColorStop(0.5, "rgba(59, 130, 246, 0.05)");
  glow.addColorStop(1, "rgba(0, 0, 0, 0)");
  ctx.fillStyle = glow;
  ctx.beginPath();
  ctx.arc(cx, cy, R * 1.3, 0, Math.PI * 2);
  ctx.fill();

  // 2. Base Globe Sphere Shading
  const sphereGrad = ctx.createRadialGradient(
    cx - R * 0.3,
    cy - R * 0.3,
    R * 0.1,
    cx,
    cy,
    R
  );
  sphereGrad.addColorStop(0, "#0e1a38");
  sphereGrad.addColorStop(0.7, "#080f24");
  sphereGrad.addColorStop(1, "#030712");

  ctx.fillStyle = sphereGrad;
  ctx.beginPath();
  ctx.arc(cx, cy, R, 0, Math.PI * 2);
  ctx.fill();

  // Sphere Rim Outline
  ctx.strokeStyle = "rgba(6, 182, 212, 0.35)";
  ctx.lineWidth = 1.2;
  ctx.beginPath();
  ctx.arc(cx, cy, R, 0, Math.PI * 2);
  ctx.stroke();

  // 3. Latitude & Longitude Cyber Grid
  ctx.lineWidth = 0.5;
  ctx.strokeStyle = "rgba(56, 189, 248, 0.12)";

  // Latitude parallels (-60, -30, 0, 30, 60)
  for (let lat = -60; lat <= 60; lat += 30) {
    ctx.beginPath();
    let first = true;
    for (let lon = -180; lon <= 180; lon += 10) {
      const p = project3D(lat, lon, rotation.yaw, rotation.pitch, R, cx, cy);
      if (p.z > 0) {
        if (first) {
          ctx.moveTo(p.x, p.y);
          first = false;
        } else {
          ctx.lineTo(p.x, p.y);
        }
      } else {
        first = true;
      }
    }
    ctx.stroke();
  }

  // 4. Continent Coastlines & Landmass Contours
  ctx.strokeStyle = "rgba(56, 189, 248, 0.4)";
  ctx.lineWidth = 1.0;
  ctx.fillStyle = "rgba(15, 23, 42, 0.55)";

  for (const poly of WORLD_CONTINENTS) {
    ctx.beginPath();
    let started = false;
    for (let i = 0; i < poly.length; i++) {
      const [lat, lon] = poly[i];
      const pt = project3D(lat, lon, rotation.yaw, rotation.pitch, R, cx, cy);
      if (pt.z > 0) {
        if (!started) {
          ctx.moveTo(pt.x, pt.y);
          started = true;
        } else {
          ctx.lineTo(pt.x, pt.y);
        }
      } else {
        started = false;
      }
    }
    ctx.stroke();
  }

  // 5. Tokyo Target Hub Point
  const tokyo = project3D(TOKYO_LAT, TOKYO_LON, rotation.yaw, rotation.pitch, R, cx, cy);
  if (tokyo.z > 0) {
    // Concentric glowing rings
    ctx.strokeStyle = "rgba(0, 245, 212, 0.8)";
    ctx.lineWidth = 1.5;
    ctx.beginPath();
    ctx.arc(tokyo.x, tokyo.y, 4 + animProgress * 6, 0, Math.PI * 2);
    ctx.stroke();

    ctx.fillStyle = "#00f5d4";
    ctx.beginPath();
    ctx.arc(tokyo.x, tokyo.y, 3, 0, Math.PI * 2);
    ctx.fill();

    // Tokyo Label
    ctx.font = "10px monospace";
    ctx.fillStyle = "#00f5d4";
    ctx.fillText("TOKYO", tokyo.x + 8, tokyo.y + 3);
  }

  // 6. Attack Vectors & Great-Circle 3D Arcs to Tokyo
  for (const evt of events) {
    const lat = evt.latitude || 30;
    const lon = evt.longitude || 0;

    const origin = project3D(lat, lon, rotation.yaw, rotation.pitch, R, cx, cy);

    // Draw origin point if visible
    if (origin.z > 0) {
      ctx.fillStyle = getLayerColor(evt.layer, 0.8);
      ctx.beginPath();
      ctx.arc(origin.x, origin.y, 2.5, 0, Math.PI * 2);
      ctx.fill();
    }

    // Great circle arc interpolation
    const steps = 30;
    ctx.beginPath();
    let arcDrawn = false;

    for (let i = 0; i <= steps; i++) {
      const t = i / steps;
      // Interpolate lat / lon
      const currLat = lat + (TOKYO_LAT - lat) * t;
      const currLon = lon + (TOKYO_LON - lon) * t;
      // Arch height peaking at midpoint
      const altitude = 1 + Math.sin(t * Math.PI) * 0.22;

      const pt = project3D(currLat, currLon, rotation.yaw, rotation.pitch, R, cx, cy, altitude);
      if (pt.z > -R * 0.2) {
        if (!arcDrawn) {
          ctx.moveTo(pt.x, pt.y);
          arcDrawn = true;
        } else {
          ctx.lineTo(pt.x, pt.y);
        }
      }
    }

    ctx.strokeStyle = getLayerColor(evt.layer, 0.45);
    ctx.lineWidth = 1.2;
    ctx.stroke();

    // Traveling packet pulse along arc
    const pulseT = (animProgress + (evt.asn % 10) * 0.1) % 1;
    const pulseLat = lat + (TOKYO_LAT - lat) * pulseT;
    const pulseLon = lon + (TOKYO_LON - lon) * pulseT;
    const pulseAlt = 1 + Math.sin(pulseT * Math.PI) * 0.22;
    const pulsePt = project3D(pulseLat, pulseLon, rotation.yaw, rotation.pitch, R, cx, cy, pulseAlt);

    if (pulsePt.z > 0) {
      ctx.fillStyle = getLayerColor(evt.layer, 0.95);
      ctx.shadowColor = getLayerColor(evt.layer, 0.8);
      ctx.shadowBlur = 8;
      ctx.beginPath();
      ctx.arc(pulsePt.x, pulsePt.y, 3, 0, Math.PI * 2);
      ctx.fill();
      ctx.shadowBlur = 0;
    }
  }
}

// Render 2D Planar Map
function render2DMap(
  ctx: CanvasRenderingContext2D,
  W: number,
  H: number,
  animProgress: number,
  events: PublicBlockEvent[],
  getLayerColor: (l: string, a?: number) => string
) {
  // Equirectangular mapping
  const toX = (lon: number) => ((lon + 180) / 360) * W;
  const toY = (lat: number) => ((90 - lat) / 180) * H;

  // 1. Grid
  ctx.strokeStyle = "rgba(56, 189, 248, 0.08)";
  ctx.lineWidth = 0.5;
  for (let x = 0; x < W; x += W / 12) {
    ctx.beginPath();
    ctx.moveTo(x, 0);
    ctx.lineTo(x, H);
    ctx.stroke();
  }
  for (let y = 0; y < H; y += H / 6) {
    ctx.beginPath();
    ctx.moveTo(0, y);
    ctx.lineTo(W, y);
    ctx.stroke();
  }

  // 2. Continents Landmass Polygons & Coastlines
  ctx.fillStyle = "rgba(15, 23, 42, 0.85)";
  ctx.strokeStyle = "rgba(56, 189, 248, 0.35)";
  ctx.lineWidth = 1.2;

  for (const poly of WORLD_CONTINENTS) {
    ctx.beginPath();
    let first = true;
    for (let i = 0; i < poly.length; i++) {
      const [lat, lon] = poly[i];
      const x = toX(lon);
      const y = toY(lat);
      if (first) {
        ctx.moveTo(x, y);
        first = false;
      } else {
        ctx.lineTo(x, y);
      }
    }
    ctx.closePath();
    ctx.fill();
    ctx.stroke();
  }

  // 3. Tokyo Hub
  const tx = toX(TOKYO_LON);
  const ty = toY(TOKYO_LAT);

  ctx.strokeStyle = "#00f5d4";
  ctx.lineWidth = 1.5;
  ctx.beginPath();
  ctx.arc(tx, ty, 5 + animProgress * 8, 0, Math.PI * 2);
  ctx.stroke();

  ctx.fillStyle = "#00f5d4";
  ctx.beginPath();
  ctx.arc(tx, ty, 3.5, 0, Math.PI * 2);
  ctx.fill();

  ctx.font = "10px monospace";
  ctx.fillText("TOKYO EDGE", tx + 8, ty + 3);

  // 4. Arcs to Tokyo
  for (const evt of events) {
    const ox = toX(evt.longitude || 0);
    const oy = toY(evt.latitude || 30);

    // Origin dot
    ctx.fillStyle = getLayerColor(evt.layer, 0.8);
    ctx.beginPath();
    ctx.arc(ox, oy, 2.5, 0, Math.PI * 2);
    ctx.fill();

    // Curved Bezier Arc
    const midX = (ox + tx) / 2;
    const midY = Math.min(oy, ty) - 50;

    ctx.strokeStyle = getLayerColor(evt.layer, 0.35);
    ctx.lineWidth = 1.2;
    ctx.beginPath();
    ctx.moveTo(ox, oy);
    ctx.quadraticCurveTo(midX, midY, tx, ty);
    ctx.stroke();

    // Moving pulse
    const t = (animProgress + (evt.asn % 10) * 0.1) % 1;
    const px = (1 - t) * (1 - t) * ox + 2 * (1 - t) * t * midX + t * t * tx;
    const py = (1 - t) * (1 - t) * oy + 2 * (1 - t) * t * midY + t * t * ty;

    ctx.fillStyle = getLayerColor(evt.layer, 0.95);
    ctx.shadowColor = getLayerColor(evt.layer, 0.8);
    ctx.shadowBlur = 6;
    ctx.beginPath();
    ctx.arc(px, py, 2.8, 0, Math.PI * 2);
    ctx.fill();
    ctx.shadowBlur = 0;
  }
}
