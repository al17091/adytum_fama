package sources

import (
	"context"
	"time"

	"github.com/al17091/adytum_fama/pkg/models"
)

// EventConnector is the pluggable interface implemented by any observability or log source
// that extracts security mitigation/block events (e.g. Wazuh, AWS WAF, Apache HTTP Server, Nginx, Envoy).
type EventConnector interface {
	Name() string
	FetchEvents(ctx context.Context, window time.Duration) ([]models.BlockEvent, error)
}
