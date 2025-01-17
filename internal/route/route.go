package route

import (
	"fmt"

	"github.com/DrC0ns0le/net-perf/internal/route/bird"
	"github.com/DrC0ns0le/net-perf/internal/system"
	"github.com/DrC0ns0le/net-perf/pkg/logging"
)

type RouteManager struct {
	stopCh     chan struct{}
	rtUpdateCh chan struct{}
	routeTable *system.RouteTable

	siteID int

	logger logging.Logger
}

func NewRouteManager(global *system.Node) *RouteManager {
	return &RouteManager{
		stopCh:     global.StopCh,
		rtUpdateCh: global.RTUpdateCh,
		routeTable: global.RouteTable,
		logger:     global.Logger,
		siteID:     global.SiteID,
	}
}

func (m *RouteManager) Start() error {

	config, err := bird.ParseBirdConfig("/etc/bird/bird.conf")
	if err != nil {
		return err
	}

	if config.ASNumber != m.siteID+64512 {
		return fmt.Errorf("unexpected AS number: %d", config.ASNumber)
	}

	// start route updates
	bird := &Bird{
		Config:     &config,
		StopCh:     m.stopCh,
		RTUpdateCh: m.rtUpdateCh,
		RouteTable: m.routeTable,
		Logger:     m.logger.With("component", "bird"),
	}
	go bird.Watcher()

	return nil
}
