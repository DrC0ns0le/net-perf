package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/DrC0ns0le/net-perf/internal/management"
	"github.com/DrC0ns0le/net-perf/internal/measure"
	"github.com/DrC0ns0le/net-perf/internal/measure/bandwidth"
	"github.com/DrC0ns0le/net-perf/internal/metrics"
	"github.com/DrC0ns0le/net-perf/internal/route"
	"github.com/DrC0ns0le/net-perf/internal/system"
	"github.com/DrC0ns0le/net-perf/internal/system/netctl"
	"github.com/DrC0ns0le/net-perf/internal/system/tunables"
	"github.com/DrC0ns0le/net-perf/internal/watchdog"
	"github.com/DrC0ns0le/net-perf/pkg/logging"

	_ "github.com/DrC0ns0le/net-perf/internal/system"
)

var (
	updateChBufSize = flag.Int("wg.updatech", 0, "channel buffer size for wg interface updates")
)

func main() {

	logging.Infof("Starting net-perf daemon")

	flag.Parse()

	err := tunables.Init()
	if err != nil {
		log.Panicf("failed to configure init sysctls: %v", err)
	}

	node := &system.Node{
		GlobalStopCh:    make(chan struct{}),
		WGUpdateCh:      make(chan netctl.WGInterface, *updateChBufSize),
		RTUpdateCh:      make(chan struct{}, *updateChBufSize),
		MeasureUpdateCh: make(chan struct{}, *updateChBufSize),

		RouteTable: &system.RouteTable{
			Routes: make([]system.Route, 0),
		},
	}

	siteID, err := netctl.GetLocalID()
	if err != nil {
		log.Fatalf("failed to get local id: %v", err)
	}

	node.SiteID, err = strconv.Atoi(siteID)
	if err != nil {
		log.Fatalf("failed to convert local id to int: %v", err)
	}

	// start metrics server
	go metrics.Serve()

	// start bandwidth measurement server
	go bandwidth.Serve()

	// start measurement workers for bandwidth & latency
	go measure.Start(node)

	// route management
	go route.Start(node)

	// start management rpc server
	go management.Serve()

	// start watchdog
	go watchdog.Start(node)

	// wait for termination signal
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
}
