package bandwidth

import (
	"encoding/binary"
	"fmt"
	"log"
	"math"
	"net"
	"sync"
	"time"

	"github.com/DrC0ns0le/net-perf/internal/system"
	"github.com/DrC0ns0le/net-perf/internal/system/netctl"
	"github.com/DrC0ns0le/net-perf/pkg/logging"
)

type ServerConfig struct {
	stopCh         chan struct{}
	runningServers []*Server

	logger logging.Logger
}

type Server struct {

	// UDP connection
	localAddr string
	conn      *net.UDPConn

	// Map of client to channel
	clients map[string]chan Packet
	mu      sync.RWMutex

	// Stop channel
	stopCh chan struct{}

	logger logging.Logger
}

func NewServer(global *system.Node) *ServerConfig {
	return &ServerConfig{
		runningServers: make([]*Server, 0),
		stopCh:         global.StopCh,
		logger:         global.Logger,
	}
}

func (s *ServerConfig) Start() error {

	// validation checks for bandwidth measurement
	err := validateFlags()
	if err != nil {
		return fmt.Errorf("flags validation failed: %v", err)
	}

	localAddrs, err := netctl.GetLocalLoopbackIP()
	if err != nil {
		return fmt.Errorf("error getting local loopback IP: %v", err)
	}

	for _, addr := range localAddrs {
		s.runningServers = append(s.runningServers, &Server{
			localAddr: fmt.Sprintf("%s:%d", addr, *bandwidthPort),
			stopCh:    s.stopCh,
			clients:   make(map[string]chan Packet),
			logger:    s.logger.With("component", "server", "listener", fmt.Sprintf("%s:%d", addr, *bandwidthPort)),
		})
	}

	for _, server := range s.runningServers {
		go server.Serve()
	}

	return nil
}

func (s *Server) Serve() {
	addr, err := net.ResolveUDPAddr("udp4", s.localAddr)
	if err != nil {
		log.Fatal(err)
	}

	s.conn, err = net.ListenUDP("udp4", addr)
	if err != nil {
		log.Fatal(err)
	}
	defer s.conn.Close()

	s.logger.Infof("bandwidth server started")
	s.processPacket()
}

func (s *Server) processPacket() {
	buffer := make([]byte, *bandwidthBufferSize)
	for {
		select {
		case <-s.stopCh:
			s.logger.Infof("stopping server")
			return
		default:
			n, remoteAddr, err := s.conn.ReadFromUDP(buffer)
			if err != nil {
				s.logger.Errorf("error reading: %v", err)
				continue
			}

			if n < 12 {
				s.logger.Errorf("received packet too small")
				continue
			}

			s.mu.Lock()
			receiveChan, exists := s.clients[remoteAddr.String()]
			if !exists {
				receiveChan = make(chan Packet, *bandwidthChannelBufferSize)
				s.clients[remoteAddr.String()] = receiveChan
			}
			s.mu.Unlock()

			// start client worker if it doesn't exist
			// started outside of the lock to avoid race conditions
			if !exists {
				go s.clientWorker(receiveChan)
			}

			packet := Packet{SourceAddr: remoteAddr}
			packet.SequenceNumber = binary.BigEndian.Uint32(buffer[:4])
			packet.Timestamp = int64(binary.BigEndian.Uint64(buffer[4:12]))

			receiveChan <- packet

			if len(receiveChan) > *bandwidthChannelBufferSize/2 {
				s.logger.Warnf("channel buffer is %.1f%% full", float64(len(receiveChan))/float64(*bandwidthChannelBufferSize)*100)
			}

		}
	}
}

