package route

import (
	"log"

	"github.com/DrC0ns0le/net-perf/internal/route/bird"
	"github.com/DrC0ns0le/net-perf/pkg/logging"

	"github.com/DrC0ns0le/net-perf/internal/system"
)

func Start(global *system.Node) {

	config, err := bird.ParseBirdConfig("/etc/bird/bird.conf")
	if err != nil {
		logging.Errorf("Error parsing BIRD config: %v\n", err)
	}

	if config.ASNumber != global.SiteID+64512 {
		log.Fatalf("AS number mismatch: %v != %v", config.ASNumber, global.SiteID+64512)
	}

	// start route updates
	bird := &Bird{
		Config:       &config,
		GlobalStopCh: global.GlobalStopCh,
		RTUpdateCh:   global.RTUpdateCh,
		RouteTable:   global.RouteTable,
	}
	go bird.Watcher()
}
