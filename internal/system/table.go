package system

import (
	"net"
	"sync"
)

type RouteTable struct {
	Routes []KernelRoute

	mu       sync.RWMutex
	updateMu sync.Mutex

	ready bool
}

type KernelRoute struct {
	Destination *net.IPNet
	Gateway     net.IP
}

func (rt *RouteTable) AddRoute(destination *net.IPNet, gateway net.IP) {
	rt.mu.Lock()
	rt.Routes = append(rt.Routes, KernelRoute{Destination: destination, Gateway: gateway})
	rt.mu.Unlock()
}

func (rt *RouteTable) GetRoutes() []KernelRoute {
	rt.mu.RLock()
	defer rt.mu.RUnlock()
	return rt.Routes
}

func (rt *RouteTable) ClearRoutes() {
	rt.mu.Lock()
	rt.Routes = []KernelRoute{}
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
