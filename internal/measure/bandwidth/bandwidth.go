package bandwidth

import (
	"flag"
	"net"
	"time"
)

var (
	bandwidthPort              = flag.Int("bandwidth.port", 5121, "port for bandwidth measurement server")
	bandwidthDuration          = flag.Duration("bandwidth.duration", 5*time.Second, "duration for bandwidth measurement")
	bandwidthBandwidth         = flag.Int("bandwidth.bandwidth", 1, "bandwidth in mbps")
	bandwidthPacketSize        = flag.Int("bandwidth.packetsize", 500, "packet size in bytes")
	bandwidthBufferSize        = flag.Int("bandwidth.buffer", 1500, "buffer size in bytes")
	bandwidthMaxRetries        = flag.Int("bandwidth.maxretries", 3, "max number of retries")
	bandwidthRetryDelay        = flag.Duration("bandwidth.retrydelay", 1*time.Second, "delay between retries")
	bandwidthTimeout           = flag.Duration("bandwidth.timeout", 10*time.Second, "timeout for bandwidth measurement")
	bandwidthStatsInterval     = flag.Duration("bandwidth.statsinterval", 2*time.Second, "interval for sending bandwidth measurement stats")
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

// func padToEight(s string) []byte {
// 	d := make([]byte, 8)
// 	copy(d, []byte(s))
// 	return d
// }
