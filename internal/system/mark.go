package system

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

const (
	ruleLabel = "AUTO_GENERATED_ROUTE"
)

// Config holds the configuration for route setup
type Config struct {
	LocalID    string
	TotalSites int
}

// RouteManager handles the setup and verification of routes
type RouteManager struct {
	config Config
}

// NewRouteManager creates a new RouteManager instance
func NewRouteManager(config Config) (*RouteManager, error) {
	if config.LocalID == "" {
		return nil, fmt.Errorf("localID cannot be empty")
	}
	if config.TotalSites <= 0 {
		return nil, fmt.Errorf("totalSites must be greater than 0")
	}
	return &RouteManager{config: config}, nil
}

// SetupAllRoutes sets up routes for all sites
func (rm *RouteManager) SetupAllRoutes() error {
	for i := 0; i < rm.config.TotalSites; i++ {
		if err := rm.SetupRoutesForSite(i); err != nil {
			return fmt.Errorf("failed to setup routes for site %d: %v", i, err)
		}
	}
	return nil
}

// SetupRoutesForSite sets up routes for a specific site index
func (rm *RouteManager) SetupRoutesForSite(siteIndex int) error {
	// Check and setup default route
	device := fmt.Sprintf("wg%s.%d_v4", rm.config.LocalID, siteIndex)
	table := fmt.Sprintf("rt_site%d", siteIndex)

	exists, err := rm.checkRouteExists(device, table)
	if err != nil {
		return fmt.Errorf("failed to check route existence: %v", err)
	}

	if !exists {
		if err := rm.executeCommand("ip", []string{
			"-6",
			"route",
			"add",
			"default",
			"dev",
			device,
			"table",
			table,
		}); err != nil {
			return fmt.Errorf("failed to add route: %v", err)
		}
	}

	// Check and setup routing rule
	exists, err = rm.checkRuleExists(strconv.Itoa(siteIndex), table)
	if err != nil {
		return fmt.Errorf("failed to check rule existence: %v", err)
	}

	if !exists {
		if err := rm.executeCommand("ip", []string{
			"-6",
			"rule",
			"add",
			"to",
			fmt.Sprintf("fdac:c9:%s::/64", rm.config.LocalID),
			"from",
			"::/0",
			"fwmark",
			strconv.Itoa(siteIndex),
			"lookup",
			table,
		}); err != nil {
			return fmt.Errorf("failed to add rule: %v", err)
		}
	}

	// Setup ip6tables rules for different TTLs
	ttlRules := []struct {
		ttl  int
		byte int
	}{
		{4, 32},
		{3, 33},
		{2, 34},
		{1, 35},
	}

	for _, rule := range ttlRules {
		exists, err := rm.checkIP6TableRuleExists(rule.ttl, rule.byte, siteIndex)
		if err != nil {
			return fmt.Errorf("failed to check ip6tables rule existence: %v", err)
		}

		if !exists {
			if err := rm.executeCommand("ip6tables", []string{
				"-t", "mangle",
				"-A", "PREROUTING",
				"-p", "icmpv6",
				"-m", "ttl",
				"--ttl-eq", strconv.Itoa(rule.ttl),
				"-m", "u32",
				"--u32", fmt.Sprintf("%d&0xFF=%d", rule.byte, siteIndex),
				"-j", "MARK",
				"--set-mark", strconv.Itoa(siteIndex),
				"-m", "comment",
				"--comment", fmt.Sprintf("%s_%d_%d", ruleLabel, rule.ttl, siteIndex),
			}); err != nil {
				return fmt.Errorf("failed to add ip6tables rule: %v", err)
			}
		}
	}

	return nil
}

func (rm *RouteManager) checkRouteExists(device, table string) (bool, error) {
	output, err := rm.executeCommandWithOutput("ip", []string{"-6", "route", "show", "table", table})
	if err != nil {
		return false, fmt.Errorf("failed to check route: %v", err)
	}

	return strings.Contains(output, fmt.Sprintf("dev %s", device)), nil
}

func (rm *RouteManager) checkRuleExists(fwmark, table string) (bool, error) {
	output, err := rm.executeCommandWithOutput("ip", []string{"-6", "rule", "show"})
	if err != nil {
		return false, fmt.Errorf("failed to check rule: %v", err)
	}

	return strings.Contains(output, fmt.Sprintf("fwmark %s lookup %s", fwmark, table)), nil
}

func (rm *RouteManager) checkIP6TableRuleExists(ttl, byte, siteIndex int) (bool, error) {
	output, err := rm.executeCommandWithOutput("ip6tables",
		[]string{"-t", "mangle", "-L", "PREROUTING", "-n", "--line-numbers"})
	if err != nil {
		return false, fmt.Errorf("failed to check ip6tables rule: %v", err)
	}

	searchStr := fmt.Sprintf("ttl eq %d u32 %d&0xFF=%d /* %s_%d_%d */",
		ttl, byte, siteIndex, ruleLabel, ttl, siteIndex)
	return strings.Contains(output, searchStr), nil
}

func (rm *RouteManager) executeCommand(name string, args []string) error {
	cmd := exec.Command(name, args...)
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("command failed: %s, output: %s", err, output)
	}
	return nil
}

func (rm *RouteManager) executeCommandWithOutput(name string, args []string) (string, error) {
	cmd := exec.Command(name, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("command failed: %s, output: %s", err, output)
	}
	return string(output), nil
}
