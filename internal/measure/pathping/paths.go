package pathping

import (
	"context"
	"fmt"
	"strconv"

	"github.com/DrC0ns0le/net-perf/internal/metrics"
)

func GetAllPeers(ctx context.Context) ([]int, error) {
	response, err := metrics.QueryRange(ctx, "now-1d", "now", "1d", "avg(network_latency_status) by (source)")
	if err != nil {
		return nil, err
	}

	var peers []int
	for _, result := range response.Data.Result {
		src, err := strconv.Atoi(result.Metric.Source)
		if err != nil {
			return nil, fmt.Errorf("error parsing source: %w", err)
		}
		peers = append(peers, src)
	}
	return peers, nil
}
