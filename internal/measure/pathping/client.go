package pathping

import (
	"fmt"
	"time"
)

type Client struct {
	Count    int
	Interval time.Duration
}

func NewClient(count int, interval time.Duration) *Client {
	return &Client{
		Count:    count,
		Interval: interval,
	}
}

func (c *Client) Measure(path []int) (Result, error) {
	if pathPingServer == nil {
		return Result{}, fmt.Errorf("pathping server not started")
	}
	return pathPingServer.Measure(func(path []int) []uint16 {
		p := make([]uint16, len(path))
		for i, v := range path {
			p[i] = uint16(v)
		}
		return p
	}(path), MeasurementOptions{
		Count:    c.Count,
		Interval: c.Interval,
	})
}
