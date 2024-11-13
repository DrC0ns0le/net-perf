package main

import (
	"flag"
	"time"

	"github.com/DrC0ns0le/net-perf/bandwidth"
	"github.com/DrC0ns0le/net-perf/link"
	"github.com/DrC0ns0le/net-perf/measure"
	"github.com/DrC0ns0le/net-perf/metrics"
)

var (
	bandwidthPort = flag.Int("bandwidth.port", 5121, "port for bandwidth measurement server")
	metricsPort   = flag.Int("metrics.port", 5120, "port for metrics server")

	metricsPath = flag.String("metrics.path", "/metrics", "path for metrics server")

	bandwidthDuration   = flag.Duration("bandwidth.duration", 5*time.Second, "duration for bandwidth measurement")
	bandwidthBandwidth  = flag.Int("bandwidth.bandwidth", 10, "bandwidth in mbps")
	bandwidthPacketSize = flag.Int("bandwidth.packetsize", 1500, "packet size in bytes")
	bandwidthBufferSize = flag.Int("bandwidth.buffer", 1500, "buffer size in bytes")
	bandwidthMaxRetries = flag.Int("bandwidth.maxretries", 3, "max number of retries")
	bandwidthRetryDelay = flag.Duration("bandwidth.retrydelay", 1*time.Second, "delay between retries")
	bandwidthTimeout    = flag.Duration("bandwidth.timeout", 10*time.Second, "timeout for bandwidth measurement")
	bandwidthOutOfOrder = flag.Int("bandwidth.outoforder", 0, "threshold for out-of-order packets")
)

func main() {
	flag.Parse()

	bandwidth.Init(&bandwidth.Config{
		Port:       5121,
		BufferSize: 1500,
		Bandwidth:  1,
		PacketSize: 500,
		Duration:   5 * time.Second,
	})

	// start bandwidth measurement server
	go bandwidth.Serve()

	// start measurement workers for bandwidth & latency
	go measure.Start()
	defer measure.Stop()

	// start link switch
	go link.Start()

	// start metrics server
	metrics.Serve("5120")
}
