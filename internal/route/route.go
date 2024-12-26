package route

import (
	"hash"
	"log"

	"github.com/DrC0ns0le/net-perf/internal/route/bird"
	"github.com/DrC0ns0le/net-perf/internal/system"
	"github.com/DrC0ns0le/net-perf/pkg/logging"
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
		RouteTable: &RouteTable{
			Routes: make([]Route, 0),
		},
	}
	go bird.Watcher()

	// start route alignment
	Aligner := &Aligner{
		RouteTable: bird.RouteTable,
		RTCache:    make(map[string]hash.Hash64),
		RTUpdateCh: global.RTUpdateCh,
	}
	go Aligner.Start()
}
