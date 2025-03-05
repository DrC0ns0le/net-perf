package pathping

import (
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
	pathPingProcessingHistory = flag.Int("pathping.processing.history", 10, "number of processing durations to keep track of")

	// Maximum possible packet size based on maximum path length
	maxPacketSize = 26 + (*pathPingMax * 4) // Fixed header (26) + max array size (pathPingMax * 4), each uint16 takes 2 bytes(path & pathver)

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
	Path     []uint16
	Duration time.Duration
	Loss     float64
}

type result struct {
	id          int64
	path        []uint16
	duration    []time.Duration
	lastUpdated time.Time
}

// Config contains the configuration for a mesh node
type Config struct {
	ID         uint16
	Port       int
	BufferSize int
	Logger     logging.Logger
}

// PathPingServer represents a mesh network node capable of forwarding and measuring latency
type PathPingServer struct {
	config    Config
	listener  *net.UDPConn
	peerAddrs map[uint16]map[uint16]*net.UDPAddr
	active    bool
	done      chan struct{}

	processingDurationCh chan time.Duration
	processingDuration   []time.Duration

	resultsMu sync.RWMutex
	results   map[string]*result
}

// packet represents the internal measurement packet structure
type packet struct {
	ID         int64    // Unique test ID
	Origin     uint16   // Origin node ID
	CurrentHop uint16   // Current hop count
	Sequence   uint16   // Sequence number
	PathLength uint16   // Length of path
	Status     uint16   // Status code
	StartTime  int64    // Start time (origin node only)
	Path       []uint16 // Path array
	PathVer    []uint16 // Version information for each hop
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
		ID:         uint16(global.SiteID),
		Port:       *pathPingPort,
		BufferSize: *pathPingBuf,
		Logger:     global.Logger.With("component", "pathping"),
	}

	s := &PathPingServer{
		config:    config,
		peerAddrs: make(map[uint16]map[uint16]*net.UDPAddr),
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

	peers, err := GetAllPeers()
	if err != nil {
		return fmt.Errorf("failed to get peers: %w", err)
	}

	for peer, ver := range peers {
		if peer == int(n.config.ID) {
			continue
		}

		// Add specific versions
		for _, v := range ver {
			if err := n.addPeer(uint16(peer), uint16(v), fmt.Sprintf("10.201.%d.%d:%d", peer, v, n.config.Port)); err != nil {
				return err
			}
		}

		// Add default version
		if err := n.addPeer(uint16(peer), 0, fmt.Sprintf("10.201.%d.1:%d", peer, n.config.Port)); err != nil {
			return err
		}
	}

	n.active = true
	go n.listen()
	n.config.Logger.Info("PathPing server started")
	go n.cleanUpResults()
	go n.updateProcessingDuration()
	return nil
}

func (n *PathPingServer) addPeer(id, ver uint16, addr string) error {
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return fmt.Errorf("invalid peer address: %w", err)
	}

	if n.peerAddrs[id] == nil {
		n.peerAddrs[id] = make(map[uint16]*net.UDPAddr)
	}

	n.peerAddrs[id][ver] = udpAddr
	return nil
}

// MeasurementOptions contains options for starting a measurement
type MeasurementOptions struct {
	Count    int           // Number of measurements to take
	Interval time.Duration // Interval between measurements
	PathVer  []uint16
}

