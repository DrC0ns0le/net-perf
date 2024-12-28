package route

import (
	"flag"
	"fmt"
	"hash"
	"time"

	"github.com/DrC0ns0le/net-perf/internal/route/bird"
	"github.com/DrC0ns0le/net-perf/internal/system/netctl"
	"github.com/DrC0ns0le/net-perf/pkg/logging"
)

var (
	alignerInterval = flag.Duration("aligner.interval", 5*time.Second, "Interval to check for missing routes")
)

type Aligner struct {
	RouteTable *RouteTable
	RTCache    map[string]hash.Hash64

	needUpdate bool
	RTUpdateCh chan struct{}
}

func (a *Aligner) Start() {

	time.Sleep(5 * time.Second)

	logging.Infof("Starting route alignment service")

	ticker := time.NewTicker(*alignerInterval)
	defer ticker.Stop()
	for {
		<-ticker.C
		a.RouteTable.updateMu.Lock()
		a.checkSystemRTAlignment()
		a.checkBirdChanges()

		if a.needUpdate {
			a.needUpdate = false
			a.RTUpdateCh <- struct{}{}
		}

		a.RouteTable.updateMu.Unlock()
	}
}

func (a *Aligner) checkSystemRTAlignment() {
	systemRoutes, err := netctl.ListManagedRoutes()
	if err != nil {
		logging.Errorf("birdwatcher: Failed to list managed routes: %v", err)
		return
	}

	managedRoutes := make(map[string]struct{})
	for _, route := range systemRoutes {
		key := fmt.Sprintf("%s_%v", route.Dst.String(), route.Gw)
		managedRoutes[key] = struct{}{}
	}

	for _, route := range a.RouteTable.Routes {
		key := fmt.Sprintf("%s_%v", route.Destination.String(), route.Gateway)
		if _, exists := managedRoutes[key]; !exists {
			logging.Errorf("birdwatcher: Missing system route %s via %s",
				route.Destination, route.Gateway)
			a.needUpdate = true
		}
	}
}

func (a *Aligner) checkBirdChanges() {
	for _, mode := range []string{"v4", "v6"} {
		_, hash, err := bird.GetRoutes(mode)
		if err != nil {
			logging.Errorf("Error getting routes: %v", err)
		}

		if _, ok := a.RTCache[mode]; !ok {
			a.RTCache[mode] = hash
		} else if a.RTCache[mode].Sum64() != hash.Sum64() {
			a.RTCache[mode] = hash
			logging.Debugf("birdwatcher: Bird route table changed: %v", hash)
			a.needUpdate = true
			return
		}
	}
}
