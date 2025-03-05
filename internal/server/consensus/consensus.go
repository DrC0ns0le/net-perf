package consensus

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/DrC0ns0le/net-perf/internal/system"
	"github.com/DrC0ns0le/net-perf/pkg/logging"
	"github.com/hashicorp/raft"
)

type Server struct {
	local      int
	peers      []int
	listenAddr string
	port       int

	logger logging.Logger
	stopCh chan struct{}

	needBootstrap bool

	raft *raft.Raft

	// Current state tracking
	isCurrentlyLeader bool
}

func NewServer(global *system.Node, listenAddr string) *Server {
	splits := strings.Split(listenAddr, ":")
	port, err := strconv.Atoi(splits[len(splits)-1])
	if err != nil {
		port = 0
	}
	return &Server{
		local:      global.SiteID,
		peers:      global.Peers,
		listenAddr: listenAddr,
		port:       port,

		logger: global.Logger,
		stopCh: global.StopCh,
	}
}

func (s *Server) Start() error {
	if err := s.NewRaft(); err != nil {
		return fmt.Errorf("failed to initialize raft: %w", err)
	}

	// Check if we need to bootstrap the cluster
	if s.needBootstrap && s.local == 0 {
		s.logger.Info("Node 0 - bootstrapping new Raft cluster")
		configuration := raft.Configuration{
			Servers: []raft.Server{
				{
					ID:      raft.ServerID(fmt.Sprintf("%d", s.local)),
					Address: raft.ServerAddress(s.listenAddr),
				},
			},
		}

		if err := s.raft.BootstrapCluster(configuration).Error(); err != nil {
			return fmt.Errorf("failed to bootstrap raft cluster: %w", err)
		}
	}

	// Start leadership monitoring
	go s.monitorLeadership()

	return nil
}

// Stop gracefully stops the Raft instance
func (s *Server) Stop() error {
	// Check if raft is initialized
	if s.raft != nil {
		future := s.raft.Shutdown()
		return future.Error()
	}
	return nil
}
