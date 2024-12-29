package system

import (
	"net"
	"sync"
)

type RouteTable struct {
	Routes []Route

	mu       sync.RWMutex
	updateMu sync.Mutex

	ready bool
}

type Route struct {
	Destination *net.IPNet
	Gateway     net.IP
}

func (rt *RouteTable) AddRoute(destination *net.IPNet, gateway net.IP) {
	rt.mu.Lock()
	rt.Routes = append(rt.Routes, Route{Destination: destination, Gateway: gateway})
	rt.mu.Unlock()
}

func (rt *RouteTable) GetRoutes() []Route {
	rt.mu.RLock()
	defer rt.mu.RUnlock()
	return rt.Routes
}

func (rt *RouteTable) ClearRoutes() {
	rt.mu.Lock()
	rt.Routes = []Route{}
	rt.mu.Unlock()
}

func (rt *RouteTable) Lock() {
	rt.updateMu.Lock()
}

func (rt *RouteTable) Unlock() {
	rt.updateMu.Unlock()
}

func (rt *RouteTable) Ready() bool {
	rt.mu.RLock()
	defer rt.mu.RUnlock()
	return rt.ready
}

func (rt *RouteTable) MarkReady() {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	rt.ready = true
}
