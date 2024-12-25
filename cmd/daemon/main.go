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
	"github.com/DrC0ns0le/net-perf/internal/system"
	"github.com/DrC0ns0le/net-perf/internal/system/netctl"
	"github.com/DrC0ns0le/net-perf/internal/watchdog"

	_ "github.com/DrC0ns0le/net-perf/internal/system"
)

var (
	updateChBufSize = flag.Int("wg.updatech", 10, "channel size for wg interface updates")

	err error
)

func main() {
	flag.Parse()

	node := &system.Node{
		GlobalStopCh:    make(chan struct{}),
		WGUpdateCh:      make(chan netctl.WGInterface, *updateChBufSize),
		RTUpdateCh:      make(chan struct{}, *updateChBufSize),
		MeasureUpdateCh: make(chan struct{}, *updateChBufSize),
	}

	siteID, err := netctl.GetLocalID()
	if err != nil {
		log.Fatalf("failed to get local id: %v", err)
	}

	node.SiteID, err = strconv.Atoi(siteID)
	if err != nil {
		log.Fatalf("failed to convert local id to int: %v", err)
	}

	// start watchdog
	go watchdog.Start(node)

	// start metrics server
	go metrics.Serve()

	// start bandwidth measurement server
	go bandwidth.Serve()

	// start measurement workers for bandwidth & latency
	go measure.Start(node)

	// start management rpc server
	go management.Serve()

	// wait for termination signal
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
}
