package system

import (
	"log"

	"github.com/DrC0ns0le/net-perf/internal/system/netctl"
	"github.com/DrC0ns0le/net-perf/internal/system/tunables"
)

type Node struct {
	GlobalStopCh chan struct{}
	WGChangeCh   chan netctl.WGInterface
	RTChangeCh   chan struct{}

	MeasureWatchDogCh chan struct{}

	SiteID int
}

func init() {
	err := tunables.Init()
	if err != nil {
		log.Panicf("failed to configure init sysctls: %v", err)
	}
}
