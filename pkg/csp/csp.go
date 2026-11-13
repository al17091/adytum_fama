package csp

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var (
	// Matches <link ...> tags in HTML
	linkTagRe = regexp.MustCompile(`(?is)<link\b[^>]*>`)
	// Matches <script ...>...</script> tags in HTML
	scriptTagRe = regexp.MustCompile(`(?is)<script\b([^>]*)>(.*?)</script>`)
	// Extracts href="..." or src="..." attribute values
	hrefAttrRe = regexp.MustCompile(`\bhref="([^"]+)"`)
	srcAttrRe  = regexp.MustCompile(`\bsrc="([^"]+)"`)
	// Extracts style.cssText="..." literals from compiled Next.js bundles (e.g. AppRouterAnnouncer)
	cssTextLiteralRe = regexp.MustCompile(`\.style\.cssText="([^"]+)"`)
)

// Engine manages CSP Level 3 asset hashing, Subresource Integrity (SRI),
// and per-request cryptographic nonce generation.
type Engine struct {
	assetIntegrity map[string]string // Maps URL path (e.g. "/guardinfo/_next/static/...") -> "sha256-<base64>"
	styleAttrHash  []string          // Sorted "'sha256-<base64>'" expressions for framework cssText attributes
}

// NewEngine initializes the CSP v3 engine by scanning static assets from staticDir (if set) or staticFS.
func NewEngine(staticFS fs.FS, staticDir string) (*Engine, error) {
	e := &Engine{
		assetIntegrity: make(map[string]string),
	}

	styleHashSet := make(map[string]struct{})

	walkFn := func(relPath string, data []byte) {
		relPath = filepath.ToSlash(relPath)
		relPath = strings.TrimPrefix(relPath, "/")

		if strings.HasSuffix(relPath, ".js") || strings.HasSuffix(relPath, ".css") {
			sum := sha256.Sum256(data)
			b64 := base64.StdEncoding.EncodeToString(sum[:])
			sri := "sha256-" + b64
			e.assetIntegrity["/guardinfo/"+relPath] = sri
			e.assetIntegrity["/"+relPath] = sri
		}

		if strings.HasSuffix(relPath, ".js") {
			matches := cssTextLiteralRe.FindAllSubmatch(data, -1)
			for _, m := range matches {
				if len(m) > 1 && len(m[1]) > 0 {
					sum := sha256.Sum256(m[1])
					b64 := base64.StdEncoding.EncodeToString(sum[:])
					expr := "'sha256-" + b64 + "'"
					styleHashSet[expr] = struct{}{}
				}
			}
		}
	}

	if staticDir != "" {
		err := filepath.WalkDir(staticDir, func(fullPath string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return err
			}
			rel, relErr := filepath.Rel(staticDir, fullPath)
			if relErr != nil {
				return nil
			}
			data, readErr := os.ReadFile(fullPath)
			if readErr == nil {
				walkFn(rel, data)
			}
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("walking staticDir for CSP hashes: %w", err)
		}
	} else if staticFS != nil {
		err := fs.WalkDir(staticFS, ".", func(p string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return err
			}
			data, readErr := fs.ReadFile(staticFS, p)
			if readErr == nil {
				walkFn(p, data)
			}
			return nil
		})
		if err != nil {
			return nil, fmt.Errorf("walking staticFS for CSP hashes: %w", err)
		}
	}

	for expr := range styleHashSet {
		e.styleAttrHash = append(e.styleAttrHash, expr)
	}
	sort.Strings(e.styleAttrHash)

	return e, nil
}

// GenerateNonce creates a fresh 128-bit cryptographic random base64 nonce.
func GenerateNonce() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("generating cryptographic nonce: %w", err)
	}
	return base64.StdEncoding.EncodeToString(b[:]), nil
}

