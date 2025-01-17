package cost

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net"
	"sync"
	"time"

	"github.com/DrC0ns0le/net-perf/internal/metrics"
	"github.com/DrC0ns0le/net-perf/internal/system/netctl"
)

const (
	costCacheExpiration = 15 * time.Second
)

type costCacheEntry struct {
	cost       float64
	expiration time.Time
}

type pathCostCache struct {
	mu    sync.RWMutex
	cache map[string]costCacheEntry
}

func newPathCostCache() *pathCostCache {
	cache := &pathCostCache{
		cache: make(map[string]costCacheEntry),
	}

	go cache.cleanExpired()
	return cache
}

var globalCostCache = newPathCostCache()

func generateCostCacheKey(src, dst int) string {
	return fmt.Sprintf("%d-%d", src, dst)
}

func (c *pathCostCache) cleanExpired() {
	ticker := time.NewTicker(1 * time.Minute)
	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for key, entry := range c.cache {
			if now.After(entry.expiration) {
				delete(c.cache, key)
			}
		}
		c.mu.Unlock()
	}
}

// GetPathCost returns the cached cost if available, otherwise calculates and caches it, src and dst is AS number
func GetPathCost(ctx context.Context, src, dst int) (float64, error) {
	key := generateCostCacheKey(src, dst)

	// Try to get from cache first
	globalCostCache.mu.RLock()
	if entry, exists := globalCostCache.cache[key]; exists && time.Now().Before(entry.expiration) {
		globalCostCache.mu.RUnlock()
		return entry.cost, nil
	}
	globalCostCache.mu.RUnlock()

	// Not in cache, calculate
	cost, err := SetPathCost(ctx, src, dst)
	if err != nil {
		return math.Inf(1), err
	}

	globalCostCache.mu.Lock()
	globalCostCache.cache[key] = costCacheEntry{
		cost:       cost,
		expiration: time.Now().Add(costCacheExpiration),
	}
	globalCostCache.mu.Unlock()

	return cost, nil
}

// SetPathCost returns the cost of a path
// Custom path costs can be added here
func SetPathCost(ctx context.Context, src, dst int) (float64, error) {

	switch dst {
	case 65000:
		return 0, nil
	}

	// Check if dst wg interface exists, if not return infinity
	if !netctl.DstWGInterfaceExists(dst - 64512) {
		return math.Inf(1), nil
	}

	_, cost, err := metrics.GetPreferredPath(ctx, src-64512, dst-64512)
	if err != nil {
		var netErr *net.OpError
		switch {
		case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded), errors.As(err, &netErr):
			return 0, fmt.Errorf("timed out getting preferred interface version for %d -> %d", src, dst)
		case errors.Is(err, metrics.ErrNoPaths):
			return 0, fmt.Errorf("no available paths for %d -> %d", src, dst)
		default:
			return 0, fmt.Errorf("failed to get path cost for %d -> %d: %v", src, dst, err)
		}
	}

	if cost == 0 {
		return 0, fmt.Errorf("unexpected cost of 0 for %d -> %d", src, dst)
	}

	switch dst {
	case 64512:
		cost = 3 * cost
	}

	return cost, nil
}
