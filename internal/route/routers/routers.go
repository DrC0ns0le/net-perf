package routers

import (
	"hash"
	"net"
)

type Route struct {
	Network  *net.IPNet
	Paths    []BGPPath
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
