package latency

import (
	"net"
)

type Result struct {
	Protocol   string
	Status     int
	AvgLatency int64
	Jitter     int64
	Loss       float64
}

type Client struct {
	SourceIP net.IP
	TargetIP net.IP
}
