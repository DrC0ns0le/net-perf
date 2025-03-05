// Implements Raft for the consensus
package consensus

import (
	"flag"
	"fmt"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/hashicorp/raft"
	boltdb "github.com/hashicorp/raft-boltdb"
)

var (
	raftDirectory = flag.String("raft.directory", "/opt/wg-mesh/raft", "path to raft directory")
)

func (s *Server) NewRaft() error {
	fsm := &FSM{}
	raftConfig := raft.DefaultConfig()
	raftConfig.LocalID = raft.ServerID(strconv.Itoa(s.local))
	raftConfig.LogLevel = "WARN"

	// Create transport
	addr, err := net.ResolveTCPAddr("tcp", s.listenAddr)
	if err != nil {
		return fmt.Errorf("error resolving TCP address: %w", err)
	}

	transport, err := raft.NewTCPTransport(s.listenAddr, addr, 3, 10*time.Second, os.Stderr)
	if err != nil {
		return fmt.Errorf("error creating TCP transport: %w", err)
	}

	// Store persistence
	if _, err := os.Stat(*raftDirectory); os.IsNotExist(err) {
		if err := os.MkdirAll(*raftDirectory, 0755); err != nil {
			return fmt.Errorf("error creating data directory: %w", err)
		}
		s.needBootstrap = true // first time bootstrap
	}
	logStore, err := boltdb.NewBoltStore(fmt.Sprintf("%s/raft-log.db", *raftDirectory))
	if err != nil {
		return fmt.Errorf("error creating log store: %w", err)
	}

	stableStore, err := boltdb.NewBoltStore(fmt.Sprintf("%s/raft-stable.db", *raftDirectory))
	if err != nil {
		return fmt.Errorf("error creating stable store: %w", err)
	}

	snapshotStore, err := raft.NewFileSnapshotStore(*raftDirectory, 3, os.Stderr)
	if err != nil {
		return fmt.Errorf("error creating snapshot store: %w", err)
	}

	s.raft, err = raft.NewRaft(raftConfig, fsm, logStore, stableStore, snapshotStore, transport)
	if err != nil {
		return fmt.Errorf("error creating raft instance: %w", err)
	}
	return nil
}

// monitorLeadership continuously monitors the Raft leadership state
// and sends signals when leadership status changes
func (s *Server) monitorLeadership() {
	// Create an observer channel that will be notified on leadership changes
	leadershipCh := s.raft.LeaderCh()

	// Initial state check
	s.isCurrentlyLeader = s.raft.State() == raft.Leader

	// Log initial state
	if s.isCurrentlyLeader {
		s.logger.Info("Initial state: Leader", "node", s.local)
		// Ensure all peers are in the cluster
		go s.ensureAllPeersInCluster()
	} else {
		s.logger.Info("Initial state: Follower", "node", s.local)
	}

	for {
		select {
		case isLeader := <-leadershipCh:
			if isLeader {
				if !s.isCurrentlyLeader {
					s.logger.Info("State changed: Became Leader", "node", s.local)
					s.isCurrentlyLeader = true

					// Ensure all peers are in the cluster
					go s.ensureAllPeersInCluster()

					s.leaderCallback()
				}
			} else {
				if s.isCurrentlyLeader {
					s.logger.Info("State changed: Became Follower", "node", s.local)
					s.isCurrentlyLeader = false

					s.followerCallback()
				}
			}
		case <-s.stopCh:
			s.logger.Info("Stopping leadership monitoring")
			return
		}
	}
}

// ensureAllPeersInCluster ensures all peers in the list are added to the Raft cluster
func (s *Server) ensureAllPeersInCluster() {
	// Wait a bit to ensure raft stability
	time.Sleep(2 * time.Second)

	// Get current configuration
	configFuture := s.raft.GetConfiguration()
	if err := configFuture.Error(); err != nil {
		s.logger.Error("Failed to get cluster configuration", "error", err)
		return
	}

	// Create a map of existing server IDs
	existingServers := make(map[string]bool)
	for _, server := range configFuture.Configuration().Servers {
		existingServers[string(server.ID)] = true
	}

	// Add all peers that are not already in the cluster
	for _, peerID := range s.peers {
		// Skip self
		if peerID == s.local {
			continue
		}

		// Check if peer is already in the cluster
		peerIDStr := fmt.Sprintf("%d", peerID)
		if existingServers[peerIDStr] {
			s.logger.Info("Peer already in cluster", "peerID", peerID)
			continue
		}

		// Construct peer address using the specified format
		peerAddr := fmt.Sprintf("10.201.%d.1:%d", peerID, s.port)

		s.logger.Info("Adding peer to cluster", "peerID", peerID, "peerAddr", peerAddr)

		// Add the peer to the cluster
		future := s.raft.AddVoter(
			raft.ServerID(peerIDStr),
			raft.ServerAddress(peerAddr),
			0,              // prevIndex (0 means no previous index)
			30*time.Second, // timeout
		)

		if err := future.Error(); err != nil {
			s.logger.Error("Failed to add peer to cluster", "peerID", peerID, "error", err)
		} else {
			s.logger.Info("Successfully added peer to cluster", "peerID", peerID)
		}
	}

	s.logger.Info("Finished ensuring all peers are in cluster")
}

