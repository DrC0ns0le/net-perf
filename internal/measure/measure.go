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

type site struct {
	siteID string

	interfaceWorkersMu sync.Mutex
	interfaceWorkers   map[string]*Worker

	stopCh     chan struct{}
	siteWorker *Worker

	wg     *sync.WaitGroup
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
	sitesMu sync.Mutex
	sites   map[int]*site

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
			m.logger.Errorf("failed managing workers: %v", err)
			return
		}

		var (
			siteFound            bool
			interfaceWorkerFound bool
		)
		m.sitesMu.Lock()
		defer m.sitesMu.Unlock()
		// handle removals
		for i, s := range m.sites {
			siteFound = false
			interfaceWorkerFound = false
			for _, iface := range ifaces {
				if iface.RemoteIDInt == i {
					siteFound = true
					if _, ok := s.interfaceWorkers[iface.Name]; ok {
						interfaceWorkerFound = true
						break
					}
				}
			}

			if !siteFound {
				s.logger.Infof("removing site: %s", s.siteID)
				s.stop()
				delete(m.sites, i)
			} else if !interfaceWorkerFound {
				s.interfaceWorkersMu.Lock()
				s.logger.Infof("removing interface worker: %s", s.interfaceWorkers[s.siteWorker.iface.Name].iface.Name)
				close(s.interfaceWorkers[s.siteWorker.iface.Name].stopCh)
				delete(s.interfaceWorkers, s.siteWorker.iface.Name)
				s.interfaceWorkersMu.Unlock()
			}
		}

		// handle additions
		for _, iface := range ifaces {
			// check if site exists
			if _, ok := m.sites[iface.RemoteIDInt]; !ok {
				m.logger.Infof("found new site: %s", iface.RemoteIDInt)
				m.sites[iface.RemoteIDInt] = &site{
					siteID:           iface.RemoteID,
					interfaceWorkers: make(map[string]*Worker),
					stopCh:           make(chan struct{}),
					siteWorker: &Worker{
						iface:      iface,
						stopCh:     make(chan struct{}),
						sourceIP:   fmt.Sprintf("10.201.%s.1", iface.LocalID),
						targetIP:   fmt.Sprintf("10.201.%s.1", iface.RemoteID),
						targetPort: 22,
						logger:     m.logger.With("worker", iface.Name),
					},
					wg:     &sync.WaitGroup{},
					logger: m.logger.With("site", iface.RemoteIDInt),
				}

				// create new interface worker
				m.sites[iface.RemoteIDInt].interfaceWorkersMu.Lock()
				m.sites[iface.RemoteIDInt].interfaceWorkers[iface.Name] = &Worker{
					iface:      iface,
					stopCh:     make(chan struct{}),
					sourceIP:   fmt.Sprintf("10.201.%s.%s", iface.LocalID, iface.IPVersion),
					targetIP:   fmt.Sprintf("10.201.%s.%s", iface.RemoteID, iface.IPVersion),
					targetPort: 22,
					logger:     m.sites[iface.RemoteIDInt].logger.With("worker", "site-"+iface.RemoteID),
				}

				// start site worker(s)
				m.sites[iface.RemoteIDInt].wg.Add(1)
				go func(site *site) {
					defer site.wg.Done()
					startPathLatencyWorker(site.siteWorker)
				}(m.sites[iface.RemoteIDInt])

				// start interface worker(s)
				m.sites[iface.RemoteIDInt].wg.Add(2)
				go func(site *site, ifaceName string) {
					defer site.wg.Done()
					startBandwidthWorker(site.interfaceWorkers[ifaceName])
				}(m.sites[iface.RemoteIDInt], iface.Name)
				go func(site *site, ifaceName string) {
					defer site.wg.Done()
					startLatencyWorker(site.interfaceWorkers[ifaceName])
				}(m.sites[iface.RemoteIDInt], iface.Name)
				m.sites[iface.RemoteIDInt].interfaceWorkersMu.Unlock()

			} else {
				// check if interface worker exists
				m.sites[iface.RemoteIDInt].interfaceWorkersMu.Lock()
				if _, ok := m.sites[iface.RemoteIDInt].interfaceWorkers[iface.Name]; !ok {
					m.logger.Infof("found new interface: %s", iface.Name)
					m.sites[iface.RemoteIDInt].interfaceWorkers[iface.Name] = &Worker{
						iface:      iface,
						stopCh:     make(chan struct{}),
						sourceIP:   fmt.Sprintf("10.201.%s.%s", iface.LocalID, iface.IPVersion),
						targetIP:   fmt.Sprintf("10.201.%s.%s", iface.RemoteID, iface.IPVersion),
						targetPort: 22,
						logger:     m.sites[iface.RemoteIDInt].logger.With("worker", iface.Name),
					}

					// start interface worker(s)
					m.sites[iface.RemoteIDInt].wg.Add(2)
					go func(site *site, ifaceName string) {
						defer site.wg.Done()
						startBandwidthWorker(site.interfaceWorkers[ifaceName])
					}(m.sites[iface.RemoteIDInt], iface.Name)
					go func(site *site, ifaceName string) {
						defer site.wg.Done()
						startLatencyWorker(site.interfaceWorkers[ifaceName])
					}(m.sites[iface.RemoteIDInt], iface.Name)
				}
				m.sites[iface.RemoteIDInt].interfaceWorkersMu.Unlock()
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
			for _, s := range m.sites {
				s.stop()
			}
			return
		}
	}
}

func NewManager(global *system.Node) *ServiceManager {
	sm := &ServiceManager{
		sites:           make(map[int]*site),
		logger:          global.Logger.With("component", "measure"),
		stopCh:          global.StopCh,
		measureUpdateCh: global.MeasureUpdateCh,
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

func (s *site) stop() {
	s.interfaceWorkersMu.Lock()
	defer s.interfaceWorkersMu.Unlock()
	for _, w := range s.interfaceWorkers {
		close(w.stopCh)
	}

	done := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		s.logger.Warnf("site cleanup timed out, some goroutines may not have finished")
	}

	close(s.stopCh)
}

func generatePathName(src, dst string) string {
	if src > dst {
		src, dst = dst, src
	}
	return fmt.Sprintf("%s-%s", src, dst)
}
