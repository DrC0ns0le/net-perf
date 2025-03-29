package watchdog

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/DrC0ns0le/net-perf/internal/metrics"
	"github.com/DrC0ns0le/net-perf/internal/system/netctl"
	"github.com/DrC0ns0le/net-perf/internal/system/tunables"
	"github.com/DrC0ns0le/net-perf/pkg/logging"
	"github.com/vishvananda/netlink"
)

type linkWatchdog struct {
	stopCh          chan struct{}
	wgUpdateCh      chan netctl.WGInterface
	rtUpdateCh      chan struct{}
	measureUpdateCh chan struct{}

	started bool
	localID int

	logger logging.Logger
}

var (
	linkUpdateInterval = flag.Duration("watchdog.link.updateinterval", 60*time.Second, "interval for link interface updates")
)

func (w *linkWatchdog) Start() {
	w.logger.Infof("starting link watchdog service")
	ticker := time.NewTicker(*linkUpdateInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			go func() {
				updated, err := w.manageLink()
				if err != nil {
					w.logger.Errorf("failed to update routes: %v", err)
				}

				if updated {
					w.rtUpdateCh <- struct{}{}
				}
			}()
		case iface := <-w.wgUpdateCh:
			go func() {
				err := w.addLink(iface)
				if err != nil {
					w.logger.Errorf("failed to handle new interface: %v", err)
				}
			}()

			time.Sleep(150 * time.Millisecond)

			w.measureUpdateCh <- struct{}{}
			w.rtUpdateCh <- struct{}{}

		case <-w.stopCh:
			return
		}
	}
}

func (w *linkWatchdog) manageLink() (bool, error) {

	var updated bool

	ifaces, err := netctl.GetAllWGInterfaces()
	if err != nil {
		return false, fmt.Errorf("failed to get WireGuard interfaces: %v", err)
	}

	var remoteVersionMap = map[string][]string{}

	for _, iface := range ifaces {
		remoteVersionMap[iface.RemoteID] = append(remoteVersionMap[iface.RemoteID], iface.IPVersion)
	}

	var version string
	for remote, versions := range remoteVersionMap {
		version = ""
		if len(versions) > 1 {

			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()

			rID, err := strconv.Atoi(remote)
			if err != nil {
				return false, fmt.Errorf("failed to convert remote ID to int: %v", err)
			}

			version, _, err = metrics.GetPreferredPath(ctx, w.localID, rID)
			if err != nil {
				var netErr *net.OpError
				switch {
				case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded), errors.As(err, &netErr):
					w.logger.Errorf("timed out getting preferred interface version for %d, assuming v4", rID)
					version = "4"
				case errors.Is(err, metrics.ErrNoPaths):
					// means prometheus has no metrics
					w.logger.Infof("neither v4 nor v6 are preferred for %d, assuming v4", rID)
					version = "4"
				default:
					return false, fmt.Errorf("failed to get preferred interface version: %v", err)
				}
			}

			if !w.started {
				version = "4"
				w.started = true
			} else if version == "" {
				w.logger.Infof("neither v4 nor v6 are preferred for %s, skipping", remote)
				continue
			}
		} else {
			version = versions[0]
		}

		preferredInterface := "wg" + strconv.Itoa(w.localID) + "." + remote + "_v" + version

		// check which interface is being used
		iface := netctl.GetOutgoingWGInterface(remote)

		if iface == "" {
			w.logger.Errorf("warning: route for %s is unexpectedly not found in the routing table, adding route via %s", remote, preferredInterface)
			err := w.addWGLinkRoutes(preferredInterface)
			if err != nil {
				return false, fmt.Errorf("failed to add route for %s: %v", remote, err)
			}
			err = tunables.ConfigureInterface(preferredInterface)
			if err != nil {
				w.logger.Errorf("error configuring sysctls for %s: %v\n", preferredInterface, err)
			}
		} else if iface != preferredInterface {
			w.logger.Infof("changing route for %s from %s to %s", remote, iface, preferredInterface)
			err := w.changeWGLinkRoutes(preferredInterface)
			if err != nil {
				return false, fmt.Errorf("failed to change route for %s: %v", remote, err)
			}
		} else {
			w.logger.Debugf("Route for %s already set to preferred interface %s", remote, iface)
		}
	}

	return updated, nil
}

func (w *linkWatchdog) addLink(iface netctl.WGInterface) error {

	err := tunables.ConfigureInterface(iface.Name)
	if err != nil {
		w.logger.Errorf("error configuring sysctls for %s: %v\n", iface, err)
	}

	remoteID, err := strconv.Atoi(iface.RemoteID)
	if err != nil {
		return fmt.Errorf("failed to convert remote ID to int: %w", err)
	}

	err = w.addRoutes(remoteID)
	if err != nil {
		return err
	}
	return nil
}

