package system

import (
	"github.com/DrC0ns0le/net-perf/internal/system/netctl"
	"github.com/DrC0ns0le/net-perf/pkg/logging"
)

type Node struct {
	StopCh          chan struct{}
	WGUpdateCh      chan netctl.WGInterface
	RTUpdateCh      chan struct{}
	MeasureUpdateCh chan struct{}

	RouteTable *RouteTable
	StateTable *StateTable

	Services         map[string]Service
	RouteService     RouteInterface
	ConsensusService ConsensusInterface
	MeasureService   MeasureInterface

	SiteID  int
	LocalIP string
	Peers   []int

	Logger logging.Logger
}

type Service interface {
	Start() error
}

type RouteInterface interface {
	GetCentralisedRoute(int) int
	GetGraphRoute(int) int
	UpdateLocalRoutes(map[int]int)
}

type ConsensusInterface interface {
	Leader() bool
	Healty() bool
}

type MeasureInterface interface {
	PathPing(path []int) (float64, error)
}
