package watchdog

import (
	"flag"
	"fmt"
	"hash"
	"time"

	"github.com/DrC0ns0le/net-perf/internal/route"
	"github.com/DrC0ns0le/net-perf/internal/route/routers"
	"github.com/DrC0ns0le/net-perf/internal/system"
	"github.com/DrC0ns0le/net-perf/internal/system/netctl"
	"github.com/DrC0ns0le/net-perf/pkg/logging"
)

var (
	routeUpdateInterval = flag.Duration("watchdog.route.updateinterval", 5*time.Second, "interval for missing route checks")
	whitelistedRouteDev = []string{"cilium"}
)

type routeWatchdog struct {
	stopCh     chan struct{}
	rtUpdateCh chan struct{}

	router     routers.Router
	routeTable *system.RouteTable
	rtCache    map[string]hash.Hash64

	logger logging.Logger
}

func (w *routeWatchdog) Start() {

	for !w.routeTable.Ready() {
		time.Sleep(100 * time.Millisecond)
	}

	w.logger.Infof("starting route watchdog service")

	ticker := time.NewTicker(*routeUpdateInterval)
	defer ticker.Stop()
	for {
		select {
		case <-w.stopCh:
			return
		case <-ticker.C:
			w.routeTable.Lock()
			needUpdate := w.checkSystemRTAlignment() || w.checkRouterChanges()
			w.routeTable.Unlock()

			// only update if there is a change
			if needUpdate {
				needUpdate = false
				// signal must be sent after routeTable mutex is unlocked to prevent deadlock as route run will try to acquire the lock
				w.rtUpdateCh <- struct{}{}
			}
		}
	}
}

func (w *routeWatchdog) checkSystemRTAlignment() bool {
	systemRoutes, err := netctl.ListManagedRoutes(route.CustomRouteProtocol)
	if err != nil {
		w.logger.Errorf("failed to list managed routes: %v", err)
		return false
	}

	managedRoutes := make(map[string]struct{})
	for _, route := range systemRoutes {
		key := fmt.Sprintf("%s_%v", route.Dst.String(), route.Gw)
		managedRoutes[key] = struct{}{}
	}

routeCheck:
	for _, route := range w.routeTable.GetRoutes() {
		key := fmt.Sprintf("%s_%v", route.Destination.String(), route.Gateway)

		if _, exists := managedRoutes[key]; !exists {
			for _, wlD := range whitelistedRouteDev {
				managedByOthers, err := netctl.RouteDeviceHasPrefix(route.Destination, wlD, 0)
				if err != nil {
					w.logger.Errorf("failed to check if route is managed by %s: %v", wlD, err)
				}
				if managedByOthers {
					continue routeCheck
				}
			}

			w.logger.Errorf("missing system route %s via %s",
				route.Destination, route.Gateway)
			return true
		}
	}

	return false
}

func (w *routeWatchdog) checkRouterChanges() bool {
	needUpdate := false
	for _, mode := range []string{"v4", "v6"} {
		_, hash, err := w.router.GetRoutes(mode)
		if err != nil {
			w.logger.Errorf("error getting routes: %v", err)
		}

		if _, ok := w.rtCache[mode]; !ok {
			w.rtCache[mode] = hash
		} else if w.rtCache[mode].Sum64() != hash.Sum64() {
			w.rtCache[mode] = hash
			needUpdate = true
		}
	}

	if needUpdate {
		w.logger.Infof("bird routes changed")
	}

	return needUpdate
}
