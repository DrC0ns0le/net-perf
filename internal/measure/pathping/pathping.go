package pathping

import (
	"context"
	"encoding/binary"
	"flag"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/DrC0ns0le/net-perf/internal/system"
	"github.com/DrC0ns0le/net-perf/pkg/logging"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	pathPingPort              = flag.Int("pathping.port", 5124, "port for path ping server")
	pathPingBuf               = flag.Int("pathping.buffer", 128, "buffer size for path ping server")
	pathPingMax               = flag.Int("pathping.max", 16, "max number of hops for path ping")
	pathPingTimeout           = flag.Duration("pathping.timeout", 1*time.Second, "timeout for path ping")
	pathPingProcessingHistory = flag.Int("pathping.processing.history", 100, "number of processing durations to keep track of")

	// singleton
	pathPingServer *PathPingServer

	pathLatencyProcessingDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "pathping_process_duration_microseconds",
		Help:    "Distribution of pathping packet processing durations in microseconds",
		Buckets: []float64{100, 250, 500, 1000, 2500, 5000, 10000}, // buckets in microseconds
	}, []string{"id"})
	pathAverageProcessingDuration = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "pathping_average_process_duration_microseconds",
		Help: "Average pathping packet processing duration in microseconds",
	}, []string{"id"})
)

// Result contains the results of a completed measurement
type Result struct {
	Status   int
	Protocol string
	Path     []uint32
	Duration time.Duration
	Loss     float64
}

type result struct {
	id          int64
	path        []uint32
	duration    []time.Duration
	lastUpdated time.Time
}

// Config contains the configuration for a mesh node
type Config struct {
	ID         uint32
	Port       int
	BufferSize int
	Logger     logging.Logger
}

// PathPingServer represents a mesh network node capable of forwarding and measuring latency
type PathPingServer struct {
	config    Config
	listener  *net.UDPConn
	peerAddrs map[uint32]*net.UDPAddr
	active    bool
	done      chan struct{}

	processingDurationCh chan time.Duration
	processingDuration   []time.Duration

	resultsMu sync.RWMutex
	results   map[string]*result
}

// packet represents the internal measurement packet structure
type packet struct {
	ID         int64 // Unique test ID
	Origin     uint32
	CurrentHop uint32
	Sequence   uint64
	PathLength uint32
	Path       []uint32
	StartTime  int64 // Only used by origin node
}

// NewServer returns a new PathPingServer instance with the given config.
//
// The PathPingServer is a singleton and only one instance is created for the
// entire program. Subsequent calls to NewServer will return the same instance.
//
// The server is configured with the given site ID, port, and buffer size.
// A logger is also provided which is used to log any errors or messages.
//
// The server is not started until the Start method is called.
func NewServer(global *system.Node) *PathPingServer {
	if pathPingServer != nil {
		return pathPingServer
	}

	config := Config{
		ID:         uint32(global.SiteID),
		Port:       *pathPingPort,
		BufferSize: *pathPingBuf,
		Logger:     global.Logger.With("component", "pathping"),
	}

	s := &PathPingServer{
		config:    config,
		peerAddrs: make(map[uint32]*net.UDPAddr),
		results:   make(map[string]*result),
		done:      make(chan struct{}),

		processingDurationCh: make(chan time.Duration, *pathPingProcessingHistory), // buffered channels to prevent blocking
		processingDuration:   make([]time.Duration, *pathPingProcessingHistory),    // circular buffer
	}

	pathPingServer = s

	return s
}

// Start the PathPingServer and begin listening for incoming packets.
//
// The server is configured to listen on the configured port and IP address
// and to connect to all other nodes in the mesh network.
//
// The server will start listening for incoming packets and will begin
// forwarding packets to the next hop.
//
// The server will return an error if it is already started.
func (n *PathPingServer) Start() error {
	if n.active {
		return fmt.Errorf("node already started")
	}

	addr := &net.UDPAddr{
		Port: *pathPingPort,
		IP:   net.ParseIP("0.0.0.0"),
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		return fmt.Errorf("failed to start listener: %w", err)
	}
	n.listener = conn

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	peers, err := GetAllPeers(ctx)
	if err != nil {
		return fmt.Errorf("failed to get peers: %w", err)
	}

	for _, peer := range peers {
		if peer == int(n.config.ID) {
			continue
		}
		n.addPeer(uint32(peer), fmt.Sprintf("10.201.%d.1:%d", peer, n.config.Port))
	}

	n.active = true
	go n.listen()
	n.config.Logger.Info("PathPing server started")
	go n.cleanUpResults()
	go n.updateProcessingDuration()
	return nil
}

func (n *PathPingServer) addPeer(id uint32, addr string) error {
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return fmt.Errorf("invalid peer address: %w", err)
	}
	n.peerAddrs[id] = udpAddr
	return nil
}

