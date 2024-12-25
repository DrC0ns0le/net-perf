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
	RouteTable *RouteTable

	GlobalStopCh chan struct{}
	RTUpdateCh   chan struct{}
}

func (b *Bird) Watcher() {
	// Initial run
	err := b.run()
	if err != nil {
		logging.Errorf("Error in intial bird route table sync: %v", err)
	}
	netctl.RemoveAllManagedRoutes()
	err = b.run()
	if err != nil {
		logging.Errorf("Error in intial bird route table sync: %v", err)
	}

	// Calculate first interval
	now := time.Now()
	nextInterval := now.Truncate(*updateInterval).Add(*updateInterval)
	firstSleep := nextInterval.Sub(now)

	// Wait for the first interval boundary
	timer := time.NewTimer(firstSleep)
	<-timer.C
	if err := b.run(); err != nil {
		logging.Errorf("Error in bird route table sync: %v", err)
	}
	timer.Stop()

	// Create ticker starting from the interval boundary
	ticker := time.NewTicker(*updateInterval)
	defer ticker.Stop()

	// Main daemon loop
	for {
		select {
		case <-b.GlobalStopCh:
			return
		case <-ticker.C:
			if err := b.run(); err != nil {
				logging.Errorf("Error in bird route table sync: %v", err)
				// Continue running even if there's an error
			}
		case <-b.RTUpdateCh:
			if err := b.run(); err != nil {
				logging.Errorf("Error in bird route table sync: %v", err)
			}
		}
	}
}

func (b *Bird) run() error {
	ctx, cancel := context.WithTimeout(context.Background(), *costContextTimeout)
	defer cancel()

	b.RouteTable.updateMu.Lock()
	defer b.RouteTable.updateMu.Unlock()
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
		minCost := math.Inf(1)
		for i, path := range route.Paths {
			totalCost := CalculateTotalCost(ctx, b.Config, path.ASPath)
			if totalCost < minCost {
				minCost = totalCost
				chosenPathIndex = i
			}
		}

		if len(route.Paths[chosenPathIndex].ASPath) == 0 || route.Paths[chosenPathIndex].ASPath[0] == b.Config.ASNumber {
			if err := netctl.RemoveRoute(route.Network); err != nil {
				return fmt.Errorf("error removing route for network %s: %w", route.Network, err)
			}
		}

		if len(route.Paths[chosenPathIndex].ASPath) == 0 || !strings.HasPrefix(route.Paths[chosenPathIndex].Interface, "wg") {
			continue
		} else {
			err := netctl.ConfigureRoute(route.Network, route.Paths[chosenPathIndex].Next, func() net.IP {
				if mode == "v6" {
					return outboundV6
				} else {
					return outboundV4
				}
			}())
			if err != nil {
				return fmt.Errorf("error configuring route for network %s: %w", route.Network, err)
			}

			b.RouteTable.AddRoute(route.Network, route.Paths[chosenPathIndex].Next)
		}
	}

	return nil
}

// CalculateTotalCost returns the total cost of a BGP path given its AS path and the local AS number.
// The total cost is calculated as the sum of the costs of each hop in the path, plus 10 for each additional hop.
// If any of the intermediate costs are infinite, the total cost is set to infinity and returned.
func CalculateTotalCost(ctx context.Context, config *bird.Config, asPath []int) float64 {
	var totalCost float64
	for i, as := range asPath {
		var c float64
		if i > 0 {
			c = cost.GetPathCost(ctx, asPath[i-1], as)
		} else {
			c = cost.GetPathCost(ctx, config.ASNumber, as)
		}
		if c == math.Inf(1) {
			return math.Inf(1)
		}
		totalCost += c + 10
	}
	return totalCost
}
