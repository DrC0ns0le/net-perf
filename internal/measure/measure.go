package measure

import (
	"fmt"
	"time"

	"github.com/DrC0ns0le/net-perf/internal/system"
	"github.com/DrC0ns0le/net-perf/internal/system/netctl"
	"github.com/DrC0ns0le/net-perf/pkg/logging"
)

type Worker struct {
	// Interface to use
	iface netctl.WGInterface
	// Source IP address
	sourceIP string
	// Target IP address
	targetIP string
	// Target port
	targetPort int

	// Stop channel
	stopCh chan struct{}
}

var workerMap = make(map[string]*Worker)

func Start(global *system.Node) {
	manageWorkers := func() {
		ifaces, err := netctl.GetAllWGInterfaces()
		if err != nil {
			logging.Errorf("Error getting interfaces: %v\n", err)
		}

		for i, w := range workerMap {
			found := false
			for _, iface := range ifaces {
				if iface.Name == i {
					found = true
					break
				}
			}
			if !found {
				logging.Infof("Interface %s no longer exists, stopping worker\n", i)
				close(w.stopCh)
				delete(workerMap, i)
			}
		}

		for _, iface := range ifaces {
			if _, ok := workerMap[iface.Name]; !ok {
				logging.Infof("Found new WG interface: %s\n", iface.Name)
				worker := &Worker{
					iface:      iface,
					stopCh:     make(chan struct{}),
					sourceIP:   fmt.Sprintf("10.201.%s.%s", iface.LocalID, iface.IPVersion),
					targetIP:   fmt.Sprintf("10.201.%s.%s", iface.RemoteID, iface.IPVersion),
					targetPort: 22,
				}
				workerMap[iface.Name] = worker
				go startLatencyWorker(worker)
				go startBandwidthWorker(worker)
			}
		}
	}

	manageWorkers()
	ticker := time.NewTicker(5 * time.Second)
	for {
		select {
		case <-ticker.C:
			manageWorkers()
		case <-global.MeasureUpdateCh:
			manageWorkers()
		case <-global.GlobalStopCh:
			stop()
			return
		}
	}
}

func stop() {
	logging.Info("Stopping measurement workers...")
	for _, w := range workerMap {
		close(w.stopCh)
	}
}

func generatePathName(src, dst string) string {
	if src > dst {
		src, dst = dst, src
	}
	return fmt.Sprintf("%s-%s", src, dst)
}
