package bandwidth

import (
	"flag"
	"fmt"
	"net"
	"time"
)

var (
	bandwidthPort              = flag.Int("bandwidth.port", 5121, "port for bandwidth measurement server")
	bandwidthDuration          = flag.Duration("bandwidth.duration", 5*time.Second, "duration for bandwidth measurement")
	bandwidthBandwidth         = flag.Int("bandwidth.bandwidth", 1, "bandwidth in mbps")
	bandwidthPacketSize        = flag.Int("bandwidth.packetsize", 500, "packet size in bytes")
	bandwidthBufferSize        = flag.Int("bandwidth.buffer", 1500, "buffer size in bytes")
	bandwidthMaxRetries        = flag.Int("bandwidth.maxretries", 4, "max number of retries")
	bandwidthRetryDelay        = flag.Duration("bandwidth.retrydelay", 250*time.Millisecond, "delay between retries")
	bandwidthTimeout           = flag.Duration("bandwidth.timeout", 3*time.Second, "timeout for bandwidth measurement")
	bandwidthStatsInterval     = flag.Duration("bandwidth.statsinterval", 1*time.Second, "interval for sending bandwidth measurement stats, also used for heartbeat")
	bandwidthChannelBufferSize = flag.Int("bandwidth.channelbuffer", 100, "buffered packets in server receive channel")

	// server side
	bandwidthOutOfOrder = flag.Int("bandwidth.outoforder", 0, "threshold for out-of-order packets")
)

type Packet struct {
	// SourceAddr is the source IP address of the packet
	SourceAddr *net.UDPAddr
	// SequenceNumber is the sequence number of the packet
	SequenceNumber uint32
	// Timestamp is the timestamp of the packet
	Timestamp int64
}

// ClientStats contains the statistics for a client
// Used by the server
type ClientStats struct {
	// IP address of the client
	ClientAddr *net.UDPAddr
	// IP address of the server
	ServerAddr *net.UDPAddr
	// Total number of packets received
	TotalPackets uint32
	// Number of dropped packets
	DroppedPackets uint32
	// Total jitter
	TotalJitter int64
	// Average jitter
	AverageJitter int64
	// Jitter variance
	JitterVariance float64
	// Max jitter
	MaxJitter int64
	// Start time
	StartTime time.Time
	// Last update time
	LastUpdate time.Time
	// Highest packet sequence number
	HighestSeq uint32
	// Number of out-of-order packets
	OutOfOrderPackets uint32
}

type Result struct {
	// Protocol used
	Protocol string
	// Test outcome, 1 for success, 0 for failure
	Status int
	// Percentage packet lost
	Loss float64
	// Percentage packet received out-of-order
	OutOfOrder float64
	// Percentage packet dropped due to being received late
	Dropped float64
	// Jitter in microseconds
	Jitter float64
	// Effective bandwidth in bits/sec
	Bandwidth int
	// Test duration in seconds
	Duration float64

	// Test information
	// Specified bandwidth in Mbits/sec
	TargetBandwidth int
	// Specified test duration in seconds
	TargetDuration int
	// Specified packet size in bytes
	PacketSize int
}

func validateFlags() error {
	// Validate bandwidth (in Mbps)
	// Max safe bandwidth to avoid overflow when converting to bps (uint32_max/1000000)
	const maxBandwidth = 4294 // ~ 4.2Gbps
	if *bandwidthBandwidth <= 0 {
		return fmt.Errorf("bandwidth must be positive")
	}
	if *bandwidthBandwidth > maxBandwidth {
		return fmt.Errorf("bandwidth too large: would cause overflow in calculations")
	}

	// Validate duration
	// Check if duration * bandwidth * 1000000 would overflow uint32
	maxSeconds := uint64(4294967295) / (uint64(*bandwidthBandwidth) * 1000000)
	if bandwidthDuration.Seconds() > float64(maxSeconds) {
		return fmt.Errorf("duration too large: would cause overflow in bandwidth calculations")
	}

	// Validate packet size
	if *bandwidthPacketSize <= 0 {
		return fmt.Errorf("packet size must be positive")
	}
	if *bandwidthPacketSize > 9000 { // Assuming ethernet max MTU 9000
		return fmt.Errorf("packet size too large: exceeds maximum ethernet MTU")
	}

	// Validate buffer size
	if *bandwidthBufferSize < *bandwidthPacketSize {
		return fmt.Errorf("buffer size must be at least as large as packet size")
	}
	if *bandwidthBufferSize > 65535 { // Reasonable max buffer size
		return fmt.Errorf("buffer size too large")
	}

	// Validate channel buffer size
	if *bandwidthChannelBufferSize <= 0 {
		return fmt.Errorf("channel buffer size must be positive")
	}
	if *bandwidthChannelBufferSize > 10000 { // Arbitrary reasonable limit
		return fmt.Errorf("channel buffer size too large")
	}

	// Validate retry parameters
	if *bandwidthMaxRetries < 0 {
		return fmt.Errorf("max retries cannot be negative")
	}
	if *bandwidthMaxRetries > 10 { // Arbitrary reasonable limit
		return fmt.Errorf("max retries too large")
	}

	// Validate timing parameters
	if bandwidthRetryDelay.Seconds() <= 0 {
		return fmt.Errorf("retry delay must be positive")
	}
	if bandwidthTimeout.Seconds() <= 0 {
		return fmt.Errorf("timeout must be positive")
	}
	if bandwidthStatsInterval.Seconds() <= 0 {
		return fmt.Errorf("stats interval must be positive")
	}

	// Validate out-of-order threshold
	if *bandwidthOutOfOrder < 0 {
		return fmt.Errorf("out-of-order threshold cannot be negative")
	}

	// Calculate packets per second to validate arithmetic
	packetsPerSecond := (uint64(*bandwidthBandwidth) * 1000000 * uint64(bandwidthDuration.Seconds())) /
		(uint64(*bandwidthPacketSize) * 8)
	if packetsPerSecond > uint64(^uint32(0)) {
		return fmt.Errorf("parameter combination would cause overflow in packets per second calculation")
	}

	// Validate timeout is greater than stats interval
	if bandwidthTimeout.Seconds() <= bandwidthStatsInterval.Seconds() {
		return fmt.Errorf("timeout must be greater than stats interval")
	}

	return nil
}
