package bandwidth

import (
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"github.com/DrC0ns0le/net-perf/internal/system"
	"github.com/DrC0ns0le/net-perf/internal/system/netctl"
	"github.com/DrC0ns0le/net-perf/pkg/logging"
)

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

var runningServers []*Server

func Serve(global *system.Node) {

	if *bandwidthBufferSize < *bandwidthPacketSize+12 {
		*bandwidthBufferSize = *bandwidthPacketSize + 12
		global.Logger.Warnf("configured buffer size too small, using %d bytes", *bandwidthBufferSize)
	}

	localAddrs, err := netctl.GetLocalLoopbackIP()
	if err != nil {
		global.Logger.Fatal(err)
	}

	for _, addr := range localAddrs {
		runningServers = append(runningServers, &Server{
			localAddr: fmt.Sprintf("%s:%d", addr, *bandwidthPort),
			stopCh:    global.StopCh,
			clients:   make(map[string]chan Packet),
			logger:    global.Logger.With("component", "server", "listener", fmt.Sprintf("%s:%d", addr, *bandwidthPort)),
		})
	}

	for _, server := range runningServers {
		go server.Run()
	}
}

func (s *Server) Run() {
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
				go s.clientWorker(receiveChan)
			}
			s.mu.Unlock()

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

	// send stats is also used as a heartbeat for the client
	ticker := time.NewTicker(*bandwidthStatsInterval)
	defer ticker.Stop()
	timeoutTicker := time.NewTicker(*bandwidthTimeout)
	defer timeoutTicker.Stop()
	for {
		select {
		case <-s.stopCh:
			s.logger.Infof("stopping client worker")
			return
		case <-ticker.C:
			if stats.TotalPackets > 0 {
				go s.sendStats(stats)
			}
		case <-timeoutTicker.C:
			s.logger.Errorf("test failed due to %s timeout, received %d packets, dropped %d, out of order %d, average jitter %d, jitter variance %f",
				stats.ClientAddr.String(), stats.TotalPackets, stats.DroppedPackets, stats.OutOfOrderPackets, stats.AverageJitter, stats.JitterVariance)
			s.mu.Lock()
			delete(s.clients, stats.ClientAddr.String())
			s.mu.Unlock()
			return
		case packet := <-receiveChan:
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
			}

			// TODO: better test control message
			// end of test
			if packet.Timestamp == 101 {
				s.sendAcknowledgment(stats.ClientAddr)
				s.sendFinalStats(stats)
				s.mu.Lock()
				delete(s.clients, stats.ClientAddr.String())
				s.mu.Unlock()
				return
			}

			timeoutTicker.Reset(10 * time.Second)
			if len(timeoutTicker.C) > 0 {
				<-timeoutTicker.C
			}
		}
	}
}

func (s *Server) sendAcknowledgment(addr *net.UDPAddr) {
	s.sendMessage(addr, "ACK")

}

func (s *Server) sendStats(stats *ClientStats) {
	statsMsg := fmt.Sprintf("STATS|%d|%d|%d|%.3f|%d",
		stats.TotalPackets,
		stats.OutOfOrderPackets,
		stats.DroppedPackets,
		stats.JitterVariance,
		stats.MaxJitter,
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
	// send 3 times to make sure it is received
	for i := 0; i < 3; i++ {
		s.sendMessage(stats.ClientAddr, finalStats)
		time.Sleep(100 * time.Millisecond)
	}
}

func (s *Server) sendMessage(addr *net.UDPAddr, message string) {
	_, err := s.conn.WriteToUDP([]byte(message), addr)
	if err != nil {
		s.logger.Errorf("error sending message to %s: %v", addr, err)
	}
}
