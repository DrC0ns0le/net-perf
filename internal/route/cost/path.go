package cost

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/DrC0ns0le/net-perf/internal/metrics"
)

const (
	pathCacheExpiration = 1 * time.Minute
)

type pathCacheEntry struct {
	latency    float64
	expiration time.Time
}

type pathLatencyCache struct {
	mu    sync.RWMutex
	cache map[string]pathCacheEntry
}

func newPathLatencyCache() *pathLatencyCache {
	cache := &pathLatencyCache{
		cache: make(map[string]pathCacheEntry),
	}

	go cache.cleanExpired()
	return cache
}

var globalPathLatencyCache = newPathLatencyCache()

func generatePathLatencyCacheKey(src, dst int) string {
	return fmt.Sprintf("%d-%d", src, dst)
}

func (c *pathLatencyCache) cleanExpired() {
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

// GetPathLatency returns the cached latency if available, otherwise calculates and caches it, src and dst is AS number
func GetPathLatency(ctx context.Context, src, dst int) (float64, error) {
	key := generatePathLatencyCacheKey(src, dst)

	// Try to get from cache first
	globalPathLatencyCache.mu.RLock()
	if entry, exists := globalPathLatencyCache.cache[key]; exists && time.Now().Before(entry.expiration) {
		globalPathLatencyCache.mu.RUnlock()
		return entry.latency, nil
	}
	globalPathLatencyCache.mu.RUnlock()

	// Not in cache, calculate
	latency, err := SetPathLatency(ctx, src, dst)
	if err != nil {
		return math.Inf(1), err
	}

	globalPathLatencyCache.mu.Lock()
	globalPathLatencyCache.cache[key] = pathCacheEntry{
		latency:    latency,
		expiration: time.Now().Add(pathCacheExpiration),
	}
	globalPathLatencyCache.mu.Unlock()

	return latency, nil
}

// SetPathLatency returns the latency of a path
// Custom path latencys can be added here
func SetPathLatency(ctx context.Context, src, dst int) (float64, error) {

	latency, _ := metrics.GetPingPathLatency(ctx, src, dst)

	if latency == 0 {
		return 0, fmt.Errorf("unexpected latency of 0 for %d -> %d", src, dst)
	}

	return latency / 2, nil
}

// OverwritePathLatency overwrites the cached latency for a given path with a user-provided value.
// The value will expire after pathCacheExpiration.
// The latency is used in the calculation of path costs.
func OverwritePathLatency(src, dst int, latency float64) error {

	globalPathLatencyCache.mu.Lock()
	globalPathLatencyCache.cache[generatePathLatencyCacheKey(src, dst)] = pathCacheEntry{
		latency:    latency,
		expiration: time.Now().Add(pathCacheExpiration),
	}
	globalPathLatencyCache.mu.Unlock()

	return nil
}

// RefreshPathLatency removes the cached path latency for the given path and then returns the newly calculated
// path latency. This can be used to force a refresh of the path latency cache for a specific path.
func RefreshPathLatency(ctx context.Context, src, dst int) (float64, error) {

	globalPathLatencyCache.mu.Lock()
	delete(globalPathLatencyCache.cache, generatePathLatencyCacheKey(src, dst))
	globalPathLatencyCache.mu.Unlock()

	return GetPathLatency(ctx, src, dst)
}
