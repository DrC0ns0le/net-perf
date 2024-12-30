package latency

import (
	"context"
	"fmt"
	"net"
	"time"
)

func (c *Client) MeasureTCP(ctx context.Context, targetPort int) (Result, error) {
	const attempts = 10
	latencies := make([]int64, 0, attempts)

	for i := 0; i < attempts; i++ {
		startTime := time.Now()

		dialer := &net.Dialer{
			LocalAddr: &net.TCPAddr{
				IP: c.SourceIP,
			},
			Timeout:   2 * time.Second,
			KeepAlive: 100 * time.Millisecond,
		}

		conn, err := dialer.DialContext(ctx, "tcp4", net.JoinHostPort(c.TargetIP.String(), fmt.Sprintf("%d", targetPort)))
		if err != nil {
			continue
		}

		latency := time.Since(startTime).Microseconds()
		latencies = append(latencies, latency)

		conn.Close()
	}

	// Calculate packet loss
	packetLoss := float64(attempts-len(latencies)) / float64(attempts) * 100

	// If all attempts failed, return error
	if len(latencies) == 0 {
		return Result{
			Status:   0,
			Protocol: "tcp",
		}, fmt.Errorf("all connection attempts failed")
	}

	// Calculate average latency
	var totalLatency int64
	for _, latency := range latencies {
		totalLatency += latency
	}
	avgLatency := totalLatency / int64(len(latencies))

	// Calculate jitter (mean deviation)
	var totalDeviation int64
	for _, latency := range latencies {
		deviation := latency - avgLatency
		if deviation < 0 {
			deviation = -deviation // Get absolute value
		}
		totalDeviation += deviation
	}
	jitter := totalDeviation / int64(len(latencies))

	return Result{
		Status:     1,
		Protocol:   "tcp",
		AvgLatency: avgLatency,
		Jitter:     jitter,
		Loss:       packetLoss,
	}, nil
}
