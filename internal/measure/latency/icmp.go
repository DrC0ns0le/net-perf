package latency

import (
	"context"
	"time"

	probing "github.com/prometheus-community/pro-bing"
)

// Measures ICMP connection latency
func (c *Client) MeasureICMP(ctx context.Context) (Result, error) {
	pinger, err := probing.NewPinger(c.TargetIP.String())
	if err != nil {
		return Result{Status: 0, Protocol: "icmp"}, err
	}
	pinger.Source = c.SourceIP.String()
	pinger.SetPrivileged(true)
	pinger.Interval = 250 * time.Millisecond
	pinger.Timeout = 2 * time.Second
	pinger.Count = 10
	err = pinger.RunWithContext(ctx) // Blocks until finished.
	if err != nil {
		return Result{Status: 0, Protocol: "icmp"}, err
	}

	if pinger.Statistics().PacketLoss == 100 {
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
