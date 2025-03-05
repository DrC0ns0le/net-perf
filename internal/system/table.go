package system

import (
	"net"
	"sync"
)

type RouteTable struct {
	// Map with destination IP network string as key
	routeMap map[string]KernelRoute

	mu       sync.RWMutex
	updateMu sync.Mutex

	ready bool
}

type KernelRoute struct {
	Destination *net.IPNet
	Gateway     net.IP
}

// NewRouteTable creates a new RouteTable with initialized map
func NewRouteTable() *RouteTable {
	return &RouteTable{
		routeMap: make(map[string]KernelRoute),
		ready:    false,
	}
}

func (rt *RouteTable) AddRoute(destination *net.IPNet, gateway net.IP) {
	rt.mu.Lock()
	defer rt.mu.Unlock()

	// Add route to map (will overwrite if destination already exists)
	rt.routeMap[destination.String()] = KernelRoute{Destination: destination, Gateway: gateway}
}

func (rt *RouteTable) GetRoutes() []KernelRoute {
	rt.mu.RLock()
	defer rt.mu.RUnlock()

	routes := make([]KernelRoute, 0, len(rt.routeMap))
	for _, route := range rt.routeMap {
		routes = append(routes, route)
	}
	return routes
}

func (rt *RouteTable) ClearRoutes() {
	rt.mu.Lock()
	rt.routeMap = make(map[string]KernelRoute)
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