func (w *linkWatchdog) addRoutes(remoteID int) error {

	w.logger.Debugf("adding route for %d", remoteID)

	var err error

	iface := "wg" + strconv.Itoa(w.localID) + "." + strconv.Itoa(remoteID) + "_v"
	version := "4"

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	version, _, err = metrics.GetPreferredPath(ctx, w.localID, remoteID)
	if err != nil {
		var netErr *net.OpError
		switch {
		case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded), errors.As(err, &netErr):
			w.logger.Debugf("timed out getting preferred interface version for %d, assuming v4", remoteID)
		case errors.Is(err, metrics.ErrNoPaths):
			w.logger.Debugf("neither v4 nor v6 are preferred for %d, assuming v4", remoteID)
		default:
			return fmt.Errorf("failed to get preferred interface version: %v", err)
		}
	}

	err = w.addWGLinkRoutes(iface + version)
	if err != nil {
		return err
	}

	return nil
}

func (w *linkWatchdog) addWGLinkRoutes(iface string) error {
	wgIface, err := netctl.ParseWGInterface(iface)
	if err != nil {
		return fmt.Errorf("error parsing interface %s: %w", iface, err)
	}

	remoteID, err := strconv.Atoi(wgIface.RemoteID)
	if err != nil {
		return fmt.Errorf("failed to convert remote ID to int: %w", err)
	}

	// Get the interface
	link, err := netlink.LinkByName(iface)
	if err != nil {
		return fmt.Errorf("error getting interface %s: %w", iface, err)
	}

	// IPv4 route configuration
	dst4, err := netlink.ParseIPNet(fmt.Sprintf("10.201.%d.0/24", remoteID))
	if err != nil {
		return fmt.Errorf("error parsing IPv4 CIDR: %w", err)
	}

	src4 := net.ParseIP(fmt.Sprintf("10.201.%d.1", w.localID))
	if src4 == nil {
		return fmt.Errorf("error parsing IPv4 source address")
	}

	route4 := &netlink.Route{
		LinkIndex: link.Attrs().Index,
		Dst:       dst4,
		Src:       src4,
		Scope:     netlink.SCOPE_LINK,
	}

	if err := netlink.RouteAdd(route4); err != nil {
		return fmt.Errorf("error adding IPv4 route: %w", err)
	}

	// IPv6 route configuration
	dst6, err := netlink.ParseIPNet(fmt.Sprintf("fdac:c9:%x::/64", remoteID))
	if err != nil {
		return fmt.Errorf("error parsing IPv6 CIDR: %w", err)
	}

	route6 := &netlink.Route{
		LinkIndex: link.Attrs().Index,
		Dst:       dst6,
		Scope:     netlink.SCOPE_LINK,
		Family:    netlink.FAMILY_V6,
	}

	if err := netlink.RouteAdd(route6); err != nil {
		return fmt.Errorf("error adding IPv6 route: %w", err)
	}

	return nil
}

func (w *linkWatchdog) changeWGLinkRoutes(iface string) error {

	wgIface, err := netctl.ParseWGInterface(iface)
	if err != nil {
		return fmt.Errorf("error parsing interface %s: %w", iface, err)
	}

	remoteID, err := strconv.Atoi(wgIface.RemoteID)
	if err != nil {
		return fmt.Errorf("failed to convert remote ID to int: %w", err)
	}

	// Get the interface
	link, err := netlink.LinkByName(iface)
	if err != nil {
		return fmt.Errorf("error getting interface %s: %w", iface, err)
	}

	// IPv4 route configuration
	dst4, err := netlink.ParseIPNet(fmt.Sprintf("10.201.%d.0/24", remoteID))
	if err != nil {
		return fmt.Errorf("error parsing IPv4 CIDR: %w", err)
	}

	src4 := net.ParseIP(fmt.Sprintf("10.201.%d.1", w.localID))
	if src4 == nil {
		return fmt.Errorf("error parsing IPv4 source address")
	}

	route4 := &netlink.Route{
		LinkIndex: link.Attrs().Index,
		Dst:       dst4,
		Src:       src4,
		Scope:     netlink.SCOPE_LINK,
	}

	if err := netlink.RouteReplace(route4); err != nil {
		return fmt.Errorf("error replacing IPv4 route: %w", err)
	}

	// IPv6 route configuration
	dst6, err := netlink.ParseIPNet(fmt.Sprintf("fdac:c9:%x::/64", remoteID))
	if err != nil {
		return fmt.Errorf("error parsing IPv6 CIDR: %w", err)
	}

	route6 := &netlink.Route{
		LinkIndex: link.Attrs().Index,
		Dst:       dst6,
		Scope:     netlink.SCOPE_LINK,
		Family:    netlink.FAMILY_V6,
	}

	if err := netlink.RouteReplace(route6); err != nil {
		return fmt.Errorf("error replacing IPv6 route: %w", err)
	}

	return nil
}