// Measure sends a packet along the given path and measures the round-trip time.
//
// The measurement is repeated 'count' times with an interval of 'interval' between
// each measurement. The server will return an error if the measurement exceeds
// the given timeout.
//
// The path must start with the local node ID and must be between 2 and
// *pathPingMax (default 16) nodes long.
func (n *PathPingServer) Measure(path []uint16, opts MeasurementOptions) (Result, error) {
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

	if opts.PathVer != nil {
		if len(opts.PathVer) != len(path) {
			return result, fmt.Errorf("path version must be the same length as the path")
		}

		for _, v := range opts.PathVer {
			if v != 0 && v != 4 && v != 6 {
				return result, fmt.Errorf("invalid path version: must be 0, 4 or 6")
			}
		}
	} else {
		opts.PathVer = make([]uint16, len(path))
		for i := range path {
			opts.PathVer[i] = 0
		}
	}

	startTime := time.Now()
	for seq := uint16(0); seq < uint16(opts.Count); seq++ {
		p := packet{
			ID:         startTime.UnixNano(),
			Origin:     n.config.ID,
			Sequence:   seq,
			PathLength: uint16(len(path)),
			StartTime:  time.Now().UnixNano(),
			Path:       make([]uint16, len(path)),
			PathVer:    make([]uint16, len(path)),
		}
		copy(p.Path[:], path)
		copy(p.PathVer[:], opts.PathVer)

		err := n.forwardPacket(&p, true)
		if err != nil {
			return result, fmt.Errorf("failed to send packet: %w", err)
		}
		time.Sleep(opts.Interval)
	}

	// Check for results
	testID := fmt.Sprintf("%v-%v", path, startTime.UnixNano())
	totalTimeout := time.Duration(opts.Count)*(opts.Interval) + *pathPingTimeout
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
				time.Sleep(opts.Interval / 2)
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
	// Allocate buffer with maximum possible size
	buf := make([]byte, maxPacketSize)

	for {
		select {
		case <-n.done:
			return
		default:
			n.listener.SetReadDeadline(time.Now().Add(*pathPingTimeout))

			// Read the complete UDP datagram
			numBytes, _, err := n.listener.ReadFromUDP(buf)
			if err != nil {
				continue
			}

			// Process only the actual bytes received
			p, err := deserializePacket(buf[:numBytes])
			if err == nil {
				n.processPacket(p)
			} else {
				n.config.Logger.Debugf("Failed to deserialize packet: %v", err)
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

			path := make([]uint16, p.PathLength)
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

	nextHop := p.Path[p.CurrentHop+1]
	nextHopVer := p.PathVer[p.CurrentHop+1]
	nextAddr, exists := n.peerAddrs[nextHop][nextHopVer]
	if !exists {
		return fmt.Errorf("peer %v not found", nextHop)
	}

	buf := p.serialize()
	_, err := n.listener.WriteToUDP(buf, nextAddr)
	return err
}

// calculatePacketSize calculates the required buffer size for a packet
func (p *packet) calculatePacketSize() int {
	headerSize := 26                   // Fixed header size: ID(8) + Origin(2) + CurrentHop(2) + Sequence(2) + PathLength(2) + Status(2) + StartTime(8)
	arraySize := int(p.PathLength) * 4 // Each uint16 array element takes 2 bytes, and we have two arrays (Path and PathVer)
	return headerSize + arraySize
}

// serializePacket serializes a packet into a byte slice
func (p *packet) serialize() []byte {
	size := p.calculatePacketSize()
	buf := make([]byte, size)

	// Write fixed-size fields
	binary.LittleEndian.PutUint64(buf[0:], uint64(p.ID))
	binary.LittleEndian.PutUint16(buf[8:], p.Origin)
	binary.LittleEndian.PutUint16(buf[10:], p.CurrentHop)
	binary.LittleEndian.PutUint16(buf[12:], p.Sequence)
	binary.LittleEndian.PutUint16(buf[14:], p.PathLength)
	binary.LittleEndian.PutUint16(buf[16:], p.Status)
	binary.LittleEndian.PutUint64(buf[18:], uint64(p.StartTime))

	offset := 26

	// Write variable-length arrays
	for i := 0; i < int(p.PathLength); i++ {
		binary.LittleEndian.PutUint16(buf[offset:], p.Path[i])
		offset += 2
	}
	for i := 0; i < int(p.PathLength); i++ {
		binary.LittleEndian.PutUint16(buf[offset:], p.PathVer[i])
		offset += 2
	}

	return buf
}

// deserializePacket deserializes a byte slice into a packet
func deserializePacket(buf []byte) (*packet, error) {
	if len(buf) < 26 { // Minimum size for fixed fields
		return nil, fmt.Errorf("packet too small: %d bytes", len(buf))
	}

	p := &packet{}

	// Read fixed-size fields
	p.ID = int64(binary.LittleEndian.Uint64(buf[0:]))
	p.Origin = binary.LittleEndian.Uint16(buf[8:])
	p.CurrentHop = binary.LittleEndian.Uint16(buf[10:])
	p.Sequence = binary.LittleEndian.Uint16(buf[12:])
	p.PathLength = binary.LittleEndian.Uint16(buf[14:])
	p.Status = binary.LittleEndian.Uint16(buf[16:])
	p.StartTime = int64(binary.LittleEndian.Uint64(buf[18:]))

	// Validate PathLength
	if p.PathLength > uint16(*pathPingMax) {
		return nil, fmt.Errorf("invalid path length: %d", p.PathLength)
	}

	expectedSize := 26 + (int(p.PathLength) * 4) // Fixed size + array sizes
	if len(buf) < expectedSize {
		return nil, fmt.Errorf("packet too small for path length: expected %d, got %d", expectedSize, len(buf))
	}

	// Initialize and read variable-length arrays
	p.Path = make([]uint16, p.PathLength)
	p.PathVer = make([]uint16, p.PathLength)

	offset := 26
	for i := 0; i < int(p.PathLength); i++ {
		p.Path[i] = binary.LittleEndian.Uint16(buf[offset:])
		offset += 2
	}
	for i := 0; i < int(p.PathLength); i++ {
		p.PathVer[i] = binary.LittleEndian.Uint16(buf[offset:])
		offset += 2
	}

	return p, nil
}
