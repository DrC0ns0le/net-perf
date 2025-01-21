package route

import (
	"context"
	"flag"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/DrC0ns0le/net-perf/internal/route/finder"
	"github.com/DrC0ns0le/net-perf/internal/route/routers"
	"github.com/DrC0ns0le/net-perf/internal/route/routers/bird"
	"github.com/DrC0ns0le/net-perf/internal/system"
	"github.com/DrC0ns0le/net-perf/internal/system/netctl"
	"github.com/DrC0ns0le/net-perf/pkg/logging"
	"github.com/vishvananda/netlink"
)

var (
	costContextTimeout = flag.Duration("router.timeout", 5*time.Second, "Timeout for route cost requests")
	updateInterval     = flag.Duration("router.interval", 15*time.Minute, "Update interval in minutes")
)

const (
	// CustomRouteProtocol is a unique identifier for routes managed by this package
	// See /etc/iproute2/rt_protos for standard protocol numbers
	CustomRouteProtocol = 201
)

type RouteManager struct {
	siteID                 int
	outboundV4, outboundV6 net.IP
	customRouteProtocol    netlink.RouteProtocol

	Config     routers.Config
	RouteTable *system.RouteTable

	Graph      *finder.Graph
	graphReady bool
	PathMapMu  sync.RWMutex
	PathMap    map[string]net.IP

	// routing daemon interface
	Router routers.Router

	stopCh     chan struct{}
	rtUpdateCh chan struct{}

	logger logging.Logger
}

func NewRouteManager(global *system.Node) *RouteManager {
	return &RouteManager{
		siteID:              global.SiteID,
		customRouteProtocol: netlink.RouteProtocol(CustomRouteProtocol),

		RouteTable: global.RouteTable,

		PathMap: make(map[string]net.IP),

		Router: bird.NewRouter(),

		stopCh:     global.StopCh,
		rtUpdateCh: global.RTUpdateCh,
		logger:     global.Logger,
	}
}

func (rm *RouteManager) Start() error {
	var err error

	rm.Config, err = rm.Router.GetConfig("v4")
	if err != nil {
		return err
	}

	if rm.Config.ASNumber != rm.siteID+64512 {
		return fmt.Errorf("node id has invalid AS number: %d", rm.Config.ASNumber)
	}

	rm.outboundV4, rm.outboundV6, err = netctl.GetOutboundIPs()
	if err != nil {
		return fmt.Errorf("error getting outbound IP: %w", err)
	}

	// Initialize graph
	ctx, cancel := context.WithTimeout(context.Background(), *costContextTimeout)
	defer cancel()
	rm.Graph, err = finder.NewGraph(ctx)
	if err != nil {
		return fmt.Errorf("error initializing graph: %w", err)
	}

	// Initial run
	err = rm.UpdateRouteTable()
	if err != nil {
		rm.logger.Errorf("error in updating router route table: %v", err)
	}
	removed, err := netctl.RemoveAllManagedRoutes(CustomRouteProtocol)
	if err != nil {
		rm.logger.Errorf("failed to remove all routes, removed only %d routes: %v", removed, err)
	}
	rm.logger.Debugf("removed %d routes", removed)
	err = rm.applyRouteTable()
	if err != nil {
		rm.logger.Errorf("error in applying router route table: %v", err)
	}

	rm.RouteTable.MarkReady()

	rm.logger.Infof("startup route table sync completed")

	// Calculate first interval
	now := time.Now()
	nextInterval := now.Truncate(*updateInterval).Add(*updateInterval)
	firstSleep := nextInterval.Sub(now)

	// Wait for the first interval boundary
	timer := time.NewTimer(firstSleep)

alignmentLoop:
	for {
		select {
		case <-rm.stopCh:
			return nil
		case <-timer.C:
			timer.Stop()
			if err := rm.SyncRouteTable(); err != nil {
				rm.logger.Errorf("error syncing router route table: %v", err)
			}
			break alignmentLoop
		case <-rm.rtUpdateCh:
			rm.logger.Debugf("triggering route table update")
			if err := rm.SyncRouteTable(); err != nil {
				rm.logger.Errorf("error syncing router route table: %v", err)
			}
		}
	}
	rm.run()

	return nil
}

