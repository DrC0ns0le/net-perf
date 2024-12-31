package watchdog

import (
	"hash"

	"github.com/DrC0ns0le/net-perf/internal/system"
)

type watchdog struct {
	link  *linkWatchdog
	route *routeWatchdog
}

func Start(global *system.Node) {

	w := &watchdog{
		link: &linkWatchdog{
			StopCh:          global.StopCh,
			WGUpdateCh:      global.WGUpdateCh,
			RTUpdateCh:      global.RTUpdateCh,
			MeasureUpdateCh: global.MeasureUpdateCh,
			localID:         global.SiteID,
			Logger:          global.Logger.With("component", "link"),
		},
		route: &routeWatchdog{
			StopCh:     global.StopCh,
			RouteTable: global.RouteTable,
			RTCache:    make(map[string]hash.Hash64),
			RTUpdateCh: global.RTUpdateCh,
			Logger:     global.Logger.With("component", "route"),
		},
	}

	// link watchdog
	go w.link.Start()

	// route watchdog
	go w.route.Start()
}
