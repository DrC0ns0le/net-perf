package metrics

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"
)

type PathMetrics struct {
	Availability float64 // in percentage (0-1)
	Latency      float64 // in microseconds
	Jitter       float64 // in microseconds
	PacketLoss   float64 // in percentage (0-100)
	Cost         float64
}

func GetPingPathLatency(ctx context.Context, origin, remote int) (float64, error) {
	response, err := QueryRange(ctx, "now-1m", "now", "1m", fmt.Sprintf(`avg(avg_over_time(network_path_latency_duration{path=~"%s"}[1m]))`, GetPathLabel(origin, remote)))
	if err != nil {
		return 0, err
	}

	if len(response.Data.Result) == 0 {
		return 0, ErrNoPaths
	}

	latency, err := strconv.ParseFloat(response.Data.Result[0].Values[0][1].(string), 64)
	if err != nil {
		return 0, err
	}

	return latency, nil

}

// GetPreferredPath returns the preferred IP version to use for communication with the given remote endpoint
// along with the associated cost. If no valid paths are available, it returns an empty string and an error.
func GetPreferredPath(ctx context.Context, origin, remote int) (string, float64, error) {
	metrics, err := GetPathMetrics(ctx, origin, remote)
	if err != nil {
		return "", 0, err
	}

	var bestVersion string
	var bestCost float64 = math.Inf(1)

	for version, path := range metrics {
		if !math.IsInf(path.Cost, 1) && (math.IsInf(bestCost, 1) || path.Cost < bestCost) {
			bestVersion = version
			bestCost = path.Cost
		}
	}

	if bestVersion == "" {
		return "", 0, ErrNoPaths
	}

	return bestVersion, bestCost, nil
}

// GetPathMetrics queries prometheus for the average latency, packet loss, and availability for both
// IPv4 and IPv6 paths to the given remote. It returns a map where the keys are the
// IP versions and the values are PathMetrics objects.
//
// The metrics queried are:
//   - network_latency_duration: average latency in microseconds
//   - network_latency_loss and network_bandwidth_packet_loss: average packet loss
//     as a percentage.
//   - network_latency_status: average availability as a percentage.
//
// The values are averaged over a 1 minute time window.
func GetPathMetrics(ctx context.Context, origin, remote int) (map[string]*PathMetrics, error) {

	var metrics map[string]*PathMetrics

	// assumes dual-stack
	path := GetPathLabel(origin, remote)
	if path == "" {
		return metrics, errors.New("could not generate path label")
	}
	queries := []string{
		fmt.Sprintf(`avg(avg_over_time(network_latency_duration{path=~"%s"}[1m])) by (version)`, path),
		fmt.Sprintf(`avg(avg_over_time(network_latency_loss{path=~"%s"}[3m]),avg_over_time(network_bandwidth_packet_loss{path=~"%s"}[3m])) by (version)`, path, path),
		fmt.Sprintf(`avg(avg_over_time(network_latency_status{path=~"%s"}[1m])) by (version)`, path),
	}

	metrics = map[string]*PathMetrics{
		"4": {},
		"6": {},
	}

	for i, query := range queries {
		response, err := QueryRange(ctx, "now-1m", "now", "1m", query)
		if err != nil {
			return metrics, err
		}

		for _, result := range response.Data.Result {
			switch i {
			case 0:
				latency, err := strconv.ParseFloat(result.Values[0][1].(string), 64)
				if err != nil {
					return nil, fmt.Errorf("error parsing latency: %w", err)
				}
				metrics[result.Metric.Version].Latency = latency
			case 1:
				loss, err := strconv.ParseFloat(result.Values[0][1].(string), 64)
				if err != nil {
					return nil, fmt.Errorf("error parsing packet loss: %w", err)
				}
				metrics[result.Metric.Version].PacketLoss = loss
			case 2:
				availability, err := strconv.ParseFloat(result.Values[0][1].(string), 64)
				if err != nil {
					return nil, fmt.Errorf("error parsing availability: %w", err)
				}
				metrics[result.Metric.Version].Availability = availability
			}
		}
	}

	for _, metric := range metrics {
		metric.calculatePathCost()
	}

	return metrics, nil
}

// GetPathLabel generates a label string for a path between two nodes. The string is
// in the format "X-Y", where X and Y are the node IDs. The IDs are sorted so that
// the smaller one is first, to ensure that the same path is always labelled with
// the same string, regardless of the order in which the nodes are given. If the
// IDs can't be parsed as integers, an empty string is returned.
func GetPathLabel(origin, remote int) string {
	if origin > remote {
		origin, remote = remote, origin
	}

	return strconv.Itoa(origin) + "-" + strconv.Itoa(remote)
}

// calculatePathCost calculates a cost for a path based on its latency, packet loss and jitter metrics.
// The cost calculation is based on the Mathis equation which models TCP throughput in relation to
// RTT and packet loss. Availability is treated as an additional loss factor, and jitter is included
// for path stability assessment.
func (pm *PathMetrics) calculatePathCost() {
	if pm == nil || pm.Availability == 0 || pm.Latency == 0 {
		pm.Cost = math.Inf(1)
		return
	}

	const (
		K1 = 1.0 // Throughput weight
		K2 = 0.5 // Jitter weight
		C  = 1e4 // Mathis equation constant
	)

	rttMs := pm.Latency / 1e3
	jitterMs := pm.Jitter / 1e3

	effectiveLoss := (1.0 - pm.Availability) * (pm.PacketLoss / 100)

	if effectiveLoss >= 1 {
		pm.Cost = math.Inf(1)
		return
	}

	var mathisCost float64
	if effectiveLoss > 0 {
		mathisCost = rttMs * C * math.Sqrt(effectiveLoss)
	}

	pm.Cost = (K1*(rttMs+mathisCost) + K2*jitterMs)
}
