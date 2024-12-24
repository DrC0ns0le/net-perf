package route

import (
	"net"
)

type Route struct {
	Destination net.IP
	Gateway     net.IP
	Flags       int
	Iface       string
	Mask        net.IPMask
}
