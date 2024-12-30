package system

import (
	"github.com/DrC0ns0le/net-perf/internal/system/netctl"
	"github.com/DrC0ns0le/net-perf/pkg/logging"
)

type Node struct {
	StopCh     chan struct{}
	WGUpdateCh chan netctl.WGInterface
	RTUpdateCh chan struct{}

	RouteTable *RouteTable

	MeasureUpdateCh chan struct{}

	SiteID  int
	LocalIP string

	Logger logging.Logger
}
