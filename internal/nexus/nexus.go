package nexus

import (
	"fmt"
	"sync"

	"github.com/DrC0ns0le/net-perf/internal/system"
)

type Server interface {
	Start() error
	Stop() error
}

type Nexus struct {
	stopCh  chan struct{}
	errCh   chan error
	servers []Server
}

func NewNexus(global *system.Node) *Nexus {
	return &Nexus{
		stopCh: global.StopCh,
		servers: []Server{
			NewGRPCServer(global),
			NewHTTPServer(global),
			NewSocketServer(global),
		},
	}
}

func (n *Nexus) Start() error {
	var wg sync.WaitGroup
	wg.Add(len(n.servers))

	// Start all servers
	for _, s := range n.servers {
		go func(s Server) {
			defer wg.Done()
			if err := s.Start(); err != nil {
				n.errCh <- err
			}
		}(s)
	}

	// Close error channel when all servers have started
	go func() {
		wg.Wait()
		close(n.errCh)
	}()

	// Collect start-up errors
	var errors []error
	for err := range n.errCh {
		errors = append(errors, err)
	}

	// If there were any start-up errors, return them
	if len(errors) > 0 {
		return fmt.Errorf("failed to start servers: %v", errors)
	}

	// Wait for stop signal
	<-n.stopCh

	// Stop all servers
	wg.Add(len(n.servers))
	for _, s := range n.servers {
		go func(s Server) {
			defer wg.Done()
			if err := s.Stop(); err != nil {
				n.errCh <- err
			}
		}(s)
	}

	// Close error channel when all servers have stopped
	go func() {
		wg.Wait()
		close(n.errCh)
	}()

	// Collect stop errors
	errors = nil
	for err := range n.errCh {
		errors = append(errors, err)
	}

	// If there were any stop errors, return them
	if len(errors) > 0 {
		return fmt.Errorf("errors occurred while stopping servers: %v", errors)
	}

	return nil
}
