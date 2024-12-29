package system

import "github.com/DrC0ns0le/net-perf/internal/system/netctl"

type Node struct {
	GlobalStopCh chan struct{}
	WGUpdateCh   chan netctl.WGInterface
	RTUpdateCh   chan struct{}

	RouteTable *RouteTable

	MeasureUpdateCh chan struct{}

	SiteID  int
	LocalIP string
}
