package server

import (
	"fmt"
	"sync"

	"github.com/DrC0ns0le/net-perf/internal/system"
	"github.com/DrC0ns0le/net-perf/pkg/logging"
)

type Server interface {
	Start() error
	Stop() error
}

type ServerManager struct {
	stopCh  chan struct{}
	errCh   chan error
	servers []Server
	logger  logging.Logger
}

func NewServerManager(global *system.Node) *ServerManager {
	return &ServerManager{
		stopCh: global.StopCh,
		servers: []Server{
			NewGRPCServer(global),
			NewHTTPServer(global),
			NewSocketServer(global),
			NewConcensusServer(global),
		},
		logger: global.Logger,
	}
}

func (n *ServerManager) Start() error {
	// Start all servers
	n.errCh = make(chan error, len(n.servers))
	for _, s := range n.servers {
		go func(s Server) {
			if err := s.Start(); err != nil {
				n.errCh <- err
			}
		}(s)
	}

	for {
		select {
		case err := <-n.errCh:
			// Log the error but continue running other servers
			n.logger.Errorf("error starting server: %v", err)

		case <-n.stopCh:
			n.logger.Info("received stop signal, shutting down servers")

			// Create a separate error channel for stop errors to avoid mixing with start errors
			stopErrCh := make(chan error, len(n.servers))

			var wg sync.WaitGroup
			for _, s := range n.servers {
				wg.Add(1)
				go func(s Server) {
					defer wg.Done()
					if err := s.Stop(); err != nil {
						n.logger.Errorf("error stopping server: %v", err)
						stopErrCh <- err
					}
				}(s)
			}

			// Wait for all stop attempts to complete
			wg.Wait()
			close(stopErrCh)

			// Collect stop errors
			var stopErrors []error
			for err := range stopErrCh {
				stopErrors = append(stopErrors, err)
			}

			if len(stopErrors) > 0 {
				return fmt.Errorf("errors occurred while stopping servers: %v", stopErrors)
			}

			return nil
		}
	}
}
