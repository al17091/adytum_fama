package dedupe

import (
	"math"

	"github.com/al17091/adytum_fama/pkg/models"
)

// CalculateHourlyMetrics applies mathematical deduplication to prevent double-counting
// between CloudFront edge requests and gateway origin drops/bypasses.
func CalculateHourlyMetrics(stats *models.HourlyTrafficStats) {
	// Mitigated hostile requests across all defense layers:
	stats.MitigatedRequests = stats.AWSWAFBlocked +
		stats.Foramen444Drops +
		stats.ForamenModSecBlocks +
		stats.Retis444Drops +
		stats.RetisModSecBlocks

	// Total requests arriving at the entire infrastructure:
	// CloudFront total requests + foramen direct IP / invalid host drops (444) + retis total requests
	// TotalRequests must always be at least as large as the mitigated requests observed
	stats.TotalRequests = stats.CFRequests + stats.Foramen444Drops + stats.RetisTotalRequests
	if stats.TotalRequests < stats.MitigatedRequests {
		stats.TotalRequests = stats.MitigatedRequests
	}

	// Legitimate / allowed traffic passed through:
	if stats.TotalRequests > stats.MitigatedRequests {
		stats.LegitimateRequests = stats.TotalRequests - stats.MitigatedRequests
	} else {
		stats.LegitimateRequests = 0
	}

	// Malicious traffic ratio percentage (bounded between 0.0% and 100.0%):
	if stats.TotalRequests > 0 {
		ratio := (float64(stats.MitigatedRequests) / float64(stats.TotalRequests)) * 100.0
		if ratio > 100.0 {
			ratio = 100.0
		}
		stats.MaliciousRatio = math.Round(ratio*100) / 100.0 // 2 decimal places
	} else {
		stats.MaliciousRatio = 0.0
	}
}
