package netctl

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/DrC0ns0le/net-perf/internal/route"
)

type WGInterface struct {
	Name      string
	LocalID   string
	RemoteID  string
	IPVersion string
}

func GetOutgoingWGInterface(dst string) string {

	routes, err := GetWGRouteTable()
	if err != nil {
		return ""
	}

	for _, route := range routes {
		ipAddr := strings.Split(route.Destination.String(), ".")
		if ipAddr[1] == "201" && ipAddr[2] == dst && ipAddr[3] == "0" {
			return route.Iface
		}
	}

	return ""
}

// GetWGRouteTable reads the /proc/net/route file and returns a slice of Route objects, each
// representing a route in the table that is related to a WireGuard interface.
// Ignores any lines that do not correspond to a WireGuard interface.
func GetWGRouteTable() ([]route.Route, error) {
	file, err := os.Open("/proc/net/route")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var routes []route.Route
	scanner := bufio.NewScanner(file)

	// Skip the header line
	scanner.Scan()

	for scanner.Scan() {
		fields := strings.Fields(scanner.Text())
		if len(fields) < 8 {
			continue
		}

		if len(fields[0]) < 2 || fields[0][:2] != "wg" {
			continue
		}

		dest, err := parseIP(fields[1])
		if err != nil {
			continue
		}

		gateway, err := parseIP(fields[2])
		if err != nil {
			continue
		}

		flags, err := strconv.ParseInt(fields[3], 16, 32)
		if err != nil {
			continue
		}

		mask, err := parseIP(fields[7])
		if err != nil {
			continue
		}

		routes = append(routes, route.Route{
			Destination: dest,
			Gateway:     gateway,
			Flags:       int(flags),
			Iface:       fields[0],
			Mask:        net.IPMask(mask),
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return routes, nil
}

func ParseWGInterface(iface string) (WGInterface, error) {
	wgPattern := regexp.MustCompile(`^wg(\d+)\.(\d+)_v(\d+)$`)

	matches := wgPattern.FindStringSubmatch(iface)

	if len(matches) != 4 {
		return WGInterface{}, fmt.Errorf("error parsing wireguard interface name: %s", iface)
	}

	return WGInterface{
		Name:      iface,
		LocalID:   matches[1],
		RemoteID:  matches[2],
		IPVersion: matches[3],
	}, nil
}

func GetAllWGInterfaces() ([]WGInterface, error) {
	wgPattern := regexp.MustCompile(`^wg(\d+)\.(\d+)_v(\d+)$`)

	interfaces, err := net.Interfaces()
	if err != nil {
		fmt.Printf("Error getting interfaces: %v\n", err)
		return nil, err
	}

	wgIfs := []WGInterface{}

	for _, iface := range interfaces {
		if len(iface.Name) > 2 && iface.Name[:2] == "wg" {
			// parse the interface name
			matches := wgPattern.FindStringSubmatch(iface.Name)

			if len(matches) != 4 {
				fmt.Printf("Error parsing interface name: %s\n", iface.Name)
				continue
			}

			wgIfs = append(wgIfs, WGInterface{
				Name:      iface.Name,
				LocalID:   matches[1],
				RemoteID:  matches[2],
				IPVersion: matches[3],
			})
		}
	}

	return wgIfs, nil
}

func GetLocalID() (string, error) {

	wgPattern := regexp.MustCompile(`^wg(\d+)\.(\d+)_v(\d+)$`)

	interfaces, err := net.Interfaces()
	if err != nil {
		fmt.Printf("Error getting interfaces: %v\n", err)
		return "", err
	}

	for _, iface := range interfaces {
		if len(iface.Name) > 2 && iface.Name[:2] == "wg" {
			// parse the interface name
			matches := wgPattern.FindStringSubmatch(iface.Name)

			if len(matches) != 4 {
				fmt.Printf("Error parsing interface name: %s\n", iface.Name)
				continue
			}

			return matches[1], nil
		}
	}

	return "", fmt.Errorf("wg interface not found")
}

func GetLocalLoopbackIP() ([]string, error) {
	interfaces, err := net.Interfaces()
	if err != nil {
		return nil, fmt.Errorf("error getting interfaces: %v", err)
	}

	lID, err := GetLocalID()
	if err != nil {
		return nil, fmt.Errorf("error getting local ID: %v", err)
	}

	localID, err := strconv.Atoi(lID)
	if err != nil {
		return nil, fmt.Errorf("error parsing local ID: %v", err)
	}

	var loopbackIPs []string
	for _, iface := range interfaces {
		if iface.Flags&net.FlagLoopback != 0 {
			addrs, err := iface.Addrs()
			if err != nil {
				return nil, fmt.Errorf("error getting addresses for interface %s: %v", iface.Name, err)
			}

			for _, addr := range addrs {
				ipNet, ok := addr.(*net.IPNet)
				if !ok {
					continue
				}
				ip := ipNet.IP.To4()
				if ip != nil && ip[0] == 10 && ip[1] == 201 && ip[2] == byte(localID) {
					if ip[3] == 4 || ip[3] == 6 {
						loopbackIPs = append(loopbackIPs, ip.String())
					}

				}
			}
		}
	}

	if len(loopbackIPs) == 0 {
		return nil, fmt.Errorf("no 10.201.x.x loopback address found")
	}

	return loopbackIPs, nil
}

func parseIP(hexIP string) (net.IP, error) {
	rawIP, err := strconv.ParseUint(hexIP, 16, 32)
	if err != nil {
		return nil, err
	}
	ip := make(net.IP, 4)
	ip[0] = byte(rawIP & 0xFF)
	ip[1] = byte((rawIP >> 8) & 0xFF)
	ip[2] = byte((rawIP >> 16) & 0xFF)
	ip[3] = byte((rawIP >> 24) & 0xFF)
	return ip, nil
}