// IsLeader returns true if this node is the leader
func (s *Server) IsLeader() bool {
	return s.raft.State() == raft.Leader
}

// GetLeader returns the address and ID of the current leader
func (s *Server) GetLeader() (raft.ServerAddress, raft.ServerID) {
	return s.raft.LeaderWithID()
}

// GetState returns the state of the Raft node as a string
func (s *Server) GetState() string {
	return s.raft.State().String()
}

func (s *Server) leaderCallback() {
	s.logger.Info("Leader callback called")

	// print all cluster members
	s.logger.Info("members: %v", s.raft.GetConfiguration())
}

func (s *Server) followerCallback() {
	s.logger.Info("Follower callback called")
}

// IsClusterHealthy checks if the Raft cluster is in a healthy state by verifying:
// 1. There is an elected leader
// 2. There are enough voting members for a quorum
// 3. This node can communicate with the leader (if not the leader itself)
func (s *Server) IsClusterHealthy() (bool, string) {
	// Check if there's a leader
	leaderAddr, _ := s.raft.LeaderWithID()
	if leaderAddr == "" {
		return false, "no leader elected"
	}

	// Get current configuration to check for quorum
	configFuture := s.raft.GetConfiguration()
	if err := configFuture.Error(); err != nil {
		return false, fmt.Sprintf("error getting configuration: %v", err)
	}

	// Count voting members
	voterCount := 0
	for _, server := range configFuture.Configuration().Servers {
		if server.Suffrage == raft.Voter {
			voterCount++
		}
	}

	// Calculate minimum quorum size (majority)
	minimumQuorum := (voterCount / 2) + 1

	// Check if we have enough voters for quorum
	if voterCount < minimumQuorum {
		return false, fmt.Sprintf("insufficient voters: %d (minimum needed: %d)", voterCount, minimumQuorum)
	}

	// If we're the leader, do additional checks
	if s.IsLeader() {
		// Leader-specific health checks could go here
		// For example, check if pending log entries are being applied efficiently
		stats := s.raft.Stats()

		// Check if there are too many uncommitted entries
		if uncommitted, err := strconv.Atoi(stats["commit_index"]); err == nil {
			if applied, err := strconv.Atoi(stats["applied_index"]); err == nil {
				if uncommitted-applied > 1000 { // Arbitrary threshold
					return false, "too many uncommitted entries"
				}
			}
		}
	} else {
		// Follower-specific checks
		// Check last contact time with leader
		stats := s.raft.Stats()
		lastContact, err := strconv.ParseInt(stats["last_contact"], 10, 64)
		if err == nil {
			// If last contact time is more than 10 seconds ago, consider unhealthy
			if lastContact > 10000 { // 10 seconds in milliseconds
				return false, "no recent contact with leader"
			}
		}
	}

	// All checks passed
	return true, "cluster is healthy"
}

// GetClusterHealth returns a structured health status with details
func (s *Server) GetClusterHealth() map[string]interface{} {
	healthy, reason := s.IsClusterHealthy()

	// Get current configuration
	configFuture := s.raft.GetConfiguration()
	var servers []map[string]string

	if err := configFuture.Error(); err == nil {
		for _, server := range configFuture.Configuration().Servers {
			serverInfo := map[string]string{
				"id":       string(server.ID),
				"address":  string(server.Address),
				"suffrage": server.Suffrage.String(),
			}
			servers = append(servers, serverInfo)
		}
	}

	// Get basic stats
	stats := s.raft.Stats()

	return map[string]interface{}{
		"healthy":   healthy,
		"reason":    reason,
		"state":     s.raft.State().String(),
		"leader":    string(s.raft.Leader()),
		"servers":   servers,
		"node_id":   s.local,
		"node_addr": s.listenAddr,
		"is_leader": s.IsLeader(),
		"stats":     stats,
		"timestamp": time.Now().Unix(),
	}
}
