package routers

import (
	"hash"
	"net"
)

type Route struct {
	// Network is the network of the route
	Network *net.IPNet
	// Paths is the list of BGP paths
	Paths []BGPPath
	// OriginAS is the AS number of the origin of the route
	OriginAS int
}
type BGPPath struct {
	AS              int
	ASPath          []int
	Next            net.IP
	Interface       string
	MED             int
	LocalPreference int
	OriginType      string
}

type Router interface {
	GetRoutes(mode string) ([]Route, hash.Hash64, error)
	GetConfig(mode string) (Config, error)
}
