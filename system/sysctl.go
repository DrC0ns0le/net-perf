package system

import (
	"fmt"
	"net"
	"os/exec"
	"strings"

	"github.com/DrC0ns0le/net-perf/utils"
)

func ConfigureInterfaceSysctls(iface string) error {
	iface = strings.Replace(iface, ".", "/", -1)
	sysctls := map[string]string{
		fmt.Sprintf("net.ipv4.conf.%s.forwarding", iface):     "1",
		fmt.Sprintf("net.ipv4.conf.%s.accept_local", iface):   "1",
		fmt.Sprintf("net.ipv4.conf.%s.route_localnet", iface): "1",
		fmt.Sprintf("net.ipv4.conf.%s.rp_filter", iface):      "0",
		fmt.Sprintf("net.ipv6.conf.%s.forwarding", iface):     "1",
	}

	// Set accept_ra based on interface type
	if strings.HasPrefix(iface, "wg") {
		sysctls[fmt.Sprintf("net.ipv6.conf.%s.accept_ra", iface)] = "0"
	} else {
		sysctls[fmt.Sprintf("net.ipv6.conf.%s.accept_ra", iface)] = "2"
	}

	// Apply interface-specific settings
	for key, value := range sysctls {
		if err := setSysctl(key, value); err != nil {
			return fmt.Errorf("failed to set interface sysctl %s: %v", key, err)
		}
	}

	return nil
}

func configureGlobalSysctls() error {
	sysctls := map[string]string{
		"net.ipv4.conf.all.accept_local":       "1",
		"net.ipv4.conf.all.route_localnet":     "1",
		"net.ipv4.conf.default.accept_local":   "1",
		"net.ipv4.conf.default.route_localnet": "1",
		"net.ipv4.conf.all.rp_filter":          "0",
		"net.ipv4.conf.default.rp_filter":      "0",
		"net.ipv6.conf.all.forwarding":         "1",
	}

	// Apply global settings
	for key, value := range sysctls {
		if err := setSysctl(key, value); err != nil {
			return fmt.Errorf("failed to set global sysctl %s: %v", key, err)
		}
	}

	return nil
}

func configureInitSysctls() error {
	if err := configureGlobalSysctls(); err != nil {
		return err
	}
	// Get WireGuard interfaces
	wgIfaces, err := utils.GetAllWGInterfaces()
	if err != nil {
		return fmt.Errorf("failed to get WireGuard interfaces: %v", err)
	}
	for _, iface := range wgIfaces {
		if err := ConfigureInterfaceSysctls(iface.Name); err != nil {
			return err
		}
	}

	// Get network interfaces
	netIfaces, err := net.Interfaces()
	if err != nil {
		return fmt.Errorf("failed to get network interfaces: %v", err)
	}
	for _, iface := range netIfaces {
		if strings.HasPrefix(iface.Name, "en") {
			if err := ConfigureInterfaceSysctls(iface.Name); err != nil {
				return err
			}
		}
	}

	return nil
}

func setSysctl(name string, value string) error {
	return exec.Command("sysctl", "-w", fmt.Sprintf("%s=%s", name, value)).Run()
}
