package finder

import (
	"context"
	"fmt"
	"strconv"

	"github.com/DrC0ns0le/net-perf/internal/metrics"
)

type networkPath struct {
	source int
	target int
}

func GetAllPaths(ctx context.Context) ([]networkPath, error) {

	response, err := metrics.QueryRange(ctx, "now-1d", "now", "1d", "avg(network_latency_status) by (source, target)")
	if err != nil {
		return nil, err
	}

	var paths []networkPath
	for _, result := range response.Data.Result {
		src, err := strconv.Atoi(result.Metric.Source)
		if err != nil {
			return nil, fmt.Errorf("error parsing source: %w", err)
		}
		tgt, err := strconv.Atoi(result.Metric.Target)
		if err != nil {
			return nil, fmt.Errorf("error parsing target: %w", err)
		}
		paths = append(paths, networkPath{
			source: src,
			target: tgt,
		})
	}
	return paths, nil
}