// TransformHTML injects a fresh cryptographic nonce and Subresource Integrity (SRI) hashes
// into all <script> and <link rel="preload" as="script"> tags, and returns the transformed
// HTML bytes along with the corresponding strict CSP Level 3 header value.
func (e *Engine) TransformHTML(rawHTML []byte) ([]byte, string, error) {
	nonce, err := GenerateNonce()
	if err != nil {
		return nil, "", err
	}

	var scriptHashes []string
	seenHashes := make(map[string]struct{})
	addHash := func(expr string) {
		if expr == "" {
			return
		}
		if _, exists := seenHashes[expr]; !exists {
			seenHashes[expr] = struct{}{}
			scriptHashes = append(scriptHashes, expr)
		}
	}

	// 1. Inject nonce and integrity into <link rel="preload" as="script" ...> tags
	htmlStr := linkTagRe.ReplaceAllStringFunc(string(rawHTML), func(tag string) string {
		lower := strings.ToLower(tag)
		if !strings.Contains(lower, `rel="preload"`) || !strings.Contains(lower, `as="script"`) {
			return tag
		}

		var integrityAttr string
		if m := hrefAttrRe.FindStringSubmatch(tag); len(m) > 1 {
			cleanHref := path.Clean(m[1])
			if sri, ok := e.assetIntegrity[cleanHref]; ok {
				if !strings.Contains(lower, "integrity=") {
					integrityAttr = fmt.Sprintf(` integrity="%s"`, sri)
				}
				addHash("'" + sri + "'")
			}
		}

		nonceAttr := ""
		if !strings.Contains(lower, "nonce=") {
			nonceAttr = fmt.Sprintf(` nonce="%s"`, nonce)
		}

		if strings.HasSuffix(tag, "/>") {
			return tag[:len(tag)-2] + nonceAttr + integrityAttr + "/>"
		}
		return tag[:len(tag)-1] + nonceAttr + integrityAttr + ">"
	})

	// 2. Inject nonce and integrity into all <script ...>...</script> tags
	htmlStr = scriptTagRe.ReplaceAllStringFunc(htmlStr, func(fullMatch string) string {
		sub := scriptTagRe.FindStringSubmatch(fullMatch)
		if len(sub) < 3 {
			return fullMatch
		}
		attrs := sub[1]
		body := sub[2]
		lowerAttrs := strings.ToLower(attrs)

		var extraAttrs strings.Builder
		if !strings.Contains(lowerAttrs, "nonce=") {
			extraAttrs.WriteString(fmt.Sprintf(` nonce="%s"`, nonce))
		}

		if m := srcAttrRe.FindStringSubmatch(attrs); len(m) > 1 {
			// External script tag
			cleanSrc := path.Clean(m[1])
			if sri, ok := e.assetIntegrity[cleanSrc]; ok {
				if !strings.Contains(lowerAttrs, "integrity=") {
					extraAttrs.WriteString(fmt.Sprintf(` integrity="%s"`, sri))
				}
				addHash("'" + sri + "'")
			}
		} else if len(body) > 0 {
			// Inline script block: compute exact SHA-256 hash of script body
			sum := sha256.Sum256([]byte(body))
			b64 := base64.StdEncoding.EncodeToString(sum[:])
			addHash("'sha256-" + b64 + "'")
		}

		return fmt.Sprintf("<script%s%s>%s</script>", attrs, extraAttrs.String(), body)
	})

	cspHeader := e.buildHTMLCSPHeader(nonce, scriptHashes)
	return []byte(htmlStr), cspHeader, nil
}

func (e *Engine) buildHTMLCSPHeader(nonce string, scriptHashes []string) string {
	scriptSources := []string{
		"'self'",
		fmt.Sprintf("'nonce-%s'", nonce),
	}
	scriptSources = append(scriptSources, scriptHashes...)
	scriptSources = append(scriptSources, "'strict-dynamic'")
	scriptSrcJoined := strings.Join(scriptSources, " ")

	styleSrc := "'self'"
	styleSrcAttr := "'none'"
	if len(e.styleAttrHash) > 0 {
		hashesJoined := strings.Join(e.styleAttrHash, " ")
		styleSrc = fmt.Sprintf("'self' 'unsafe-hashes' %s", hashesJoined)
		styleSrcAttr = fmt.Sprintf("'unsafe-hashes' %s", hashesJoined)
	}

	directives := []string{
		"default-src 'none'",
		"script-src " + scriptSrcJoined,
		"script-src-elem " + scriptSrcJoined,
		"script-src-attr 'none'",
		"style-src " + styleSrc,
		"style-src-elem 'self'",
		"style-src-attr " + styleSrcAttr,
		"img-src 'self' data:",
		"font-src 'self'",
		"connect-src 'self'",
		"manifest-src 'self'",
		"worker-src 'self'",
		"media-src 'none'",
		"object-src 'none'",
		"frame-src 'none'",
		"frame-ancestors 'none'",
		"base-uri 'none'",
		"form-action 'none'",
		"upgrade-insecure-requests",
	}

	return strings.Join(directives, "; ")
}

// StrictAPICSP returns the locked-down CSP Level 3 policy for JSON API and non-HTML responses.
func StrictAPICSP() string {
	return "default-src 'none'; frame-ancestors 'none'; object-src 'none'; base-uri 'none'; form-action 'none'; upgrade-insecure-requests"
}
