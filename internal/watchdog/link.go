package watchdog

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os/exec"
	"strconv"
	"time"

	"github.com/DrC0ns0le/net-perf/internal/metrics"
	"github.com/DrC0ns0le/net-perf/internal/system/netctl"
	"github.com/DrC0ns0le/net-perf/internal/system/tunables"
	"github.com/DrC0ns0le/net-perf/pkg/logging"
	"github.com/vishvananda/netlink"
)

func updateRoutes(justStarted bool, localID int) (bool, error) {

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

			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()

			rID, err := strconv.Atoi(remote)
			if err != nil {
				return false, fmt.Errorf("failed to convert remote ID to int: %v", err)
			}

			version, _, err = metrics.GetPreferredPath(ctx, localID, rID)
			if err != nil {
				var netErr *net.OpError
				switch {
				case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded), errors.As(err, &netErr):
					logging.Errorf("timed out getting preferred interface version for %d, assuming v4", rID)
					version = "4"
				case errors.Is(err, metrics.ErrNoPaths):
					// means prometheus has no metrics
					logging.Infof("neither v4 nor v6 are preferred for %d, assuming v4", rID)
					version = "4"
				default:
					return false, fmt.Errorf("failed to get preferred interface version: %v", err)
				}
			}

			if justStarted {
				version = "4"
			} else if version == "" {
				logging.Infof("neither v4 nor v6 are preferred for %s, skipping", remote)
				continue
			}
		} else {
			version = versions[0]
		}

		preferredInterface := "wg" + strconv.Itoa(localID) + "." + remote + "_v" + version

		// check which interface is being used
		iface := netctl.GetOutgoingWGInterface(remote)

		if iface == "" {
			logging.Errorf("Warning:route for %s not found in the routing table, adding route via %s", remote, preferredInterface)
			// run "ip route add 10.201.{remote}.0/24 dev preferredInterface scope link src 10.201.{local}.1"
			err := exec.Command("ip", "route", "add", "10.201."+remote+".0/24", "dev", preferredInterface, "scope", "link", "src", "10.201."+strconv.Itoa(localID)+".1").Run()
			if err != nil {
				return false, fmt.Errorf("error executing ip add command: %w", err)
			}
			// run "ip -6 route add fdac:c9:{remote}::/64 dev preferredInterface scope link"
			err = exec.Command("ip", "-6", "route", "add", "fdac:c9:"+remote+"::/64", "dev", preferredInterface, "scope", "link").Run()
			if err != nil {
				return false, fmt.Errorf("error executing ip -6 add command: %w", err)
			}
			err = tunables.ConfigureInterface(preferredInterface)
			if err != nil {
				logging.Errorf("Error configuring sysctls for %s: %v\n", preferredInterface, err)
			}
		} else if iface != preferredInterface {
			// run "ip route change 10.201.{remote}.0/24 dev preferredInterface scope link"
			logging.Infof("changing route for %s from %s to %s", remote, iface, preferredInterface)
			err := exec.Command("ip", "route", "change", "10.201."+remote+".0/24", "dev", preferredInterface, "scope", "link", "src", "10.201."+strconv.Itoa(localID)+".1").Run()
			if err != nil {
				return false, fmt.Errorf("error executing ip change command: %w", err)
			}
			// run "ip -6 route change fdac:c9:{remote}::/64 dev preferredInterface scope link"
			err = exec.Command("ip", "-6", "route", "change", "fdac:c9:"+remote+"::/64", "dev", preferredInterface, "scope", "link").Run()
			if err != nil {
				return false, fmt.Errorf("error executing ip -6 change command: %w", err)
			}
		} else {
			logging.Debugf("Route for %s already set to preferred interface %s", remote, iface)
		}
	}

	return updated, nil
}

func addLinkRoute(localID int, remoteID int) error {

	logging.Debugf("adding route for %d", remoteID)

	iface := "wg" + strconv.Itoa(localID) + "." + strconv.Itoa(remoteID) + "_v"
	version := "4"

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	version, _, err := metrics.GetPreferredPath(ctx, localID, remoteID)
	if err != nil {
		var netErr *net.OpError
		switch {
		case errors.Is(err, context.Canceled), errors.Is(err, context.DeadlineExceeded), errors.As(err, &netErr):
			logging.Debugf("timed out getting preferred interface version for %d, assuming v4", remoteID)
		case errors.Is(err, metrics.ErrNoPaths):
			logging.Debugf("neither v4 nor v6 are preferred for %d, assuming v4", remoteID)
		default:
			return fmt.Errorf("failed to get preferred interface version: %v", err)
		}
	}

	err = addWGLinkRoutes(remoteID, localID, iface+version)
	if err != nil {
		return err
	}

	return nil
}

func addWGLinkRoutes(remoteID, localID int, iface string) error {
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

	src4 := net.ParseIP(fmt.Sprintf("10.201.%d.1", localID))
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
	dst6, err := netlink.ParseIPNet(fmt.Sprintf("fdac:c9:%d::/64", remoteID))
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
