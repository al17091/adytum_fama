package web

import "embed"

// StaticAssets embeds pre-built Next.js export files
//
//go:embed all:out
var StaticAssets embed.FS
