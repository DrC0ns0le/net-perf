package latency

import (
	"context"
	"time"

	probing "github.com/prometheus-community/pro-bing"
)

// Measures ICMP connection latency
func MeasureICMP(ctx context.Context, sourceIP, targetIP string) (Result, error) {
	pinger, err := probing.NewPinger(targetIP)
	if err != nil {
		return Result{Status: 0, Protocol: "icmp"}, err
	}
	pinger.Source = sourceIP
	pinger.SetPrivileged(true)
	pinger.Interval = 250 * time.Millisecond
	pinger.Count = 10
	err = pinger.RunWithContext(ctx) // Blocks until finished.
	if err != nil {
		return Result{Status: 0, Protocol: "icmp"}, err
	}

	return Result{
		Status:     1,
		Protocol:   "icmp",
		AvgLatency: pinger.Statistics().AvgRtt.Microseconds(),
		Jitter:     pinger.Statistics().StdDevRtt.Microseconds(),
		Loss:       pinger.Statistics().PacketLoss,
	}, nil
}
