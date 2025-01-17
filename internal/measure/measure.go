package measure

import (
	"fmt"
	"sync"
	"time"

	"github.com/DrC0ns0le/net-perf/internal/measure/bandwidth"
	"github.com/DrC0ns0le/net-perf/internal/measure/pathping"
	"github.com/DrC0ns0le/net-perf/internal/system"
	"github.com/DrC0ns0le/net-perf/internal/system/netctl"
	"github.com/DrC0ns0le/net-perf/pkg/logging"
)

type Worker struct {
	// Interface to use
	iface netctl.WGInterface
	// Source IP address
	sourceIP string
	// Target IP address
	targetIP string
	// Target port
	targetPort int

	// Stop channel
	stopCh chan struct{}

	// Logger
	logger logging.Logger
}

type Config struct {
	SourceIP   string
	TargetIP   string
	TargetPort int

	Logger logging.Logger
}

type Service interface {
	Start() error
}

type ServiceManager struct {
	workers map[string]*Worker

	errCh           chan error
	stopCh          chan struct{}
	measureUpdateCh chan struct{}

	// to manage starting and stopping of measurement services
	services map[string]Service

	logger logging.Logger
}

func (m *ServiceManager) manageWorkers() {
	manageWorkers := func() {
		ifaces, err := netctl.GetAllWGInterfaces()
		if err != nil {
			m.logger.Errorf("error getting interfaces: %v", err)
		}

		for i, w := range m.workers {
			found := false
			for _, iface := range ifaces {
				if iface.Name == i {
					found = true
					break
				}
			}
			if !found {
				m.logger.Infof("wg interface %s no longer exists, stopping worker", i)
				close(w.stopCh)
				delete(m.workers, i)
			}
		}

		for _, iface := range ifaces {
			if _, ok := m.workers[iface.Name]; !ok {
				m.logger.Infof("found new WG interface: %s", iface.Name)
				worker := &Worker{
					iface:      iface,
					stopCh:     make(chan struct{}),
					sourceIP:   fmt.Sprintf("10.201.%s.%s", iface.LocalID, iface.IPVersion),
					targetIP:   fmt.Sprintf("10.201.%s.%s", iface.RemoteID, iface.IPVersion),
					targetPort: 22,
					logger:     m.logger.With("worker", iface.Name),
				}
				m.workers[iface.Name] = worker
				go startLatencyWorker(worker)
				go startBandwidthWorker(worker)
				if iface.IPVersion == "4" {
					// Run one path latency worker per target
					go startPathLatencyWorker(worker) // TO-DO: improve this, dont assume all sites have IPv4
				}
			}
		}
	}

	manageWorkers()
	ticker := time.NewTicker(5 * time.Second)
	for {
		select {
		case <-ticker.C:
			manageWorkers()
		case <-m.measureUpdateCh:
			manageWorkers()
		case <-m.stopCh:
			m.logger.Info("stopping measurement workers...")
			for _, w := range m.workers {
				close(w.stopCh)
			}
			return
		}
	}
}

func NewManager(global *system.Node) *ServiceManager {
	sm := &ServiceManager{
		workers: make(map[string]*Worker),
		logger:  global.Logger.With("component", "measure"),
		stopCh:  global.StopCh,
		services: map[string]Service{
			"bandwidth": bandwidth.NewServer(global),
			"pathping":  pathping.NewServer(global),
		},
	}

	sm.errCh = make(chan error, len(sm.services))

	return sm

}

func (m *ServiceManager) Start() error {
	var wg sync.WaitGroup
	wg.Add(len(m.services))

	for _, s := range m.services {
		go func(s Service) {
			defer wg.Done()
			if err := s.Start(); err != nil {
				m.errCh <- err
			}
		}(s)
	}

	// Close the error channel when all services have started
	go func() {
		wg.Wait()
		close(m.errCh)
	}()

	// Collect all errors
	var errors []error
	for err := range m.errCh {
		errors = append(errors, err)
	}

	go m.manageWorkers()

	if len(errors) > 0 {
		return fmt.Errorf("failed to start services: %v", errors)
	}

	return nil
}

func generatePathName(src, dst string) string {
	if src > dst {
		src, dst = dst, src
	}
	return fmt.Sprintf("%s-%s", src, dst)
}
