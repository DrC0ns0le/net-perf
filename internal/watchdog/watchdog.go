package watchdog

import (
	"hash"

	"github.com/DrC0ns0le/net-perf/internal/system"
	"github.com/DrC0ns0le/net-perf/pkg/logging"
)

type watchdog struct {
	link  *linkWatchdog
	route *routeWatchdog
}

func Start(global *system.Node) {

	w := &watchdog{
		link: &linkWatchdog{
			StopCh:          global.GlobalStopCh,
			WGUpdateCh:      global.WGUpdateCh,
			RTUpdateCh:      global.RTUpdateCh,
			MeasureUpdateCh: global.MeasureUpdateCh,
			localID:         global.SiteID,
		},
		route: &routeWatchdog{
			RouteTable: global.RouteTable,
			RTCache:    make(map[string]hash.Hash64),
			RTUpdateCh: global.RTUpdateCh,
		},
	}

	err := Serve(global)
	if err != nil {
		logging.Errorf("failed to start watchdog socket: %v", err)
	}

	// link watchdog
	go w.link.Start()

	// route watchdog
	go w.route.Start()
}
