package watchdog

import (
	"flag"
	"fmt"
	"hash"
	"time"

	"github.com/DrC0ns0le/net-perf/internal/route/bird"
	"github.com/DrC0ns0le/net-perf/internal/system"
	"github.com/DrC0ns0le/net-perf/internal/system/netctl"
	"github.com/DrC0ns0le/net-perf/pkg/logging"
)

var (
	routeUpdateInterval = flag.Duration("watchdog.route.updateinterval", 5*time.Second, "interval for missing route checks")
)

type routeWatchdog struct {
	StopCh     chan struct{}
	RTUpdateCh chan struct{}

	RouteTable *system.RouteTable
	RTCache    map[string]hash.Hash64
	needUpdate bool

	Logger logging.Logger
}

func (w *routeWatchdog) Start() {

	for !w.RouteTable.Ready() {
		time.Sleep(100 * time.Millisecond)
	}

	w.Logger.Infof("starting route watchdog service")

	ticker := time.NewTicker(*routeUpdateInterval)
	defer ticker.Stop()
	for {
		select {
		case <-w.StopCh:
			return
		case <-ticker.C:
			w.RouteTable.Lock()
			w.checkSystemRTAlignment()
			w.checkBirdChanges()
			w.RouteTable.Unlock()

			if w.needUpdate {
				w.needUpdate = false
				w.RTUpdateCh <- struct{}{}
			}
		}
	}
}

func (w *routeWatchdog) checkSystemRTAlignment() {
	systemRoutes, err := netctl.ListManagedRoutes()
	if err != nil {
		w.Logger.Errorf("failed to list managed routes: %v", err)
		return
	}

	managedRoutes := make(map[string]struct{})
	for _, route := range systemRoutes {
		key := fmt.Sprintf("%s_%v", route.Dst.String(), route.Gw)
		managedRoutes[key] = struct{}{}
	}

	for _, route := range w.RouteTable.Routes {
		key := fmt.Sprintf("%s_%v", route.Destination.String(), route.Gateway)
		if _, exists := managedRoutes[key]; !exists {
			w.Logger.Errorf("missing system route %s via %s",
				route.Destination, route.Gateway)
			w.needUpdate = true
		}
	}
}

func (w *routeWatchdog) checkBirdChanges() {
	for _, mode := range []string{"v4", "v6"} {
		_, hash, err := bird.GetRoutes(mode)
		if err != nil {
			w.Logger.Errorf("error getting routes: %v", err)
		}

		if _, ok := w.RTCache[mode]; !ok {
			w.RTCache[mode] = hash
		} else if w.RTCache[mode].Sum64() != hash.Sum64() {
			w.RTCache[mode] = hash
			w.Logger.Infof("bird %s route table changed", mode)
			w.needUpdate = true
		}
	}
}
