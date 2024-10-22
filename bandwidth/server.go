package bandwidth

import (
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"strconv"
	"sync"
	"time"

	"github.com/DrC0ns0le/net-perf/utils"
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
}

var runningServers []*Server

func Serve() {

	localAddrs, err := utils.GetLocalLoopbackIP()
	if err != nil {
		log.Fatal(err)
	}

	for _, addr := range localAddrs {
		runningServers = append(runningServers, &Server{
			localAddr: addr + ":" + strconv.Itoa(defaultPort),
			stopCh:    make(chan struct{}),
			clients:   make(map[string]chan Packet),
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

	log.Printf("Bandwidth server listening on %s", s.conn.LocalAddr().String())
	s.processPacket()
}

func (s *Server) processPacket() {
	buffer := make([]byte, bufferSize)
	for {
		select {
		case <-s.stopCh:
			log.Println("Stopping server")
			return
		default:
			n, remoteAddr, err := s.conn.ReadFromUDP(buffer)
			if err != nil {
				log.Println("Error reading:", err)
				continue
			}

			if n < 12 {
				log.Println("Received packet too small")
				continue
			}

			s.mu.Lock()
			receiveChan, exists := s.clients[remoteAddr.String()]
			if !exists {
				receiveChan = make(chan Packet, channelBufferSize)
				s.clients[remoteAddr.String()] = receiveChan
				go s.clientWorker(receiveChan)
			}
			s.mu.Unlock()

			packet := Packet{SourceAddr: remoteAddr}
			packet.SequenceNumber = binary.BigEndian.Uint32(buffer[:4])
			packet.Timestamp = int64(binary.BigEndian.Uint64(buffer[4:12]))

			receiveChan <- packet

			if len(receiveChan) > channelBufferSize/2 {
				log.Printf("Warning: channel buffer is more than half full")
			}

		}
	}
}

func (s *Server) clientWorker(receiveChan chan Packet) {
	stats := &ClientStats{}

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	timeoutTicker := time.NewTicker(3 * time.Second)
	defer timeoutTicker.Stop()
	for {
		select {
		case <-s.stopCh:
			log.Println("Stopping client worker for", stats.ClientAddr.String())
			return
		case <-ticker.C:
			if stats.TotalPackets > 0 {
				go s.sendStats(stats)
			}
		case <-timeoutTicker.C:
			log.Printf("Test failed due to %s timeout, received %d packets, dropped %d, out of order %d, average jitter %d, jitter variance %f",
				stats.ClientAddr.String(), stats.TotalPackets, stats.DroppedPackets, stats.OutOfOrderPackets, stats.AverageJitter, stats.JitterVariance)
			s.mu.Lock()
			delete(s.clients, stats.ClientAddr.String())
			s.mu.Unlock()
			return
		case packet := <-receiveChan:
			if stats.TotalPackets > 0 {
				if packet.SequenceNumber > stats.HighestSeq {
					stats.HighestSeq = packet.SequenceNumber
				} else {
					if stats.HighestSeq-packet.SequenceNumber < uint32(outOfOrderThreshold) {
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
				stats.StartTime = time.Now()
				stats.LastUpdate = stats.StartTime
				stats.ClientAddr = packet.SourceAddr
				stats.TotalPackets = 1
				stats.AverageJitter = 0
				stats.JitterVariance = 0
			}

			stats.LastUpdate = time.Now()

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
		log.Printf("Error sending message to %s: %v", addr, err)
	}
}