// MeasurementOptions contains options for starting a measurement
type MeasurementOptions struct {
	Count    int           // Number of measurements to take
	Interval time.Duration // Interval between measurements
}

// Measure sends a packet along the given path and measures the round-trip time.
//
// The measurement is repeated 'count' times with an interval of 'interval' between
// each measurement. The server will return an error if the measurement exceeds
// the given timeout.
//
// The path must start with the local node ID and must be between 2 and
// *pathPingMax (default 16) nodes long.
func (n *PathPingServer) Measure(path []uint32, opts MeasurementOptions) (Result, error) {
	result := Result{
		Status:   0,
		Protocol: "pathping",
		Path:     path,
	}
	if len(path) < 2 || len(path) > *pathPingMax {
		return result, fmt.Errorf("invalid path length: must be between 2 and %d", *pathPingMax)
	}

	if path[0] != n.config.ID {
		return result, fmt.Errorf("path must start with local node ID")
	}

	if path[0] != path[len(path)-1] {
		return result, fmt.Errorf("path must end with local node ID")
	}

	if opts.Count <= 0 {
		opts.Count = 10
	}

	if opts.Interval == 0 {
		opts.Interval = 100 * time.Millisecond
	}

	startTime := time.Now()
	for seq := uint64(0); seq < uint64(opts.Count); seq++ {
		p := packet{
			ID:         startTime.UnixNano(),
			Origin:     n.config.ID,
			Sequence:   seq,
			PathLength: uint32(len(path)),
			StartTime:  time.Now().UnixNano(),
			Path:       make([]uint32, len(path)),
		}
		copy(p.Path[:], path)

		err := n.forwardPacket(&p, true)
		if err != nil {
			return result, fmt.Errorf("failed to send packet: %w", err)
		}
		time.Sleep(opts.Interval)
	}

	// Check for results
	testID := fmt.Sprintf("%v-%v", path, startTime.UnixNano())
	totalTimeout := time.Duration(opts.Count) * (opts.Interval + *pathPingTimeout*2)
	timeout := time.After(totalTimeout)

	for {
		select {
		case <-timeout:
			// On timeout, return any results we have
			n.resultsMu.Lock()
			if r, ok := n.results[testID]; ok && len(r.duration) > 0 {
				for _, d := range r.duration {
					result.Duration += d
				}
				result.Duration /= time.Duration(len(r.duration))
				delete(n.results, testID)
				n.resultsMu.Unlock()
				return result, nil
			}
			n.resultsMu.Unlock()
			return result, fmt.Errorf("timeout waiting for results")

		default:
			n.resultsMu.Lock()
			r, ok := n.results[testID]
			if !ok {
				// Results not ready, unlock and try again
				n.resultsMu.Unlock()
				time.Sleep(100 * time.Millisecond)
				continue
			}

			// Check if we have all packets or if we haven't received new ones for a while
			if len(r.duration) == opts.Count || time.Since(r.lastUpdated) > *pathPingTimeout*2 {
				// Calculate average of received packets
				for _, d := range r.duration {
					result.Duration += d
				}
				if len(r.duration) > 0 {
					result.Duration /= time.Duration(len(r.duration))
				}
				delete(n.results, testID)
				n.resultsMu.Unlock()
				result.Status = 1
				return result, nil
			}

			// Results not complete yet
			n.resultsMu.Unlock()
			time.Sleep(opts.Interval / 2)
		}
	}
}

func (n *PathPingServer) Stop() error {
	return n.listener.Close()
}

func (n *PathPingServer) cleanUpResults() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-n.done:
			return
		case <-ticker.C:
			n.resultsMu.Lock()
			for i, r := range n.results {
				if time.Since(r.lastUpdated) > *pathPingTimeout*3 {
					n.config.Logger.Warnf("removing stale result: %v", i)
					delete(n.results, i)
				}
			}
			n.resultsMu.Unlock()
		}
	}
}

func (n *PathPingServer) updateProcessingDuration() {
	var duration time.Duration
	index := 0
	for {
		select {
		case <-n.done:
			return
		case duration = <-n.processingDurationCh:
			n.processingDuration[index] = duration
			index++
			if index == *pathPingProcessingHistory {
				index = 0
			}
			pathLatencyProcessingDuration.With(prometheus.Labels{
				"id": fmt.Sprintf("%d", n.config.ID),
			}).Observe(float64(duration.Microseconds()))
			pathAverageProcessingDuration.WithLabelValues(fmt.Sprintf("%d", n.config.ID)).Set(float64(n.getAverageProcessingDuration().Microseconds()))
		}
	}
}

func (n *PathPingServer) getAverageProcessingDuration() time.Duration {

	sum := time.Duration(0)
	for _, d := range n.processingDuration {
		if d == 0 {
			continue
		}
		sum += d
	}

	if sum == 0 {
		return 0
	}

	return sum / time.Duration(*pathPingProcessingHistory)
}

