package route

import (
	"context"
	"flag"
	"fmt"
	"math"
	"net"
	"strings"
	"time"

	"github.com/DrC0ns0le/net-perf/internal/route/bird"
	"github.com/DrC0ns0le/net-perf/internal/route/cost"
	"github.com/DrC0ns0le/net-perf/internal/system"
	"github.com/DrC0ns0le/net-perf/internal/system/netctl"
	"github.com/DrC0ns0le/net-perf/pkg/logging"
)

var (
	costContextTimeout = flag.Duration("bird.timeout", 5*time.Second, "Timeout for route cost requests")
	updateInterval     = flag.Duration("bird.interval", 15*time.Minute, "Update interval in minutes")
)

const (
	// CustomRouteProtocol is a unique identifier for routes managed by this package
	// See /etc/iproute2/rt_protos for standard protocol numbers
	CustomRouteProtocol = 201
)

type Bird struct {
	Config     *bird.Config
	RouteTable *system.RouteTable

	StopCh     chan struct{}
	RTUpdateCh chan struct{}

	Logger logging.Logger
}

func (b *Bird) Watcher() {
	// Initial run
	err := b.run()
	if err != nil {
		b.Logger.Errorf("error in initial bird route table sync: %v", err)
	}
	removed, err := netctl.RemoveAllManagedRoutes()
	if err != nil {
		b.Logger.Errorf("removed only %d routes: %v", removed, err)
	}
	b.Logger.Infof("removed %d routes", removed)
	err = b.run()
	if err != nil {
		b.Logger.Errorf("error in initial bird route table sync: %v", err)
	}

	b.RouteTable.MarkReady()

	// Calculate first interval
	now := time.Now()
	nextInterval := now.Truncate(*updateInterval).Add(*updateInterval)
	firstSleep := nextInterval.Sub(now)

	// Wait for the first interval boundary
	timer := time.NewTimer(firstSleep)

	for {
		select {
		case <-b.StopCh:
			return
		case <-timer.C:
			timer.Stop()
			if err := b.run(); err != nil {
				b.Logger.Errorf("error syncing bird route table: %v", err)
			}
			goto mainDaemonLoop
		case <-b.RTUpdateCh:
			b.Logger.Debugf("triggering route table update")
			if err := b.run(); err != nil {
				b.Logger.Errorf("error syncing bird route table: %v", err)
			}
		}
	}

mainDaemonLoop:

	// Create ticker starting from after we handled the first event
	ticker := time.NewTicker(*updateInterval)
	defer ticker.Stop()

	// Main daemon loop
	for {
		select {
		case <-b.StopCh:
			return
		case <-ticker.C:
			if err := b.run(); err != nil {
				b.Logger.Errorf("error syncing bird route table: %v", err)
				// Continue running even if there's an error
			}
		case <-b.RTUpdateCh:
			b.Logger.Infof("triggering route table update")
			if err := b.run(); err != nil {
				b.Logger.Errorf("error syncing bird route table: %v", err)
			}
		}
	}
}

func (b *Bird) run() error {
	ctx, cancel := context.WithTimeout(context.Background(), *costContextTimeout)
	defer cancel()

	b.RouteTable.Lock()
	defer b.RouteTable.Unlock()
	b.RouteTable.ClearRoutes()

	for _, mode := range []string{"v4", "v6"} {
		routes, _, err := bird.GetRoutes(mode)
		if err != nil {
			return fmt.Errorf("error getting routes: %v", err)
		}

		if err := b.UpdateRoutes(ctx, routes, mode); err != nil {
			return fmt.Errorf("error updating routes for %s: %v", mode, err)
		}

	}

	return nil
}

func (b *Bird) UpdateRoutes(ctx context.Context, routes []bird.Route, mode string) error {
	outboundV4, outboundV6, err := netctl.GetOutboundIPs()
	if err != nil {
		return fmt.Errorf("error getting outbound IP: %w", err)

	}
	for _, route := range routes {
		if len(route.Paths) == 0 {
			continue
		}
		var chosenPathIndex int
		var shortestPathIndex int
		minCost := math.Inf(1)
		minASLen := math.MaxInt
		for i, path := range route.Paths {
			// as fallback, choose the shortest path
			if len(path.ASPath) < minASLen {
				minASLen = len(path.ASPath)
				shortestPathIndex = i
			}

			totalCost, err := CalculateTotalCost(ctx, b.Config, path.ASPath)
			if err != nil {
				b.Logger.Errorf("error calculating total cost for path %v: %v", path, err)
				continue
			}
			if totalCost < minCost {
				minCost = totalCost
				chosenPathIndex = i
			}
		}

		if minCost == math.Inf(1) {
			chosenPathIndex = shortestPathIndex
		}

		if len(route.Paths[chosenPathIndex].ASPath) == 0 || route.Paths[chosenPathIndex].ASPath[0] == b.Config.ASNumber {
			if err := netctl.RemoveRoute(route.Network); err != nil {
				return fmt.Errorf("error removing route for network %s: %w", route.Network, err)
			}
		}

		if len(route.Paths[chosenPathIndex].ASPath) == 0 || !strings.HasPrefix(route.Paths[chosenPathIndex].Interface, "wg") {
			continue
		} else {
			code, err := netctl.ConfigureRoute(route.Network, route.Paths[chosenPathIndex].Next, func() net.IP {
				if mode == "v6" {
					return outboundV6
				} else {
					return outboundV4
				}
			}())
			if err != nil {
				return fmt.Errorf("error configuring route for network %s: %w", route.Network, err)
			}
			switch code {
			case 1:
				b.Logger.Infof("new route added for network %s via %s", route.Network, route.Paths[chosenPathIndex].Next)
			case 2:
				b.Logger.Infof("existing route updated for network %s via %s", route.Network, route.Paths[chosenPathIndex].Next)
			default:
				b.Logger.Debugf("route unchanged for network %s via %s", route.Network, route.Paths[chosenPathIndex].Next)
			}

			b.RouteTable.AddRoute(route.Network, route.Paths[chosenPathIndex].Next)
		}
	}

	return nil
}

// CalculateTotalCost returns the total cost of a BGP path given its AS path and the local AS number.
// The total cost is calculated as the sum of the costs of each hop in the path, plus 10 for each additional hop.
// If any of the intermediate costs are infinite, the total cost is set to infinity and returned.
func CalculateTotalCost(ctx context.Context, config *bird.Config, asPath []int) (float64, error) {
	var totalCost float64
	for i, as := range asPath {
		var c float64
		var err error
		if i > 0 {
			c, err = cost.GetPathCost(ctx, asPath[i-1], as)
		} else {
			c, err = cost.GetPathCost(ctx, config.ASNumber, as)
		}
		if err != nil {
			return math.Inf(1), err
		}
		// if any of the intermediate costs are infinite, early return infinity
		if c == math.Inf(1) {
			return math.Inf(1), nil
		}
		totalCost += c + 10
	}
	return totalCost, nil
}
