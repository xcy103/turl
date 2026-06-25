package cache

import (
	"context"
	"errors"

	"github.com/beihai0xff/turl/pkg/metrics"
)

var _ Interface = (*metricsCache)(nil)

// metricsCache decorates a cache layer and records the hit/miss outcome of every
// Get. It embeds Interface, so Set/Del/Ping/Close pass straight through and only
// the read path is instrumented.
type metricsCache struct {
	Interface

	layer string
}

// newMetricsCache wraps c so its lookups are counted under the given layer label.
func newMetricsCache(layer string, c Interface) Interface {
	return &metricsCache{Interface: c, layer: layer}
}

// Get records whether the lookup hit or missed before returning the result.
// A cache miss is the expected ErrCacheMiss; any other error is not counted as a
// lookup outcome, since it reflects a backend failure rather than a hit or miss.
func (m *metricsCache) Get(ctx context.Context, k string) ([]byte, error) {
	v, err := m.Interface.Get(ctx, k)

	switch {
	case err == nil:
		metrics.RecordCacheResult(m.layer, true)
	case errors.Is(err, ErrCacheMiss):
		metrics.RecordCacheResult(m.layer, false)
	}

	return v, err
}