// listen for incoming packets and process them.
func (n *PathPingServer) listen() {
	buf := make([]byte, n.config.BufferSize)

	for {
		select {
		case <-n.done:
			return
		default:
			n.listener.SetReadDeadline(time.Now().Add(*pathPingTimeout))

			numBytes, _, err := n.listener.ReadFromUDP(buf)
			if err != nil {
				continue
			}

			if numBytes < 20 {
				continue // Minimum packet size
			}

			p := n.deserializePacket(buf[:numBytes])
			if p != nil {
				n.processPacket(p)
			}
		}
	}
}

// processPacket handles an incoming packet and checks if the path has completed.
// If so, the round-trip time is added to the results map. If not, the packet is
// forwarded to the next hop.
func (n *PathPingServer) processPacket(p *packet) {
	startTime := time.Now()
	defer func() {
		n.processingDurationCh <- time.Since(startTime)
	}()

	// Check if we've completed the path
	if p.CurrentHop+1 == p.PathLength-1 {
		if p.Origin == n.config.ID {
			rtt := time.Duration(time.Now().UnixNano() - p.StartTime)

			path := make([]uint32, p.PathLength)
			copy(path, p.Path[:p.PathLength])

			testID := fmt.Sprintf("%v-%v", path, p.ID)

			n.resultsMu.Lock()
			defer n.resultsMu.Unlock()
			if _, exists := n.results[testID]; !exists {
				// First packet
				n.results[testID] = &result{
					id:          p.ID,
					path:        path,
					duration:    []time.Duration{rtt},
					lastUpdated: time.Now(),
				}
			} else {
				// Subsequent packet(s)
				n.results[testID].duration = append(n.results[testID].duration, rtt)
				n.results[testID].lastUpdated = time.Now()
			}
		} else {
			// something is wrong
			n.config.Logger.Warnf("received packet from unknown origin: %v", p.Origin)
			n.config.Logger.Warnf("path: %+v", p)
		}
		return
	}

	// Forward packet
	err := n.forwardPacket(p, false)
	if err != nil {
		n.config.Logger.Warnf("failed to forward packet: %v", err)
	}
}

// forwardPacket forwards a packet to the next hop in the path. If the packet is
// an origin packet, it is initialized to the first hop. Otherwise, the packet's
// current hop is incremented and the packet is forwarded to that next hop.
func (n *PathPingServer) forwardPacket(p *packet, isOrigin bool) error {
	if isOrigin {
		p.CurrentHop = 0
	} else {
		p.CurrentHop++
	}

	p.StartTime += n.getAverageProcessingDuration().Nanoseconds()

	// Forward to next hop
	nextHop := p.Path[p.CurrentHop+1]
	nextAddr, exists := n.peerAddrs[nextHop]
	if !exists {
		return fmt.Errorf("peer %v not found", nextHop)
	}

	buf := n.serializePacket(p)
	_, err := n.listener.WriteToUDP(buf, nextAddr)

	return err
}

func (n *PathPingServer) serializePacket(p *packet) []byte {
	buf := make([]byte, n.config.BufferSize)

	// Add ID field at the start (int64 - 8 bytes)
	binary.LittleEndian.PutUint64(buf[0:], uint64(p.ID))

	// Shift all other fields by 8 bytes
	binary.LittleEndian.PutUint32(buf[8:], p.Origin)
	binary.LittleEndian.PutUint32(buf[12:], p.CurrentHop)
	binary.LittleEndian.PutUint64(buf[16:], p.Sequence)
	binary.LittleEndian.PutUint32(buf[24:], p.PathLength)
	binary.LittleEndian.PutUint64(buf[28:], uint64(p.StartTime))

	// Path array starts 8 bytes later than before
	for i := 0; i < int(p.PathLength); i++ {
		offset := 36 + (i * 4)
		binary.LittleEndian.PutUint32(buf[offset:], p.Path[i])
	}

	return buf
}

func (n *PathPingServer) deserializePacket(buf []byte) *packet {
	var p packet

	// Read ID field first
	p.ID = int64(binary.LittleEndian.Uint64(buf[0:]))

	// Shift all other fields by 8 bytes
	p.Origin = binary.LittleEndian.Uint32(buf[8:])
	p.CurrentHop = binary.LittleEndian.Uint32(buf[12:])
	p.Sequence = binary.LittleEndian.Uint64(buf[16:])
	p.PathLength = binary.LittleEndian.Uint32(buf[24:])
	p.StartTime = int64(binary.LittleEndian.Uint64(buf[28:]))

	if p.PathLength > uint32(*pathPingMax) {
		return nil
	}

	p.Path = make([]uint32, p.PathLength)
	for i := 0; i < int(p.PathLength); i++ {
		offset := 36 + (i * 4)
		p.Path[i] = binary.LittleEndian.Uint32(buf[offset:])
	}

	return &p
}
