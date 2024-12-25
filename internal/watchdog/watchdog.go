package watchdog

import (
	"flag"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/DrC0ns0le/net-perf/internal/system"
	"github.com/DrC0ns0le/net-perf/internal/system/netctl"
	"github.com/DrC0ns0le/net-perf/internal/system/tunables"
	"github.com/DrC0ns0le/net-perf/pkg/logging"
)

type watchdog struct {
	localID int
}

var (
	wgInterfaceUpdateInterval = flag.Duration("wg.updateinterval", 60*time.Second, "interval for wg interface updates")
)

func Start(global *system.Node) {

	w := &watchdog{
		localID: global.SiteID,
	}

	logging.Infof("starting watchdog service")

	err := Serve(global)
	if err != nil {
		logging.Errorf("failed to start watchdog socket: %v", err)
	}

	err = updateRoutes(true, global.SiteID)
	if err != nil {
		log.Fatalf("failed to update routes: %v", err)
	}

	var iface netctl.WGInterface

	ticker := time.NewTicker(*wgInterfaceUpdateInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			go func() {
				err := updateRoutes(false, global.SiteID)
				if err != nil {
					logging.Errorf("failed to update routes: %v", err)
				}
			}()
		case iface = <-global.WGChangeCh:
			go func() {
				err := w.handleNewInterface(iface.Name)
				if err != nil {
					logging.Errorf("failed to handle new interface: %v", err)
				}
			}()

		case <-global.GlobalStopCh:
			return
		}
	}
}

func (w *watchdog) handleNewInterface(iface string) error {

	err := tunables.ConfigureInterface(iface)
	if err != nil {
		logging.Errorf("Error configuring sysctls for %s: %v\n", iface, err)
	}

	wgIface, err := netctl.ParseWGInterface(iface)
	if err != nil {
		return err
	}

	remoteID, err := strconv.Atoi(wgIface.RemoteID)
	if err != nil {
		return fmt.Errorf("failed to convert remote ID to int: %w", err)
	}

	err = addLinkRoute(w.localID, remoteID)
	if err != nil {
		return err
	}
	return nil
}