func (rm *RouteManager) run() {
	// Create ticker starting from after we handled the first event
	ticker := time.NewTicker(*updateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-rm.stopCh:
			return
		case <-ticker.C:
			if err := rm.SyncRouteTable(); err != nil {
				rm.logger.Errorf("error syncing router route table: %v", err)
				// Continue running even if there's an error
			}
		case <-rm.rtUpdateCh:
			rm.logger.Infof("triggering route table update")
			if err := rm.SyncRouteTable(); err != nil {
				rm.logger.Errorf("error syncing router route table: %v", err)
			}
		}
	}
}

func (rm *RouteManager) SyncRouteTable() error {
	var err error

	if err = rm.SyncGraph(); err != nil {
		return fmt.Errorf("error updating graph: %w", err)
	}
	if err = rm.UpdateRouteTable(); err != nil {
		return fmt.Errorf("error updating route table: %w", err)
	}
	if err = rm.removeOldRoutes(); err != nil {
		return fmt.Errorf("error removing outdated routes: %w", err)
	}
	return rm.applyRouteTable()
}

func (rm *RouteManager) SyncGraph() error {
	var err error
	ctx, cancel := context.WithTimeout(context.Background(), *costContextTimeout)
	defer cancel()
	if err = rm.Graph.RefreshWeights(ctx); err != nil {
		rm.logger.Errorf("error refreshing graph route weights: %v", err)
		rm.graphReady = false
	} else {
		rm.PathMapMu.Lock()
		defer rm.PathMapMu.Unlock()
		rm.PathMap = make(map[string]net.IP)
		rm.graphReady = true
	}
	return nil
}

func (rm *RouteManager) UpdateRouteTable() error {
	rm.RouteTable.Lock()
	defer rm.RouteTable.Unlock()
	rm.RouteTable.ClearRoutes()

	for _, mode := range []string{"v4", "v6"} {
		routes, _, err := rm.Router.GetRoutes(mode)
		if err != nil {
			return fmt.Errorf("error getting routes: %w", err)
		}

		if err := rm.addToRouteTable(routes); err != nil {
			return fmt.Errorf("error adding routes to route table: %w", err)
		}

	}

	return nil
}

func (rm *RouteManager) addToRouteTable(route []routers.Route) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(len(route))**costContextTimeout)
	defer cancel()
	for _, r := range route {
		if r.OriginAS == rm.Config.ASNumber || r.OriginAS == 0 || len(r.Paths) == 0 {
			continue
		} else {
			// we need to handle this route, ie running through the routing algorithms to find the gw
			if rm.graphReady {
				if err := rm.graphBasedShortestPath(ctx, r); err != nil {
					return fmt.Errorf("error selecting graph based shortest path: %w", err)
				}
			} else if err := rm.selectLowestCostBGPPath(ctx, r); err != nil {
				return fmt.Errorf("error selecting lowest cost BGP path: %w", err)
			}
		}
	}

	return nil
}

func (rm *RouteManager) applyRouteTable() error {
	for _, route := range rm.RouteTable.GetRoutes() {
		code, err := netctl.ConfigureRoute(route.Destination, route.Gateway, func() net.IP {
			if route.Destination.IP.To4() == nil {
				return rm.outboundV6
			}
			return rm.outboundV4
		}(), CustomRouteProtocol)
		if err != nil {
			return fmt.Errorf("error configuring route for network %s: %w", route.Destination, err)
		}
		switch code {
		case 1:
			rm.logger.Infof("new route added for network %s via %s", route.Destination, route.Gateway)
		case 2:
			rm.logger.Infof("existing route updated for network %s via %s", route.Destination, route.Gateway)
		default:
			rm.logger.Debugf("route unchanged for network %s via %s", route.Destination, route.Gateway)
		}
	}

	return nil
}

func (rm *RouteManager) removeOldRoutes() error {
	routes, err := netctl.ListManagedRoutes(CustomRouteProtocol)
	if err != nil {
		return fmt.Errorf("error getting managed routes: %w", err)
	}

	rm.RouteTable.Lock()
	defer rm.RouteTable.Unlock()
	routesMap := make(map[string]struct{})
	for _, route := range rm.RouteTable.GetRoutes() {
		routesMap[route.Destination.String()] = struct{}{}
	}

	for _, route := range routes {
		if _, ok := routesMap[route.Dst.String()]; !ok {
			// route is outdated, remove it
			if err := netctl.RemoveRoute(route.Dst, rm.customRouteProtocol); err != nil {
				return fmt.Errorf("error removing route for network %s: %w", route.Dst, err)
			}
			rm.logger.Infof("old route removed for network %s via %s", route.Dst, route.Gw)
		}
	}

	return nil
}
