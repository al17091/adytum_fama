import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "AdytumFama | Perimeter Observability & Threat Telemetry",
  description: "24H / 7D sliding window perimeter security telemetry, perimeter probes analytics, and multi-layer defense funnel.",
  other: {
    "format-detection": "telephone=no, date=no, address=no, email=no",
  },
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en">
      <head>
        <meta name="format-detection" content="telephone=no, date=no, address=no, email=no" />
      </head>
      <body className="bg-[#040711] text-slate-100 min-h-screen flex flex-col selection:bg-cyan-500/30 selection:text-cyan-200">
        {children}
      </body>
    </html>
  );
}
