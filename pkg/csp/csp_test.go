package csp

import (
	"crypto/sha256"
	"encoding/base64"
	"strings"
	"testing"
	"testing/fstest"
)

func TestEngine_TransformHTML_StrictCSPv3(t *testing.T) {
	chunkContent := []byte(`console.log("chunk"); e.style.cssText="position:absolute";`)
	chunkSum := sha256.Sum256(chunkContent)
	expectedChunkSRI := "sha256-" + base64.StdEncoding.EncodeToString(chunkSum[:])

	cssTextSum := sha256.Sum256([]byte("position:absolute"))
	expectedStyleHash := "'sha256-" + base64.StdEncoding.EncodeToString(cssTextSum[:]) + "'"

	inlineScript := `(self.__next_f=self.__next_f||[]).push([0])`
	inlineSum := sha256.Sum256([]byte(inlineScript))
	expectedInlineHash := "'sha256-" + base64.StdEncoding.EncodeToString(inlineSum[:]) + "'"

	mockFS := fstest.MapFS{
		"_next/static/chunks/webpack-123.js": &fstest.MapFile{
			Data: chunkContent,
		},
	}

	engine, err := NewEngine(mockFS, "")
	if err != nil {
		t.Fatalf("NewEngine failed: %v", err)
	}

	rawHTML := []byte(`<!DOCTYPE html><html><head>` +
		`<link rel="stylesheet" href="/guardinfo/_next/static/css/app.css"/>` +
		`<link rel="preload" as="script" fetchPriority="low" href="/guardinfo/_next/static/chunks/webpack-123.js"/>` +
		`<script src="/guardinfo/_next/static/chunks/webpack-123.js" async=""></script>` +
		`</head><body>` +
		`<script>` + inlineScript + `</script>` +
		`</body></html>`)

	out1, hdr1, err := engine.TransformHTML(rawHTML)
	if err != nil {
		t.Fatalf("TransformHTML #1 failed: %v", err)
	}
	out2, hdr2, err := engine.TransformHTML(rawHTML)
	if err != nil {
		t.Fatalf("TransformHTML #2 failed: %v", err)
	}

	// 1. Per-request nonces must be unique
	if hdr1 == hdr2 || string(out1) == string(out2) {
		t.Fatalf("expected unique cryptographic nonces across requests, got identical outputs")
	}

	// 2. Zero unsafe-inline or unsafe-eval anywhere in CSP header
	if strings.Contains(hdr1, "'unsafe-inline'") {
		t.Errorf("CSP header must not contain 'unsafe-inline': %s", hdr1)
	}
	if strings.Contains(hdr1, "'unsafe-eval'") {
		t.Errorf("CSP header must not contain 'unsafe-eval': %s", hdr1)
	}

	// 3. Verify strict-dynamic, SRI hash, and inline script hash in CSP header
	for _, required := range []string{
		"default-src 'none'",
		"'strict-dynamic'",
		"'" + expectedChunkSRI + "'",
		expectedInlineHash,
		"script-src-attr 'none'",
		"style-src-elem 'self'",
		expectedStyleHash,
		"object-src 'none'",
		"base-uri 'none'",
		"frame-ancestors 'none'",
		"form-action 'none'",
	} {
		if !strings.Contains(hdr1, required) {
			t.Errorf("expected CSP header to contain %q, got: %s", required, hdr1)
		}
	}

	// 4. Verify HTML tags received nonce and integrity attributes
	htmlStr := string(out1)
	if !strings.Contains(htmlStr, `integrity="`+expectedChunkSRI+`"`) {
		t.Errorf("expected HTML to contain integrity=%q, got: %s", expectedChunkSRI, htmlStr)
	}
	if strings.Count(htmlStr, `nonce="`) != 3 {
		t.Errorf("expected 3 nonce attributes (1 preload link + 2 script tags), got %d in: %s", strings.Count(htmlStr, `nonce="`), htmlStr)
	}
}
