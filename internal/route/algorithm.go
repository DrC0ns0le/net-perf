package route

import (
	"context"
	"fmt"
	"math"
	"strings"

	"github.com/DrC0ns0le/net-perf/internal/route/cost"
	"github.com/DrC0ns0le/net-perf/internal/route/routers"
	"github.com/DrC0ns0le/net-perf/internal/system/netctl"
)

func (rm *RouteManager) selectLowestCostBGPPath(ctx context.Context, route routers.Route) error {
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

		ctx, cancel := context.WithTimeout(ctx, *costContextTimeout)
		defer cancel()
		totalCost, err := rm.CalculateTotalCost(ctx, path.ASPath)
		if err != nil {
			rm.logger.Errorf("error calculating total cost for path %v: %v", path, err)
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

	if len(route.Paths[chosenPathIndex].ASPath) == 0 || route.Paths[chosenPathIndex].ASPath[0] == rm.Config.ASNumber {
		if err := netctl.RemoveRoute(route.Network, CustomRouteProtocol); err != nil {
			return fmt.Errorf("error removing route for network %s: %w", route.Network, err)
		}
	}

	if len(route.Paths[chosenPathIndex].ASPath) == 0 || !strings.HasPrefix(route.Paths[chosenPathIndex].Interface, "wg") {
		return nil
	} else {
		rm.RouteTable.AddRoute(route.Network, route.Paths[chosenPathIndex].Next)
	}

	return nil
}

// CalculateTotalCost returns the total cost of a BGP path given its AS path and the local AS number.
// The total cost is calculated as the sum of the costs of each hop in the path, plus 10 for each additional hop.
// If any of the intermediate costs are infinite, the total cost is set to infinity and returned.
func (rm *RouteManager) CalculateTotalCost(ctx context.Context, asPath []int) (float64, error) {
	var totalCost float64
	for i, as := range asPath {
		var c float64
		var err error
		if i > 0 {
			c, err = cost.GetPathCost(ctx, asPath[i-1], as)
		} else {
			c, err = cost.GetPathCost(ctx, rm.Config.ASNumber, as)
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
