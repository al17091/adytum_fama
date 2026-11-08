package wazuh

import (
	"testing"
)

func TestIsReservedOrPrivateIP(t *testing.T) {
	tests := []struct {
		ip       string
		reserved bool
	}{
		// Public IPs (should NOT be reserved)
		{"152.32.225.108", false},
		{"118.193.39.117", false},
		{"165.154.129.220", false},
		{"34.17.201.195", false},
		{"45.138.12.54", false},
		{"8.8.8.8", false},
		{"1.1.1.1", false},
		{"2600:1f18::1", false},

		// RFC 1918 Private IPs (MUST be reserved)
		{"10.0.0.1", true},
		{"10.42.1.227", true},
		{"172.16.0.1", true},
		{"172.20.10.5", true},
		{"172.31.255.254", true},
		{"192.168.1.1", true},
		{"192.168.0.100", true},

		// Loopback (MUST be reserved)
		{"127.0.0.1", true},
		{"::1", true},

		// Link-Local (MUST be reserved)
		{"169.254.1.1", true},
		{"fe80::1", true},

		// CGNAT / Tailscale (RFC 6598: 100.64.0.0/10) (MUST be reserved)
		{"100.64.0.2", true},
		{"100.64.0.7", true},
		{"100.127.255.255", true},

		// Reserved / Documentation (RFC 5737)
		{"192.0.2.1", true},
		{"198.51.100.1", true},
		{"203.0.113.1", true},
		{"198.18.0.1", true},
		{"240.0.0.1", true},

		// Invalid IPs
		{"", true},
		{"invalid-ip", true},
		{"999.999.999.999", true},
	}

	for _, tt := range tests {
		got := IsReservedOrPrivateIP(tt.ip)
		if got != tt.reserved {
			t.Errorf("IsReservedOrPrivateIP(%q) = %v, want %v", tt.ip, got, tt.reserved)
		}
	}
}
