package netctl

import (
	"fmt"
	"net"

	"github.com/vishvananda/netlink"
)

// CreateRouteTable creates a new routing table with the specified table ID and rules.
// tableID should be a unique identifier between 1-252 (253-255 are reserved).
// The function returns an error if the operation fails.
func CreateRouteTable(tableID int, rules []RouteTableRule) error {
	if tableID < 1 || tableID > 252 {
		return fmt.Errorf("invalid table ID: must be between 1 and 252")
	}

	// Create rules to direct traffic to the new table
	for _, rule := range rules {
		if err := validateRule(rule); err != nil {
			return fmt.Errorf("invalid rule configuration: %v", err)
		}

		nlRule := &netlink.Rule{
			Table:    tableID,
			Priority: rule.Priority,
			Family:   getIPFamily(rule.Src),
		}

		// Set source network if specified
		if rule.Src != nil {
			nlRule.Src = rule.Src
		}

		// Set destination network if specified
		if rule.Dst != nil {
			nlRule.Dst = rule.Dst
		}

		// Add the rule to the routing policy database
		if err := netlink.RuleAdd(nlRule); err != nil {
			return fmt.Errorf("failed to add rule: %v", err)
		}
	}

	return nil
}

// RouteTableRule represents a routing rule configuration
type RouteTableRule struct {
	Priority int        // Priority of the rule (lower values are processed first)
	Src      *net.IPNet // Source network (optional)
	Dst      *net.IPNet // Destination network (optional)
	Mark     int        // Firewall mark (optional)
	Mask     int        // Mask for firewall mark (optional)
}

// DeleteRouteTable removes a routing table and all associated rules.
func DeleteRouteTable(tableID int) error {
	if tableID < 1 || tableID > 252 {
		return fmt.Errorf("invalid table ID: must be between 1-252")
	}

	// List all rules
	rules, err := netlink.RuleList(netlink.FAMILY_ALL)
	if err != nil {
		return fmt.Errorf("failed to list rules: %v", err)
	}

	// Delete rules associated with this table
	for _, rule := range rules {
		if rule.Table == tableID {
			if err := netlink.RuleDel(&rule); err != nil {
				return fmt.Errorf("failed to delete rule: %v", err)
			}
		}
	}

	// Delete all routes in this table
	filter := &netlink.Route{
		Table: tableID,
	}
	routes, err := netlink.RouteListFiltered(netlink.FAMILY_ALL, filter, netlink.RT_FILTER_TABLE)
	if err != nil {
		return fmt.Errorf("failed to list routes: %v", err)
	}

	for _, route := range routes {
		if err := netlink.RouteDel(&route); err != nil {
			return fmt.Errorf("failed to delete route from table %d: %v", tableID, err)
		}
	}

	return nil
}

// Helper function to validate rule configuration
func validateRule(rule RouteTableRule) error {
	if rule.Priority < 0 {
		return fmt.Errorf("priority must be non-negative")
	}

	if rule.Src != nil && rule.Dst != nil {
		// Verify IP versions match if both are specified
		srcIsV4 := rule.Src.IP.To4() != nil
		dstIsV4 := rule.Dst.IP.To4() != nil
		if srcIsV4 != dstIsV4 {
			return fmt.Errorf("source and destination IP versions must match")
		}
	}

	return nil
}

// Helper function to determine IP family
func getIPFamily(network *net.IPNet) int {
	if network == nil {
		return netlink.FAMILY_V4 // Default to IPv4
	}
	if network.IP.To4() != nil {
		return netlink.FAMILY_V4
	}
	return netlink.FAMILY_V6
}
