package dedupe

import (
	"testing"
	"time"

	"github.com/al17091/adytum_fama/pkg/models"
)

func TestCalculateHourlyMetrics(t *testing.T) {
	tests := []struct {
		name                 string
		input                models.HourlyTrafficStats
		expectedTotal        int64
		expectedMitigated    int64
		expectedLegitimate   int64
		expectedRatio        float64
	}{
		{
			name: "Standard traffic mix",
			input: models.HourlyTrafficStats{
				BucketTime:          time.Now(),
				CFRequests:          10000,
				AWSWAFBlocked:       1500,
				Foramen444Drops:     500,
				ForamenModSecBlocks: 100,
				RetisTotalRequests:  2000,
				Retis444Drops:       300,
				RetisModSecBlocks:   50,
			},
			expectedTotal:      12500, // 10000 + 500 + 2000
			expectedMitigated:  2450,  // 1500 + 500 + 100 + 300 + 50
			expectedLegitimate: 10050, // 12500 - 2450
			expectedRatio:      19.60, // 2450 / 12500 * 100
		},
		{
			name: "Zero traffic",
			input: models.HourlyTrafficStats{
				BucketTime: time.Now(),
			},
			expectedTotal:      0,
			expectedMitigated:  0,
			expectedLegitimate: 0,
			expectedRatio:      0.0,
		},
		{
			name: "Hostile spike exceeding totals (clamped to total requests)",
			input: models.HourlyTrafficStats{
				BucketTime:          time.Now(),
				CFRequests:          100,
				AWSWAFBlocked:       100,
				Foramen444Drops:     50,
				ForamenModSecBlocks: 10,
				RetisTotalRequests:  0,
			},
			expectedTotal:      160,   // clamped to mitigated (100 + 50 + 10)
			expectedMitigated:  160,   // 100 + 50 + 10
			expectedLegitimate: 0,     // max(0, 160-160)
			expectedRatio:      100.0, // 160 / 160 * 100 = 100%
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats := tt.input
			CalculateHourlyMetrics(&stats)

			if stats.TotalRequests != tt.expectedTotal {
				t.Errorf("TotalRequests = %d, expected %d", stats.TotalRequests, tt.expectedTotal)
			}
			if stats.MitigatedRequests != tt.expectedMitigated {
				t.Errorf("MitigatedRequests = %d, expected %d", stats.MitigatedRequests, tt.expectedMitigated)
			}
			if stats.LegitimateRequests != tt.expectedLegitimate {
				t.Errorf("LegitimateRequests = %d, expected %d", stats.LegitimateRequests, tt.expectedLegitimate)
			}
			if stats.MaliciousRatio != tt.expectedRatio {
				t.Errorf("MaliciousRatio = %f, expected %f", stats.MaliciousRatio, tt.expectedRatio)
			}
		})
	}
}
