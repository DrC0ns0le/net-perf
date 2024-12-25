package system

import (
	"log"

	"github.com/DrC0ns0le/net-perf/internal/system/netctl"
	"github.com/DrC0ns0le/net-perf/internal/system/tunables"
)

type Node struct {
	GlobalStopCh chan struct{}
	WGUpdateCh   chan netctl.WGInterface
	RTUpdateCh   chan struct{}

	MeasureUpdateCh chan struct{}

	SiteID  int
	LocalIP string
}

func init() {
	err := tunables.Init()
	if err != nil {
		log.Panicf("failed to configure init sysctls: %v", err)
	}
}
