package route

import (
	"context"
	"fmt"
	"math"
	"net"
	"strings"

	"github.com/DrC0ns0le/net-perf/internal/route/cost"
	"github.com/DrC0ns0le/net-perf/internal/route/routers"
	"github.com/DrC0ns0le/net-perf/internal/system/netctl"
)

// selectLowestCostBGPPath iterates through the BGP paths of a given route and chooses
// the one with the lowest total cost. If no path has a calculable cost, it falls
// back to the shortest path. If the selected path is empty or the first AS in the
// path is the local AS number, the route is removed. If the selected path is not
// empty and the interface is a wireguard interface, a route is installed to the
// next hop in the path.
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
		totalCost, err := rm.calculateTotalCost(ctx, path.ASPath)
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
func (rm *RouteManager) calculateTotalCost(ctx context.Context, asPath []int) (float64, error) {
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

func (rm *RouteManager) graphBasedShortestPath(ctx context.Context, route routers.Route) error {
	rm.PathMapMu.Lock()
	defer rm.PathMapMu.Unlock()

	var (
		gw       net.IP
		ok       bool
		originAS int
	)

	cost := math.Inf(1)
	originAS = route.OriginAS

	// special case for ASes > 65000
	if originAS >= 65000 {
		for _, path := range route.Paths {
			if strings.HasPrefix(path.Interface, "en") {
				return nil
			}
			if len(path.ASPath) < 2 {
				continue
			}
			_, c, err := rm.Graph.GetShortestPath(asToSiteID(rm.Config.ASNumber), asToSiteID(path.ASPath[len(path.ASPath)-2]))
			if err != nil {
				return fmt.Errorf("error getting shortest path: %w", err)
			}
			if c < cost {
				cost = c
				originAS = path.ASPath[len(path.ASPath)-2]
			}
		}
		if originAS >= 65000 {
			return fmt.Errorf("no origin AS found")
		}
	}

	mode := "v4"
	if route.Network.IP.To4() == nil {
		mode = "v6"
	}
	key := fmt.Sprintf("%s-%d", mode, originAS)

	if gw, ok = rm.PathMap[key]; !ok {
		path, _, err := rm.Graph.GetShortestPath(asToSiteID(rm.Config.ASNumber), asToSiteID(originAS))
		if err != nil {
			return fmt.Errorf("error getting shortest path for %d -> %d: %w", asToSiteID(rm.Config.ASNumber), asToSiteID(originAS), err)
		}
		if len(path) < 2 {
			return fmt.Errorf("no path found")
		}
		gw, err = rm.findSiteGW(mode, path[1])
		if err != nil {
			return fmt.Errorf("error finding site gw: %w", err)
		}
		rm.PathMap[key] = gw
	}

	rm.RouteTable.AddRoute(route.Network, gw)

	return nil
}

func (rm *RouteManager) findSiteGW(mode string, via int) (net.IP, error) {
	routes, _, err := rm.Router.GetRoutes(mode)
	if err != nil {
		return nil, fmt.Errorf("error getting routes: %w", err)
	}

	for _, route := range routes {
		for _, path := range route.Paths {
			if len(path.ASPath) > 0 && asToSiteID(path.ASPath[0]) == via {
				return path.Next, nil
			}
		}
	}

	return nil, fmt.Errorf("could not find gateway for site %d", via)
}

func siteIDToAS(siteID int) int {
	return siteID + 64512
}

func asToSiteID(as int) int {
	return as - 64512
}
