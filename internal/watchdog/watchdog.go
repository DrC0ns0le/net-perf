package watchdog

import (
	"hash"

	"github.com/DrC0ns0le/net-perf/internal/route/routers/bird"
	"github.com/DrC0ns0le/net-perf/internal/system"
)

type watchdog struct {
	watchdogComponents []watchdogComponent
}

type watchdogComponent interface {
	Start()
}

func NewWatchdog(global *system.Node) *watchdog {
	return &watchdog{
		watchdogComponents: []watchdogComponent{
			NewLinkWatchdog(global),
			NewRouteWatchdog(global),
		},
	}
}

func NewLinkWatchdog(global *system.Node) watchdogComponent {
	return &linkWatchdog{
		stopCh:          global.StopCh,
		wgUpdateCh:      global.WGUpdateCh,
		rtUpdateCh:      global.RTUpdateCh,
		measureUpdateCh: global.MeasureUpdateCh,
		localID:         global.SiteID,
		logger:          global.Logger.With("component", "link"),
	}
}

func NewRouteWatchdog(global *system.Node) watchdogComponent {
	return &routeWatchdog{
		stopCh:     global.StopCh,
		router:     bird.NewRouter(),
		routeTable: global.RouteTable,
		rtCache:    make(map[string]hash.Hash64),
		rtUpdateCh: global.RTUpdateCh,

		logger: global.Logger.With("component", "route"),
	}
}

func Start(global *system.Node) {
	w := NewWatchdog(global)

	for _, c := range w.watchdogComponents {
		go c.Start()
	}
}