func (s *Server) clientWorker(receiveChan chan Packet) {
	stats := &ClientStats{}

	// Send stats is also used as a heartbeat for the client
	ticker := time.NewTicker(*bandwidthStatsInterval)
	defer ticker.Stop()
	timeoutTicker := time.NewTicker(*bandwidthTimeout)
	defer timeoutTicker.Stop()

	// Main loop to process packets
	var packet Packet
	for {
		select {
		// check if stop channel is closed
		case <-s.stopCh:
			s.logger.Infof("stopping client worker")
			return
		// periodically send stats
		case <-ticker.C:
			if stats.TotalPackets > 0 {
				go s.sendStats(stats)
			}
		// check if client has timed out, then stop and clean up
		case <-timeoutTicker.C:
			s.logger.Errorf("test failed due to client %s timeout after %d seconds, received %d packets, dropped %d, out of order %d, average jitter %d, jitter variance %f",
				stats.ClientAddr.String(), *bandwidthTimeout/time.Second, stats.TotalPackets, stats.DroppedPackets, stats.OutOfOrderPackets, stats.AverageJitter, stats.JitterVariance)
			s.mu.Lock()
			delete(s.clients, stats.ClientAddr.String())
			s.mu.Unlock()
			return
		// receive packet
		case packet = <-receiveChan:
			// process packet and update stats
			stats.LastUpdate = time.Now()

			// check if test is already ongoing
			if stats.TotalPackets > 0 {
				// check if packet is out of order
				if packet.SequenceNumber > stats.HighestSeq {
					stats.HighestSeq = packet.SequenceNumber
				} else {
					if stats.HighestSeq-packet.SequenceNumber < uint32(*bandwidthOutOfOrder) {
						stats.OutOfOrderPackets++
					} else {
						stats.DroppedPackets++
					}
				}

				// Calculate jitter using Welford's online algorithm
				jitter := time.Since(stats.LastUpdate).Microseconds()
				oldM := stats.AverageJitter
				stats.TotalPackets++
				stats.AverageJitter += (jitter - oldM) / int64(stats.TotalPackets)
				stats.JitterVariance += float64(jitter-oldM) * float64(jitter-stats.AverageJitter)

				if jitter > stats.MaxJitter {
					stats.MaxJitter = jitter
				}
			} else {
				// first packet
				stats.StartTime = stats.LastUpdate
				stats.ClientAddr = packet.SourceAddr
				stats.TotalPackets = 1
				stats.AverageJitter = 0
				stats.JitterVariance = 0
				stats.HighestSeq = packet.SequenceNumber
			}

			// End of test
			if packet.Timestamp == math.MaxInt64 && packet.SequenceNumber == math.MaxUint32 {
				s.sendFinalStats(stats)

				// Set up a timer for delayed cleanup
				cleanupTimer := time.NewTimer(2 * *bandwidthRetryDelay)
				defer cleanupTimer.Stop()

				// Wait for potential retries
				select {
				case <-cleanupTimer.C:
					// Cleanup after timeout
					s.mu.Lock()
					delete(s.clients, stats.ClientAddr.String())
					s.mu.Unlock()
					return
				case <-s.stopCh:
					// Server is stopping, exit immediately
					s.mu.Lock()
					delete(s.clients, stats.ClientAddr.String())
					s.mu.Unlock()
					return
				case packet := <-receiveChan:
					// Handle potential retry
					if packet.Timestamp == math.MaxInt64 && packet.SequenceNumber == math.MaxUint32 {
						s.sendFinalStats(stats)
						// Reset the timer
						cleanupTimer.Reset(2 * *bandwidthRetryDelay)
						if len(cleanupTimer.C) > 0 {
							<-cleanupTimer.C
						}
					} else {
						// Unexpected packet after end of test
						s.logger.Errorf("unexpected packet received after end of test from client %s: %v", stats.ClientAddr, packet)
					}
				}
			}

			// Reset timeout ticker
			timeoutTicker.Reset(*bandwidthTimeout)
			if len(timeoutTicker.C) > 0 {
				<-timeoutTicker.C
			}
		}
	}
}

func (s *Server) sendStats(stats *ClientStats) {
	statsMsg := fmt.Sprintf("STATS|%d",
		stats.TotalPackets,
	)
	s.sendMessage(stats.ClientAddr, statsMsg)
}

func (s *Server) sendFinalStats(stats *ClientStats) {
	finalStats := fmt.Sprintf("FINAL|%d|%d|%d|%.3f|%d|%d",
		stats.TotalPackets,
		stats.OutOfOrderPackets,
		stats.DroppedPackets,
		stats.JitterVariance,
		stats.MaxJitter,
		stats.LastUpdate.UnixMicro()-stats.StartTime.UnixMicro(),
	)
	s.sendMessage(stats.ClientAddr, finalStats)
}

func (s *Server) sendMessage(addr *net.UDPAddr, message string) {
	_, err := s.conn.WriteToUDP([]byte(message), addr)
	if err != nil {
		s.logger.Errorf("error sending message to %s: %v", addr, err)
	}
}
