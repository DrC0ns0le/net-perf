package nexus

import "github.com/DrC0ns0le/net-perf/internal/system"

type Server interface {
	Start() error
	Stop() error
}

type Nexus struct {
	Servers []Server
}

func Serve(global *system.Node) {
	n := &Nexus{
		Servers: []Server{
			NewGRPCServer(global),
			NewHTTPServer(global),
			NewSocketServer(global),
		},
	}

	for _, s := range n.Servers {
		go s.Start()
	}

	<-global.StopCh

	for _, s := range n.Servers {
		s.Stop()
	}
}
