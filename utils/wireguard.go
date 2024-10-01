package utils

import (
	"fmt"
	"net"
	"regexp"
	"strconv"
)

type WGInterface struct {
	Name      string
	LocalID   string
	RemoteID  string
	IPVersion string
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
