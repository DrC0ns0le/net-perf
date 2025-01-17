package netctl

import (
	"fmt"
	"net"

	"github.com/vishvananda/netlink"
	"golang.org/x/sys/unix"
)

// ConfigureRoute configures a route for the specified network.
// Returns 0 if no route changes were made, 1 if a new route was added, and 2 if an existing route was updated.
func ConfigureRoute(dst *net.IPNet, gw, src net.IP, proto netlink.RouteProtocol) (int, error) {
	if dst == nil {
		return 0, fmt.Errorf("network cannot be nil")
	}
	if gw == nil {
		return 0, fmt.Errorf("gateway cannot be nil")
	}

	// Validate IP versions match
	if (dst.IP.To4() != nil) != (gw.To4() != nil) {
		return 0, fmt.Errorf("destination and gateway IP versions must match")
	}
	if src != nil && (gw.To4() != nil) != (src.To4() != nil) {
		return 0, fmt.Errorf("source and gateway IP versions must match")
	}

	route := &netlink.Route{
		Dst:      dst,
		Gw:       gw,
		Protocol: proto,
		Src:      src,
	}

	// List all routes matching our destination
	filter := &netlink.Route{
		Dst: dst,
	}
	existing, err := netlink.RouteListFiltered(netlink.FAMILY_ALL, filter, netlink.RT_FILTER_DST)
	if err != nil {
		return 0, fmt.Errorf("failed to list routes: %v", err)
	}

	// Check for existing managed route
	for _, r := range existing {
		if r.Protocol == proto {
			// Check if parameters need updating
			if r.Gw.Equal(gw) &&
				((src == nil && r.Src == nil) || (src != nil && r.Src != nil && r.Src.Equal(src))) {
				// No changes needed
				return 0, nil
			}
			// Update existing route
			if err := netlink.RouteReplace(route); err != nil {
				return 0, fmt.Errorf("failed to update route: %v", err)
			}
			return 2, nil
		}
	}

	err = netlink.RouteAdd(route)
	if err != nil {
		if err == unix.EEXIST {
			return 2, netlink.RouteReplace(route)
		}
		return 0, fmt.Errorf("failed to add route: %v", err)
	}

	return 1, nil
}

// RemoveRoute removes a route from the Linux routing table given a network.
// Only removes routes that were created by this package (identified by CustomRouteProtocol).
// Returns an error if the operation fails.
func RemoveRoute(dst *net.IPNet, proto netlink.RouteProtocol) error {
	if dst == nil {
		return fmt.Errorf("network cannot be nil")
	}

	// Create a route object with the destination and protocol
	route := &netlink.Route{
		Dst:      dst,
		Protocol: proto,
	}

	// Try to delete the route directly without checking existence
	err := netlink.RouteDel(route)
	if err != nil {
		// Only return error if it's not "not exists" error
		if err != unix.ESRCH {
			return fmt.Errorf("failed to remove route: %v", err)
		}
		return nil
	}

	return nil
}

// RouteExists checks if a route to the specified network exists.
// Only checks for routes managed by this package (identified by CustomRouteProtocol).
// If src is provided, only checks for routes with matching source IP.
// Returns true if the route exists, false otherwise.
func RouteExists(dst *net.IPNet, src net.IP, proto netlink.RouteProtocol) (bool, error) {
	if dst == nil {
		return false, fmt.Errorf("network cannot be nil")
	}

	// Try to get the route directly
	routes, err := netlink.RouteGet(dst.IP)
	if err != nil {
		return false, fmt.Errorf("failed to get route: %v", err)
	}

	// Check if any of the returned routes match our criteria
	for _, r := range routes {
		if r.Protocol == proto {
			// If source IP is specified, check if it matches
			if src != nil {
				if r.Src != nil && r.Src.Equal(src) {
					return true, nil
				}
			} else {
				return true, nil
			}
		}
	}

	return false, nil
}

// ListManagedRoutes returns a list of all routes managed by this package
// (identified by CustomRouteProtocol).
func ListManagedRoutes(proto netlink.RouteProtocol) ([]netlink.Route, error) {
	// Get all routes but filter by our protocol in the kernel
	filter := &netlink.Route{
		Protocol: proto,
	}

	routes, err := netlink.RouteListFiltered(netlink.FAMILY_ALL, filter, netlink.RT_FILTER_PROTOCOL)
	if err != nil {
		return nil, fmt.Errorf("failed to list routes: %v", err)
	}

	return routes, nil
}

// GetRoute returns a specific route matching the destination and optionally the gateway and/or source IP.
func GetRoute(dst *net.IPNet, gw, src net.IP, proto netlink.RouteProtocol) (*netlink.Route, error) {
	if dst == nil {
		return nil, fmt.Errorf("network cannot be nil")
	}

	routes, err := netlink.RouteGet(dst.IP)
	if err != nil {
		return nil, fmt.Errorf("failed to get route: %v", err)
	}

	for _, r := range routes {
		if r.Protocol == proto {
			// If neither gw nor src specified, return first matching route
			if gw == nil && src == nil {
				return &r, nil
			}

			// Check gateway match if specified
			gwMatch := gw == nil || (r.Gw != nil && r.Gw.Equal(gw))
			// Check source match if specified
			srcMatch := src == nil || (r.Src != nil && r.Src.Equal(src))

			// Return route if both specified conditions match
			if gwMatch && srcMatch {
				return &r, nil
			}
		}
	}
	return nil, ErrRouteNotFound
}

// RemoveAllManagedRoutes deletes all routes managed by this package
// (identified by CustomRouteProtocol).
// Returns the number of routes removed and any error encountered.
func RemoveAllManagedRoutes(proto netlink.RouteProtocol) (int, error) {
	var failedRoutes []netlink.Route
	// Get list of all managed routes
	routes, err := ListManagedRoutes(proto)
	if err != nil {
		return 0, fmt.Errorf("failed to list managed routes: %v", err)
	}

	// Keep track of successfully removed routes
	removed := 0

	// Remove each route
	for _, route := range routes {
		err := netlink.RouteDel(&route)
		if err != nil {
			// Log error but continue trying to remove other routes
			failedRoutes = append(failedRoutes, route)
			continue
		}
		removed++
	}

	if len(failedRoutes) > 0 {
		return removed, fmt.Errorf("failed to remove %d routes: %v", len(failedRoutes), failedRoutes)
	}

	return removed, nil
}
