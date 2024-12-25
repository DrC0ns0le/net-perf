package route

import (
	"net"
	"sync"
)

type RouteTable struct {
	Routes []Route

	mu       sync.RWMutex
	updateMu sync.Mutex
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
