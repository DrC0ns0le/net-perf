package main

import (
	"flag"
	"os"
	"os/signal"
	"strconv"
	"syscall"

	"github.com/DrC0ns0le/net-perf/internal/measure"
	"github.com/DrC0ns0le/net-perf/internal/nexus"
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

type DaemonService interface {
	Start() error
}

func main() {

	flag.Parse()

	node := &system.Node{
		StopCh:          make(chan struct{}),
		WGUpdateCh:      make(chan netctl.WGInterface, *updateChBufSize),
		RTUpdateCh:      make(chan struct{}, *updateChBufSize),
		MeasureUpdateCh: make(chan struct{}, *updateChBufSize),

		RouteTable: &system.RouteTable{
			Routes: make([]system.Route, 0),
		},

		Logger: logging.NewDefaultLogger(),
	}

	node.Logger.Infof("starting net-perf daemon")

	err := tunables.Init()
	if err != nil {
		node.Logger.Fatalf("failed to configure init sysctls: %v", err)
	}

	siteID, err := netctl.GetLocalID()
	if err != nil {
		node.Logger.Fatalf("failed to get local id: %v", err)
	}

	node.SiteID, err = strconv.Atoi(siteID)
	if err != nil {
		node.Logger.Fatalf("failed to convert local id to int: %v", err)
	}

	service := []DaemonService{
		nexus.NewNexus(node),
		measure.NewManager(node),
		route.NewRouteManager(node),
	}

	for _, s := range service {
		go func(s DaemonService) {
			err = s.Start()
			if err != nil {
				node.Logger.Errorf("failed to start service: %v", err)
			}
		}(s)
	}

	// start watchdog
	go watchdog.Start(node)

	// wait for termination signal
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
}
