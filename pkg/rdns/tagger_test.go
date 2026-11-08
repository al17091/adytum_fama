package rdns

import (
	"context"
	"testing"
)

func TestTagger_Classification(t *testing.T) {
	tagger := NewTagger()
	ctx := context.Background()

	tests := []struct {
		name          string
		ip            string
		ua            string
		expectedType  string
		expectedActor string
	}{
		{
			name:          "Censys User Agent",
			ip:            "1.2.3.4",
			ua:            "Mozilla/5.0 (compatible; CensysInspect/1.1; +https://about.censys.io/)",
			expectedType:  "RESEARCH_SCANNER",
			expectedActor: "Censys",
		},
		{
			name:          "Masscan Scanner",
			ip:            "1.2.3.5",
			ua:            "masscan/1.3.2",
			expectedType:  "BOT_SCANNER",
			expectedActor: "Automated Security Probe",
		},
		{
			name:          "Googlebot",
			ip:            "1.2.3.6",
			ua:            "Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)",
			expectedType:  "SEARCH_ENGINE",
			expectedActor: "Googlebot",
		},
		{
			name:          "Unknown standard browser",
			ip:            "1.2.3.7",
			ua:            "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
			expectedType:  "UNKNOWN",
			expectedActor: "Unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := tagger.Lookup(ctx, tt.ip, tt.ua)
			if info.ActorType != tt.expectedType {
				t.Errorf("ActorType = %s, expected %s", info.ActorType, tt.expectedType)
			}
			if info.ActorName != tt.expectedActor {
				t.Errorf("ActorName = %s, expected %s", info.ActorName, tt.expectedActor)
			}
		})
	}
}
