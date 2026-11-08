/**
 * Utility for formatting timestamps to a uniform canonical display:
 * `yyyy-mm-dd hh:mm:ss JST` (e.g. `2026-11-08 13:09:30 JST`).
 */
export function formatJST(input?: string | Date | number | null): string {
  if (!input) return "--";
  const d = typeof input === "string" || typeof input === "number" ? new Date(input) : input;
  if (isNaN(d.getTime())) return "--";

  const formatter = new Intl.DateTimeFormat("ja-JP", {
    timeZone: "Asia/Tokyo",
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
    hour12: false,
  });

  const parts = formatter.formatToParts(d);
  const get = (type: string) => parts.find((p) => p.type === type)?.value || "00";
  return `${get("year")}-${get("month")}-${get("day")} ${get("hour")}:${get("minute")}:${get("second")} JST`;
}

