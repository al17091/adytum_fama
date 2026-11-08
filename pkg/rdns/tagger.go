package rdns

import (
	"context"
	"net"
	"strings"
	"sync"
	"time"
)

// ActorInfo holds categorized scanner / bot metadata
type ActorInfo struct {
	ReverseDNS string
	ActorType  string // RESEARCH_SCANNER, BOT_SCANNER, SEARCH_ENGINE, UNKNOWN
	ActorName  string // Censys, Shadowserver, Shodan, Googlebot, etc.
}

// Tagger handles reverse DNS lookup with an in-memory cache and scanner matching
type Tagger struct {
	cache sync.Map
}

// NewTagger creates an instance of the rDNS and scanner classifier
func NewTagger() *Tagger {
	return &Tagger{}
}

// Lookup resolves the rDNS host and determines actor classification
func (t *Tagger) Lookup(ctx context.Context, ipStr string, userAgent string) ActorInfo {
	if val, ok := t.cache.Load(ipStr); ok {
		return val.(ActorInfo)
	}

	info := ActorInfo{
		ReverseDNS: "",
		ActorType:  "UNKNOWN",
		ActorName:  "Unknown",
	}

	// 1. Perform rDNS Lookup with a strict 800ms timeout
	lookupCtx, cancel := context.WithTimeout(ctx, 800*time.Millisecond)
	defer cancel()

	resolver := net.DefaultResolver
	names, err := resolver.LookupAddr(lookupCtx, ipStr)
	if err == nil && len(names) > 0 {
		info.ReverseDNS = strings.TrimSuffix(names[0], ".")
	}

	// 2. Classify based on rDNS hostname and User-Agent
	rdnsLower := strings.ToLower(info.ReverseDNS)
	uaLower := strings.ToLower(userAgent)

	switch {
	// Censys
	case strings.Contains(rdnsLower, "censys-scanner.com") || strings.Contains(uaLower, "censys"):
		info.ActorType = "RESEARCH_SCANNER"
		info.ActorName = "Censys"

	// Shadowserver
	case strings.Contains(rdnsLower, "shadowserver.org") || strings.Contains(uaLower, "shadowserver"):
		info.ActorType = "RESEARCH_SCANNER"
		info.ActorName = "Shadowserver"

	// Shodan
	case strings.Contains(rdnsLower, "shodan.io") || strings.Contains(uaLower, "shodan"):
		info.ActorType = "RESEARCH_SCANNER"
		info.ActorName = "Shodan"

	// Project Sonar / Rapid7
	case strings.Contains(rdnsLower, "sonar.labs.rapid7.com") || strings.Contains(uaLower, "rapid7"):
		info.ActorType = "RESEARCH_SCANNER"
		info.ActorName = "Rapid7"

	// Palo Alto Networks
	case strings.Contains(rdnsLower, "paloaltonetworks.com") || strings.Contains(uaLower, "expanse"):
		info.ActorType = "RESEARCH_SCANNER"
		info.ActorName = "Palo Alto Networks"

	// LeakIX
	case strings.Contains(rdnsLower, "leakix.org") || strings.Contains(uaLower, "leakix"):
		info.ActorType = "RESEARCH_SCANNER"
		info.ActorName = "LeakIX"

	// Academic & University Scanners
	case strings.Contains(rdnsLower, "rwth-aachen.de") || strings.Contains(rdnsLower, "rub.de") || strings.Contains(rdnsLower, "secsys.sec.uni-hannover.de"):
		info.ActorType = "RESEARCH_SCANNER"
		info.ActorName = "Academic Research Scanner"

	// Search Engine Bots
	case strings.Contains(rdnsLower, "googlebot.com") || strings.Contains(uaLower, "googlebot"):
		info.ActorType = "SEARCH_ENGINE"
		info.ActorName = "Googlebot"
	case strings.Contains(rdnsLower, "search.msn.com") || strings.Contains(uaLower, "bingbot"):
		info.ActorType = "SEARCH_ENGINE"
		info.ActorName = "Bingbot"

	// Known automated scanners / exploit bots
	case strings.Contains(uaLower, "zgrab") || strings.Contains(uaLower, "masscan") || strings.Contains(uaLower, "nmap") || strings.Contains(uaLower, "nikto"):
		info.ActorType = "BOT_SCANNER"
		info.ActorName = "Automated Security Probe"

	default:
		info.ActorType = "UNKNOWN"
		info.ActorName = "Unknown"
	}

	t.cache.Store(ipStr, info)
	return info
}
