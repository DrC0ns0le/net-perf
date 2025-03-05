package finder

import (
	"context"
	"fmt"
	"strconv"

	"github.com/DrC0ns0le/net-perf/internal/metrics"
)

type NetworkPath struct {
	Source int
	Target int
}

func GetAllPaths(ctx context.Context) ([]NetworkPath, error) {

	response, err := metrics.QueryRange(ctx, "now-15m", "now", "15m", "avg(network_latency_status) by (source, target)")
	if err != nil {
		return nil, err
	}

	var paths []NetworkPath
	for _, result := range response.Data.Result {
		src, err := strconv.Atoi(result.Metric.Source)
		if err != nil {
			return nil, fmt.Errorf("error parsing source: %w", err)
		}
		tgt, err := strconv.Atoi(result.Metric.Target)
		if err != nil {
			return nil, fmt.Errorf("error parsing target: %w", err)
		}
		paths = append(paths, NetworkPath{
			Source: src,
			Target: tgt,
		})
	}
	return paths, nil
}
