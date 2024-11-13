package bandwidth

import (
	"net"
	"time"
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

type Config struct {
	// Port is the UDP port that the server will listen on
	Port int
	// Duration is the maximum total test duration
	Duration time.Duration
	// BufferSize is the size of the UDP packet buffer in bytes
	BufferSize int
	// Bandwidth is the bandwidth in Mbits/sec
	Bandwidth int
	// PacketSize is the size of the UDP packet in bytes
	PacketSize int
	// MaxRetries is the maximum number of times to retry sending a packet
	MaxRetries int
	// RetryDelay is the delay between retries
	RetryDelay time.Duration
	// Out of order threshold
	OutOfOrderThreshold int
}

var (
	defaultPort int
	duration    time.Duration
	bufferSize  int
	bandwidth   int
	packetSize  int
	maxRetries  int
	retryDelay  time.Duration

	// server side
	outOfOrderThreshold int
	channelBufferSize   int
)

func Init(config *Config) {
	if config.Port == 0 {
		defaultPort = 5210
	} else {
		defaultPort = config.Port
	}

	if config.Duration == 0 {
		duration = 5 * time.Second
	} else {
		duration = config.Duration
	}

	if config.Bandwidth == 0 {
		bandwidth = 1
	} else {
		bandwidth = config.Bandwidth
	}

	if config.PacketSize == 0 {
		packetSize = 1500
	} else {
		packetSize = config.PacketSize
	}

	if config.BufferSize == 0 {
		bufferSize = packetSize + 12
	} else {
		bufferSize = config.BufferSize
	}

	if config.MaxRetries == 0 {
		maxRetries = 5
	} else {
		maxRetries = config.MaxRetries
	}

	if config.RetryDelay == 0 {
		retryDelay = 1 * time.Second
	} else {
		retryDelay = config.RetryDelay
	}

	if config.OutOfOrderThreshold == 0 {
		outOfOrderThreshold = 50
	} else {
		outOfOrderThreshold = config.OutOfOrderThreshold
	}

	if config.BufferSize == 0 {
		channelBufferSize = 50
	} else {
		channelBufferSize = config.BufferSize
	}

}

// func padToEight(s string) []byte {
// 	d := make([]byte, 8)
// 	copy(d, []byte(s))
// 	return d
// }
