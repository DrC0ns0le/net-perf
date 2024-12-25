package route

import (
	"errors"
	"flag"
	"time"

	"github.com/DrC0ns0le/net-perf/internal/system/netctl"
	"github.com/DrC0ns0le/net-perf/pkg/logging"
)

var (
	alignerInterval = flag.Duration("aligner.interval", 5*time.Second, "Interval to check for missing routes")
)

type Aligner struct {
	RouteTable *RouteTable

	RTUpdateCh chan struct{}
}

func (a *Aligner) Start() {

	time.Sleep(10 * time.Second)

	logging.Infof("Starting route alignment service")

	ticker := time.NewTicker(*alignerInterval)
	defer ticker.Stop()
	for {
		<-ticker.C
		a.check()
	}
}

func (a *Aligner) check() {
	a.RouteTable.updateMu.Lock()
	defer a.RouteTable.updateMu.Unlock()

	var needUpdate bool

	for _, route := range a.RouteTable.Routes {
		if _, err := netctl.GetRoute(route.Destination, route.Gateway, nil); err == errors.New("no matching route found") {
			logging.Errorf("Missing route %s via %s", route.Destination, route.Gateway)
			needUpdate = true
		}
	}

	if needUpdate {
		logging.Info("Triggering route table update due to missing routes")
		a.RTUpdateCh <- struct{}{}
	}

}
